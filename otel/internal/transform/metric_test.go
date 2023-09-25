package transform_test

import (
	"testing"
	"time"

	"github.com/DimmyJing/valise/otel/internal/transform"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

type metricBuilder struct {
	pMetric       pmetric.Metrics
	pScopeMetrics pmetric.ScopeMetricsSlice
	pMetrics      pmetric.MetricSlice

	resourceMetrics *metricdata.ResourceMetrics
}

func newMetricBuilder(resSchemaURL string, resAttrs []attribute.KeyValue) *metricBuilder {
	pMetric := pmetric.NewMetrics()

	//nolint:exhaustruct
	builder := &metricBuilder{
		pMetric: pMetric,

		resourceMetrics: &metricdata.ResourceMetrics{
			Resource:     resource.NewWithAttributes(resSchemaURL, resAttrs...),
			ScopeMetrics: []metricdata.ScopeMetrics{},
		},
	}

	pResourceMetric := pMetric.ResourceMetrics().AppendEmpty()
	transform.TransformKeyValues(resAttrs, pResourceMetric.Resource().Attributes())
	pResourceMetric.SetSchemaUrl(resSchemaURL)
	builder.pScopeMetrics = pResourceMetric.ScopeMetrics()

	return builder
}

func (b *metricBuilder) addScope(
	scopeName string,
	scopeVersion string,
	scopeSchemaURL string,
) {
	b.resourceMetrics.ScopeMetrics = append(
		b.resourceMetrics.ScopeMetrics,
		metricdata.ScopeMetrics{
			Scope: instrumentation.Scope{
				Name:      scopeName,
				Version:   scopeVersion,
				SchemaURL: scopeSchemaURL,
			},
			Metrics: nil,
		},
	)
	pScopeMetric := b.pScopeMetrics.AppendEmpty()
	pScope := pScopeMetric.Scope()
	pScope.SetName(scopeName)
	pScope.SetVersion(scopeVersion)
	pScopeMetric.SetSchemaUrl(scopeSchemaURL)
	b.pMetrics = pScopeMetric.Metrics()
}

func (b *metricBuilder) addMetric(
	name string,
	description string,
	unit string,
	data metricdata.Aggregation,
	pData pmetric.Metric,
) {
	metrics := metricdata.Metrics{
		Name:        name,
		Description: description,
		Unit:        unit,
		Data:        data,
	}

	pMetrics := b.pMetrics.AppendEmpty()
	pData.CopyTo(pMetrics)
	pMetrics.SetName(name)
	pMetrics.SetDescription(description)
	pMetrics.SetUnit(unit)

	scopeMetrics := &b.resourceMetrics.ScopeMetrics[len(b.resourceMetrics.ScopeMetrics)-1]
	scopeMetrics.Metrics = append(scopeMetrics.Metrics, metrics)
}

func (b *metricBuilder) compare(t *testing.T, expErr bool) {
	t.Helper()

	want := b.pMetric
	has, err := transform.ResourceMetrics(b.resourceMetrics)

	if expErr {
		assert.Error(t, err)
	} else {
		assert.Equal(t, want, has)
	}
}

func getGauge[N int64 | float64](
	attributes []attribute.Set,
	startTimes []time.Time,
	times []time.Time,
	values []N,
	exemplars [][]metricdata.Exemplar[N],
	pExemplars []pmetric.ExemplarSlice,
) (metricdata.Gauge[N], pmetric.Metric) {
	datapoints := make([]metricdata.DataPoint[N], len(attributes))
	pMetric := pmetric.NewMetric()
	pMetric.SetEmptyGauge()
	pGauge := pMetric.Gauge()
	pDataPoints := pGauge.DataPoints()

	for idx := range attributes {
		pDataPoint := pDataPoints.AppendEmpty()
		transform.TransformKeyValues(attributes[idx].ToSlice(), pDataPoint.Attributes())
		pDataPoint.SetStartTimestamp(pcommon.NewTimestampFromTime(startTimes[idx]))
		pDataPoint.SetTimestamp(pcommon.NewTimestampFromTime(times[idx]))

		switch v := any(values[idx]).(type) {
		case int64:
			pDataPoint.SetIntValue(v)
		case float64:
			pDataPoint.SetDoubleValue(v)
		}
		pExemplars[idx].CopyTo(pDataPoint.Exemplars())
		datapoints[idx] = metricdata.DataPoint[N]{
			Attributes: attributes[idx],
			StartTime:  startTimes[idx],
			Time:       times[idx],
			Value:      values[idx],
			Exemplars:  exemplars[idx],
		}
	}

	aggregation := metricdata.Gauge[N]{DataPoints: datapoints}

	return aggregation, pMetric
}

func getSum[N int64 | float64](
	temporality metricdata.Temporality,
	pTemporality pmetric.AggregationTemporality,
	isMonotonic bool,
	attributes []attribute.Set,
	startTimes []time.Time,
	times []time.Time,
	values []N,
	exemplars [][]metricdata.Exemplar[N],
	pExemplars []pmetric.ExemplarSlice,
) (metricdata.Sum[N], pmetric.Metric) {
	datapoints := make([]metricdata.DataPoint[N], len(attributes))
	pMetric := pmetric.NewMetric()
	pMetric.SetEmptySum()
	pSum := pMetric.Sum()
	pSum.SetAggregationTemporality(pTemporality)
	pSum.SetIsMonotonic(isMonotonic)
	pDataPoints := pSum.DataPoints()

	for idx := range attributes {
		pDataPoint := pDataPoints.AppendEmpty()
		transform.TransformKeyValues(attributes[idx].ToSlice(), pDataPoint.Attributes())
		pDataPoint.SetStartTimestamp(pcommon.NewTimestampFromTime(startTimes[idx]))
		pDataPoint.SetTimestamp(pcommon.NewTimestampFromTime(times[idx]))

		switch v := any(values[idx]).(type) {
		case int64:
			pDataPoint.SetIntValue(v)
		case float64:
			pDataPoint.SetDoubleValue(v)
		}
		pExemplars[idx].CopyTo(pDataPoint.Exemplars())
		datapoints[idx] = metricdata.DataPoint[N]{
			Attributes: attributes[idx],
			StartTime:  startTimes[idx],
			Time:       times[idx],
			Value:      values[idx],
			Exemplars:  exemplars[idx],
		}
	}

	aggregation := metricdata.Sum[N]{
		DataPoints:  datapoints,
		Temporality: temporality,
		IsMonotonic: isMonotonic,
	}

	return aggregation, pMetric
}

func getHistogram[N int64 | float64]( //nolint:funlen
	temporality metricdata.Temporality,
	pTemporality pmetric.AggregationTemporality,
	attributes []attribute.Set,
	startTimes []time.Time,
	times []time.Time,
	counts []uint64,
	sums []N,
	bucketCounts [][]uint64,
	explicitBounds [][]float64,
	mins []*N,
	maxes []*N,
	exemplars [][]metricdata.Exemplar[N],
	pExemplars []pmetric.ExemplarSlice,
) (metricdata.Histogram[N], pmetric.Metric) {
	datapoints := make([]metricdata.HistogramDataPoint[N], len(attributes))
	pMetric := pmetric.NewMetric()
	pMetric.SetEmptyHistogram()
	pHistogram := pMetric.Histogram()
	pHistogram.SetAggregationTemporality(pTemporality)
	pDataPoints := pHistogram.DataPoints()

	for idx := range attributes {
		pDataPoint := pDataPoints.AppendEmpty()
		transform.TransformKeyValues(attributes[idx].ToSlice(), pDataPoint.Attributes())
		pDataPoint.SetStartTimestamp(pcommon.NewTimestampFromTime(startTimes[idx]))
		pDataPoint.SetTimestamp(pcommon.NewTimestampFromTime(times[idx]))
		pDataPoint.SetCount(counts[idx])
		pDataPoint.SetSum(float64(sums[idx]))
		pDataPoint.BucketCounts().FromRaw(bucketCounts[idx])
		pDataPoint.ExplicitBounds().FromRaw(explicitBounds[idx])

		if mins[idx] != nil {
			pDataPoint.SetMin(float64(*mins[idx]))
		}

		if maxes[idx] != nil {
			pDataPoint.SetMax(float64(*maxes[idx]))
		}

		pExemplars[idx].CopyTo(pDataPoint.Exemplars())
		datapoints[idx] = metricdata.HistogramDataPoint[N]{
			Attributes:   attributes[idx],
			StartTime:    startTimes[idx],
			Time:         times[idx],
			Count:        counts[idx],
			Bounds:       explicitBounds[idx],
			BucketCounts: bucketCounts[idx],
			Min:          metricdata.Extrema[N]{},
			Max:          metricdata.Extrema[N]{},
			Sum:          sums[idx],
			Exemplars:    exemplars[idx],
		}

		if mins[idx] != nil {
			datapoints[idx].Min = metricdata.NewExtrema[N](*mins[idx])
		}

		if maxes[idx] != nil {
			datapoints[idx].Max = metricdata.NewExtrema[N](*maxes[idx])
		}
	}

	aggregation := metricdata.Histogram[N]{
		DataPoints:  datapoints,
		Temporality: temporality,
	}

	return aggregation, pMetric
}

func getExponentialHistogram[N int64 | float64]( //nolint:funlen
	temporality metricdata.Temporality,
	pTemporality pmetric.AggregationTemporality,
	attributes []attribute.Set,
	startTimes []time.Time,
	times []time.Time,
	counts []uint64,
	sums []N,
	scales []int32,
	zeroCounts []uint64,
	positiveOffsets []int32,
	positiveBucketCounts [][]uint64,
	negativeOffsets []int32,
	negativeBucketCounts [][]uint64,
	mins []*N,
	maxes []*N,
	exemplars [][]metricdata.Exemplar[N],
	pExemplars []pmetric.ExemplarSlice,
) (metricdata.ExponentialHistogram[N], pmetric.Metric) {
	datapoints := make([]metricdata.ExponentialHistogramDataPoint[N], len(attributes))
	pMetric := pmetric.NewMetric()
	pMetric.SetEmptyExponentialHistogram()
	pHistogram := pMetric.ExponentialHistogram()
	pHistogram.SetAggregationTemporality(pTemporality)
	pDataPoints := pHistogram.DataPoints()

	for idx := range attributes {
		pDataPoint := pDataPoints.AppendEmpty()
		transform.TransformKeyValues(attributes[idx].ToSlice(), pDataPoint.Attributes())
		pDataPoint.SetStartTimestamp(pcommon.NewTimestampFromTime(startTimes[idx]))
		pDataPoint.SetTimestamp(pcommon.NewTimestampFromTime(times[idx]))
		pDataPoint.SetCount(counts[idx])
		pDataPoint.SetSum(float64(sums[idx]))

		pDataPoint.SetScale(scales[idx])
		pDataPoint.SetZeroCount(zeroCounts[idx])
		pos := pDataPoint.Positive()
		pos.SetOffset(positiveOffsets[idx])
		pos.BucketCounts().FromRaw(positiveBucketCounts[idx])

		neg := pDataPoint.Negative()
		neg.SetOffset(negativeOffsets[idx])
		neg.BucketCounts().FromRaw(negativeBucketCounts[idx])

		if mins[idx] != nil {
			pDataPoint.SetMin(float64(*mins[idx]))
		}

		if maxes[idx] != nil {
			pDataPoint.SetMax(float64(*maxes[idx]))
		}

		pExemplars[idx].CopyTo(pDataPoint.Exemplars())
		datapoints[idx] = metricdata.ExponentialHistogramDataPoint[N]{
			Attributes: attributes[idx],
			StartTime:  startTimes[idx],
			Time:       times[idx],
			Count:      counts[idx],
			Min:        metricdata.Extrema[N]{},
			Max:        metricdata.Extrema[N]{},
			Sum:        sums[idx],
			Scale:      scales[idx],
			ZeroCount:  zeroCounts[idx],
			PositiveBucket: metricdata.ExponentialBucket{
				Offset: positiveOffsets[idx],
				Counts: positiveBucketCounts[idx],
			},
			NegativeBucket: metricdata.ExponentialBucket{
				Offset: negativeOffsets[idx],
				Counts: negativeBucketCounts[idx],
			},
			ZeroThreshold: 0,
			Exemplars:     exemplars[idx],
		}

		if mins[idx] != nil {
			datapoints[idx].Min = metricdata.NewExtrema[N](*mins[idx])
		}

		if maxes[idx] != nil {
			datapoints[idx].Max = metricdata.NewExtrema[N](*maxes[idx])
		}
	}

	aggregation := metricdata.ExponentialHistogram[N]{
		DataPoints:  datapoints,
		Temporality: temporality,
	}

	return aggregation, pMetric
}

func getExemplars[N int64 | float64](
	attributes [][][]attribute.KeyValue,
	times [][]time.Time,
	values [][]N,
	spanIDs [][]pcommon.SpanID,
	traceIDs [][]pcommon.TraceID,
) ([][]metricdata.Exemplar[N], []pmetric.ExemplarSlice) {
	resExemplars := [][]metricdata.Exemplar[N]{}
	resPExemplars := []pmetric.ExemplarSlice{}

	for exemplarIdx := range attributes {
		exemplars := []metricdata.Exemplar[N]{}
		pExemplars := pmetric.NewExemplarSlice()
		pExemplars.EnsureCapacity(len(attributes[exemplarIdx]))

		for idx := range attributes[exemplarIdx] {
			pExemplar := pExemplars.AppendEmpty()

			exemplars = append(exemplars, metricdata.Exemplar[N]{
				FilteredAttributes: attributes[exemplarIdx][idx],
				Time:               times[exemplarIdx][idx],
				Value:              values[exemplarIdx][idx],
				SpanID:             spanIDs[exemplarIdx][idx][:],
				TraceID:            traceIDs[exemplarIdx][idx][:],
			})

			transform.TransformKeyValues(
				attributes[exemplarIdx][idx],
				pExemplar.FilteredAttributes(),
			)
			pExemplar.SetTimestamp(pcommon.NewTimestampFromTime(times[exemplarIdx][idx]))

			switch v := any(values[exemplarIdx][idx]).(type) {
			case int64:
				pExemplar.SetIntValue(v)
			case float64:
				pExemplar.SetDoubleValue(v)
			}
			pExemplar.SetSpanID(spanIDs[exemplarIdx][idx])
			pExemplar.SetTraceID(traceIDs[exemplarIdx][idx])
		}

		resExemplars = append(resExemplars, exemplars)
		resPExemplars = append(resPExemplars, pExemplars)
	}

	return resExemplars, resPExemplars
}

func TestMetricNil(t *testing.T) {
	t.Parallel()

	res, err := transform.ResourceMetrics(nil)
	assert.NoError(t, err)
	assert.Equal(t, pmetric.NewMetrics(), res)
}

func TestGauge(t *testing.T) { //nolint:funlen
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	aggregation, pMetric := getGauge[int64](
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]int64{1, 2},
		[][]metricdata.Exemplar[int64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)
	builder.addScope("scopename2", "scopeversion2", "scopeschema2")

	aggregation2, pMetric2 := getGauge[float64](
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("5", "5"),
				attribute.String("6", "6"),
			),
			attribute.NewSet(
				attribute.String("7", "7"),
				attribute.String("8", "8"),
			),
		},
		[]time.Time{
			time.Unix(4, 4),
			time.Unix(5, 5),
		},
		[]time.Time{
			time.Unix(6, 6),
			time.Unix(7, 7),
		},
		[]float64{3, 4},
		[][]metricdata.Exemplar[float64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname2",
		"metricdescription2",
		"metricunit2",
		aggregation2,
		pMetric2,
	)
	builder.compare(t, false)
}

func TestSum(t *testing.T) { //nolint:funlen
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	aggregation, pMetric := getSum[int64](
		metricdata.DeltaTemporality,
		pmetric.AggregationTemporalityDelta,
		false,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]int64{1, 2},
		[][]metricdata.Exemplar[int64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)

	aggregation2, pMetric2 := getSum[float64](
		metricdata.CumulativeTemporality,
		pmetric.AggregationTemporalityCumulative,
		true,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("5", "5"),
				attribute.String("6", "6"),
			),
			attribute.NewSet(
				attribute.String("7", "7"),
				attribute.String("8", "8"),
			),
		},
		[]time.Time{
			time.Unix(4, 4),
			time.Unix(5, 5),
		},
		[]time.Time{
			time.Unix(6, 6),
			time.Unix(7, 7),
		},
		[]float64{3, 4},
		[][]metricdata.Exemplar[float64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname2",
		"metricdescription2",
		"metricunit2",
		aggregation2,
		pMetric2,
	)
	builder.compare(t, false)
}

func TestSumUnknownTemporality(t *testing.T) {
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	aggregation, pMetric := getSum[int64](
		metricdata.Temporality(0),
		pmetric.AggregationTemporalityUnspecified,
		false,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]int64{1, 2},
		[][]metricdata.Exemplar[int64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)
	builder.compare(t, true)
}

func TestHistogram(t *testing.T) { //nolint:funlen
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	min1 := int64(13)
	max1 := int64(14)
	aggregation, pMetric := getHistogram[int64](
		metricdata.DeltaTemporality,
		pmetric.AggregationTemporalityDelta,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]uint64{1, 2},
		[]int64{3, 4},
		[][]uint64{
			{5, 6},
			{7, 8},
		},
		[][]float64{
			{9.5, 10.5},
			{11.5, 12.5},
		},
		[]*int64{&min1, nil},
		[]*int64{&max1, nil},
		[][]metricdata.Exemplar[int64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)

	min2 := float64(29.5)
	max2 := float64(30.5)
	aggregation2, pMetric2 := getHistogram[float64](
		metricdata.CumulativeTemporality,
		pmetric.AggregationTemporalityCumulative,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("5", "5"),
				attribute.String("6", "6"),
			),
			attribute.NewSet(
				attribute.String("7", "7"),
				attribute.String("8", "8"),
			),
		},
		[]time.Time{
			time.Unix(13, 13),
			time.Unix(14, 14),
		},
		[]time.Time{
			time.Unix(15, 15),
			time.Unix(16, 16),
		},
		[]uint64{17, 18},
		[]float64{19.5, 20.5},
		[][]uint64{
			{21, 22},
			{23, 24},
		},
		[][]float64{
			{25.5, 26.5},
			{27.5, 28.5},
		},
		[]*float64{&min2, nil},
		[]*float64{&max2, nil},
		[][]metricdata.Exemplar[float64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname2",
		"metricdescription2",
		"metricunit2",
		aggregation2,
		pMetric2,
	)
	builder.compare(t, false)
}

func TestHistogramError(t *testing.T) {
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	min1 := int64(13)
	max1 := int64(14)
	aggregation, pMetric := getHistogram[int64](
		metricdata.Temporality(0),
		pmetric.AggregationTemporalityDelta,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]uint64{1, 2},
		[]int64{3, 4},
		[][]uint64{
			{5, 6},
			{7, 8},
		},
		[][]float64{
			{9.5, 10.5},
			{11.5, 12.5},
		},
		[]*int64{&min1, nil},
		[]*int64{&max1, nil},
		[][]metricdata.Exemplar[int64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)
	builder.compare(t, true)
}

func TestExponentialHistogram(t *testing.T) { //nolint:funlen
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	min1 := int64(21)
	max1 := int64(22)
	aggregation, pMetric := getExponentialHistogram[int64](
		metricdata.DeltaTemporality,
		pmetric.AggregationTemporalityDelta,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]uint64{1, 2},
		[]int64{3, 4},
		[]int32{5, 6},
		[]uint64{7, 8},
		[]int32{9, 10},
		[][]uint64{
			{11, 12},
			{13, 14},
		},
		[]int32{15, 16},
		[][]uint64{
			{17, 18},
			{19, 20},
		},
		[]*int64{&min1, nil},
		[]*int64{&max1, nil},
		[][]metricdata.Exemplar[int64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)

	min2 := float64(47.5)
	max2 := float64(48.5)
	aggregation2, pMetric2 := getExponentialHistogram[float64](
		metricdata.CumulativeTemporality,
		pmetric.AggregationTemporalityCumulative,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("5", "5"),
				attribute.String("6", "6"),
			),
			attribute.NewSet(
				attribute.String("7", "7"),
				attribute.String("8", "8"),
			),
		},
		[]time.Time{
			time.Unix(23, 23),
			time.Unix(24, 24),
		},
		[]time.Time{
			time.Unix(25, 25),
			time.Unix(26, 26),
		},
		[]uint64{27, 28},
		[]float64{29.5, 30.5},
		[]int32{31, 32},
		[]uint64{33, 34},
		[]int32{35, 36},
		[][]uint64{
			{37, 38},
			{39, 40},
		},
		[]int32{41, 42},
		[][]uint64{
			{43, 44},
			{45, 46},
		},
		[]*float64{&min2, nil},
		[]*float64{&max2, nil},
		[][]metricdata.Exemplar[float64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname2",
		"metricdescription2",
		"metricunit2",
		aggregation2,
		pMetric2,
	)
	builder.compare(t, false)
}

func TestExponentialHistogramError(t *testing.T) {
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	min1 := int64(21)
	max1 := int64(22)
	aggregation, pMetric := getExponentialHistogram[int64](
		metricdata.Temporality(0),
		pmetric.AggregationTemporalityDelta,
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]uint64{1, 2},
		[]int64{3, 4},
		[]int32{5, 6},
		[]uint64{7, 8},
		[]int32{9, 10},
		[][]uint64{
			{11, 12},
			{13, 14},
		},
		[]int32{15, 16},
		[][]uint64{
			{17, 18},
			{19, 20},
		},
		[]*int64{&min1, nil},
		[]*int64{&max1, nil},
		[][]metricdata.Exemplar[int64]{nil, nil},
		[]pmetric.ExemplarSlice{pmetric.NewExemplarSlice(), pmetric.NewExemplarSlice()},
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)
	builder.compare(t, true)
}

func TestExemplars(t *testing.T) { //nolint:funlen
	t.Parallel()

	builder := newMetricBuilder("resourceschema1", []attribute.KeyValue{
		attribute.String("resourceattr1", "resourcevalue1"),
		attribute.String("resourceattr2", "resourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	exemplars, pExemplars := getExemplars[int64](
		[][][]attribute.KeyValue{
			{
				{
					attribute.String("1", "2"),
					attribute.String("3", "4"),
				},
				{
					attribute.String("5", "6"),
					attribute.String("7", "8"),
				},
			},
			{
				{
					attribute.String("9", "10"),
					attribute.String("11", "12"),
				},
				{
					attribute.String("13", "14"),
					attribute.String("15", "16"),
				},
			},
		},
		[][]time.Time{
			{
				time.Unix(17, 18),
				time.Unix(19, 20),
			},
			{
				time.Unix(21, 22),
				time.Unix(23, 24),
			},
		},
		[][]int64{
			{
				25,
				26,
			},
			{
				27,
				28,
			},
		},
		[][]pcommon.SpanID{
			{
				pcommon.SpanID(getSpanID(29)),
				pcommon.SpanID(getSpanID(30)),
			},
			{
				pcommon.SpanID(getSpanID(31)),
				pcommon.SpanID(getSpanID(32)),
			},
		},
		[][]pcommon.TraceID{
			{
				pcommon.TraceID(getTraceID(33)),
				pcommon.TraceID(getTraceID(34)),
			},
			{
				pcommon.TraceID(getTraceID(35)),
				pcommon.TraceID(getTraceID(36)),
			},
		},
	)
	aggregation, pMetric := getGauge[int64](
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("1", "1"),
				attribute.String("2", "2"),
			),
			attribute.NewSet(
				attribute.String("3", "3"),
				attribute.String("4", "4"),
			),
		},
		[]time.Time{
			time.Unix(0, 0),
			time.Unix(1, 1),
		},
		[]time.Time{
			time.Unix(2, 2),
			time.Unix(3, 3),
		},
		[]int64{1, 2},
		exemplars,
		pExemplars,
	)
	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		aggregation,
		pMetric,
	)

	exemplars2, pExemplars2 := getExemplars[float64](
		[][][]attribute.KeyValue{
			{
				{
					attribute.String("29", "30"),
					attribute.String("31", "32"),
				},
			},
			{
				{
					attribute.String("33", "34"),
					attribute.String("35", "36"),
				},
			},
		},
		[][]time.Time{
			{
				time.Unix(37, 38),
			},
			{
				time.Unix(39, 40),
			},
		},
		[][]float64{
			{
				41.5,
			},
			{
				42.5,
			},
		},
		[][]pcommon.SpanID{
			{
				pcommon.SpanID(getSpanID(43)),
			},
			{
				pcommon.SpanID(getSpanID(44)),
			},
		},
		[][]pcommon.TraceID{
			{
				pcommon.TraceID(getTraceID(45)),
			},
			{
				pcommon.TraceID(getTraceID(46)),
			},
		},
	)
	aggregation2, pMetric2 := getGauge[float64](
		[]attribute.Set{
			attribute.NewSet(
				attribute.String("5", "5"),
				attribute.String("6", "6"),
			),
			attribute.NewSet(
				attribute.String("7", "7"),
				attribute.String("8", "8"),
			),
		},
		[]time.Time{
			time.Unix(4, 4),
			time.Unix(5, 5),
		},
		[]time.Time{
			time.Unix(6, 6),
			time.Unix(7, 7),
		},
		[]float64{3, 4},
		exemplars2,
		pExemplars2,
	)
	builder.addMetric(
		"metricname2",
		"metricdescription2",
		"metricunit2",
		aggregation2,
		pMetric2,
	)
	builder.compare(t, false)
}

type unknownAggregation struct {
	metricdata.Aggregation
}

func TestInvalidAggregation(t *testing.T) {
	t.Parallel()

	builder := newMetricBuilder("invalidresourceschema1", []attribute.KeyValue{
		attribute.String("invalidresourceattr1", "invalidresourcevalue1"),
		attribute.String("invalidresourceattr2", "invalidresourcevalue2"),
	})
	builder.addScope("scopename1", "scopeversion1", "scopeschema1")

	builder.addMetric(
		"metricname1",
		"metricdescription1",
		"metricunit1",
		unknownAggregation{Aggregation: nil},
		pmetric.NewMetric(),
	)
	builder.compare(t, true)
}
