package transform

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/otlp/otlpmetric/otlpmetrichttp

var (
	errUnknownAggregation = errors.New("unknown aggregation")
	errUnknownTemporality = errors.New("unknown temporality")
)

func ResourceMetrics(resourceMetrics *metricdata.ResourceMetrics) (pmetric.Metrics, error) {
	metrics := pmetric.NewMetrics()
	if resourceMetrics == nil {
		return metrics, nil
	}

	pResourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	TransformKeyValues(
		resourceMetrics.Resource.Attributes(),
		pResourceMetrics.Resource().Attributes(),
	)

	err := transformScopeMetrics(resourceMetrics.ScopeMetrics, pResourceMetrics.ScopeMetrics())
	pResourceMetrics.SetSchemaUrl(resourceMetrics.Resource.SchemaURL())

	return metrics, err
}

func transformScopeMetrics(
	scopeMetrics []metricdata.ScopeMetrics,
	pScopeMetrics pmetric.ScopeMetricsSlice,
) error {
	pScopeMetrics.EnsureCapacity(len(scopeMetrics))

	for _, scopeMetric := range scopeMetrics {
		pScopeMetric := pScopeMetrics.AppendEmpty()

		err := transformMetrics(scopeMetric.Metrics, pScopeMetric.Metrics())
		if err != nil {
			return err
		}

		scope := pScopeMetric.Scope()
		scope.SetName(scopeMetric.Scope.Name)
		scope.SetVersion(scopeMetric.Scope.Version)
		pScopeMetric.SetSchemaUrl(scopeMetric.Scope.SchemaURL)
	}

	return nil
}

func transformMetrics(metrics []metricdata.Metrics, pMetrics pmetric.MetricSlice) error {
	pMetrics.EnsureCapacity(len(metrics))

	for _, m := range metrics {
		err := transformMetric(m, pMetrics.AppendEmpty())
		if err != nil {
			return err
		}
	}

	return nil
}

func transformMetric(metrics metricdata.Metrics, pMetrics pmetric.Metric) error {
	var err error

	pMetrics.SetName(metrics.Name)
	pMetrics.SetDescription(metrics.Description)
	pMetrics.SetUnit(metrics.Unit)

	switch data := metrics.Data.(type) {
	case metricdata.Gauge[int64]:
		transformGauge[int64](data, pMetrics.SetEmptyGauge())
	case metricdata.Gauge[float64]:
		transformGauge[float64](data, pMetrics.SetEmptyGauge())
	case metricdata.Sum[int64]:
		err = transformSum[int64](data, pMetrics.SetEmptySum())
	case metricdata.Sum[float64]:
		err = transformSum[float64](data, pMetrics.SetEmptySum())
	case metricdata.Histogram[int64]:
		err = transformHistogram[int64](data, pMetrics.SetEmptyHistogram())
	case metricdata.Histogram[float64]:
		err = transformHistogram[float64](data, pMetrics.SetEmptyHistogram())
	case metricdata.ExponentialHistogram[int64]:
		err = transformExponentialHistogram[int64](data, pMetrics.SetEmptyExponentialHistogram())
	case metricdata.ExponentialHistogram[float64]:
		err = transformExponentialHistogram[float64](data, pMetrics.SetEmptyExponentialHistogram())
	default:
		err = fmt.Errorf("%w: %T", errUnknownAggregation, data)
	}

	return err
}

func transformGauge[N int64 | float64](gauge metricdata.Gauge[N], pGauge pmetric.Gauge) {
	transformDataPoints(gauge.DataPoints, pGauge.DataPoints())
}

func transformSum[N int64 | float64](sum metricdata.Sum[N], pSum pmetric.Sum) error {
	t, err := temporality(sum.Temporality)
	if err != nil {
		return err
	}

	pSum.SetAggregationTemporality(t)
	pSum.SetIsMonotonic(sum.IsMonotonic)
	transformDataPoints(sum.DataPoints, pSum.DataPoints())

	return nil
}

func transformDataPoints[N int64 | float64](
	dataPoints []metricdata.DataPoint[N],
	pDataPoints pmetric.NumberDataPointSlice,
) {
	pDataPoints.EnsureCapacity(len(dataPoints))

	for _, dataPoint := range dataPoints {
		pDataPoint := pDataPoints.AppendEmpty()
		TransformKeyValues(dataPoint.Attributes.ToSlice(), pDataPoint.Attributes())
		pDataPoint.SetStartTimestamp(pcommon.NewTimestampFromTime(dataPoint.StartTime))
		pDataPoint.SetTimestamp(pcommon.NewTimestampFromTime(dataPoint.Time))
		transformExemplars[N](dataPoint.Exemplars, pDataPoint.Exemplars())

		switch v := any(dataPoint.Value).(type) {
		case int64:
			pDataPoint.SetIntValue(v)
		case float64:
			pDataPoint.SetDoubleValue(v)
		}
	}
}

func transformHistogram[N int64 | float64](
	histogram metricdata.Histogram[N],
	pHistogram pmetric.Histogram,
) error {
	t, err := temporality(histogram.Temporality)
	if err != nil {
		return err
	}

	pHistogram.SetAggregationTemporality(t)
	transformHistogramDataPoints[N](histogram.DataPoints, pHistogram.DataPoints())

	return nil
}

func transformHistogramDataPoints[N int64 | float64](
	dataPoints []metricdata.HistogramDataPoint[N],
	pDataPoints pmetric.HistogramDataPointSlice,
) {
	pDataPoints.EnsureCapacity(len(dataPoints))

	for _, dataPoint := range dataPoints {
		pDataPoint := pDataPoints.AppendEmpty()
		TransformKeyValues(dataPoint.Attributes.ToSlice(), pDataPoint.Attributes())
		pDataPoint.SetStartTimestamp(pcommon.NewTimestampFromTime(dataPoint.StartTime))
		pDataPoint.SetTimestamp(pcommon.NewTimestampFromTime(dataPoint.Time))
		pDataPoint.SetCount(dataPoint.Count)
		pDataPoint.SetSum(float64(dataPoint.Sum))
		pDataPoint.BucketCounts().FromRaw(dataPoint.BucketCounts)
		pDataPoint.ExplicitBounds().FromRaw(dataPoint.Bounds)
		transformExemplars[N](dataPoint.Exemplars, pDataPoint.Exemplars())

		if v, ok := dataPoint.Min.Value(); ok {
			pDataPoint.SetMin(float64(v))
		}

		if v, ok := dataPoint.Max.Value(); ok {
			pDataPoint.SetMax(float64(v))
		}
	}
}

func transformExponentialHistogram[N int64 | float64](
	histogram metricdata.ExponentialHistogram[N],
	pHistogram pmetric.ExponentialHistogram,
) error {
	temporality, err := temporality(histogram.Temporality)
	if err != nil {
		return err
	}

	pHistogram.SetAggregationTemporality(temporality)
	transformExponentialHistogramDataPoints[N](histogram.DataPoints, pHistogram.DataPoints())

	return nil
}

func transformExponentialHistogramDataPoints[N int64 | float64](
	dataPoints []metricdata.ExponentialHistogramDataPoint[N],
	pDataPoints pmetric.ExponentialHistogramDataPointSlice,
) {
	pDataPoints.EnsureCapacity(len(dataPoints))

	for _, dataPoint := range dataPoints {
		pDataPoint := pDataPoints.AppendEmpty()
		TransformKeyValues(dataPoint.Attributes.ToSlice(), pDataPoint.Attributes())
		pDataPoint.SetStartTimestamp(pcommon.NewTimestampFromTime(dataPoint.StartTime))
		pDataPoint.SetTimestamp(pcommon.NewTimestampFromTime(dataPoint.Time))
		pDataPoint.SetCount(dataPoint.Count)
		pDataPoint.SetSum(float64(dataPoint.Sum))
		pDataPoint.SetScale(dataPoint.Scale)
		pDataPoint.SetZeroCount(dataPoint.ZeroCount)
		transformExponentialHistogramDataPointBuckets(
			dataPoint.PositiveBucket,
			pDataPoint.Positive(),
		)
		transformExponentialHistogramDataPointBuckets(
			dataPoint.NegativeBucket,
			pDataPoint.Negative(),
		)
		transformExemplars[N](dataPoint.Exemplars, pDataPoint.Exemplars())

		if v, ok := dataPoint.Min.Value(); ok {
			pDataPoint.SetMin(float64(v))
		}

		if v, ok := dataPoint.Max.Value(); ok {
			pDataPoint.SetMax(float64(v))
		}
	}
}

func transformExponentialHistogramDataPointBuckets(
	bucket metricdata.ExponentialBucket,
	pBucket pmetric.ExponentialHistogramDataPointBuckets,
) {
	pBucket.SetOffset(bucket.Offset)
	pBucket.BucketCounts().FromRaw(bucket.Counts)
}

func temporality(temporality metricdata.Temporality) (pmetric.AggregationTemporality, error) {
	switch temporality {
	case metricdata.DeltaTemporality:
		return pmetric.AggregationTemporalityDelta, nil
	case metricdata.CumulativeTemporality:
		return pmetric.AggregationTemporalityCumulative, nil
	default:
		return pmetric.AggregationTemporalityUnspecified, fmt.Errorf(
			"%w: %s",
			errUnknownTemporality,
			temporality,
		)
	}
}

func transformExemplars[N int64 | float64](
	exemplars []metricdata.Exemplar[N],
	pExemplars pmetric.ExemplarSlice,
) {
	pExemplars.EnsureCapacity(len(exemplars))

	for _, exemplar := range exemplars {
		pExemplar := pExemplars.AppendEmpty()
		pExemplar.SetTimestamp(pcommon.NewTimestampFromTime(exemplar.Time))

		switch v := any(exemplar.Value).(type) {
		case int64:
			pExemplar.SetIntValue(v)
		case float64:
			pExemplar.SetDoubleValue(v)
		}
		TransformKeyValues(exemplar.FilteredAttributes, pExemplar.FilteredAttributes())
		pExemplar.SetTraceID(pcommon.TraceID(exemplar.TraceID))
		pExemplar.SetSpanID(pcommon.SpanID(exemplar.SpanID))
	}
}
