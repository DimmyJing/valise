package signoz

import (
	"context"
	"fmt"
	"sort"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheus"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheusremotewrite"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/collector/pdata/pmetric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const (
	namespace      = "default"
	nameStr        = "__name__"
	temporalityStr = "__temporality__"
	envStr         = "env"

	offset64      uint64 = 14695981039346656037
	prime64       uint64 = 1099511628211
	separatorByte byte   = 255
)

func hashAdd(hash uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime64
	}

	return hash
}

func hashAddByte(h uint64, b byte) uint64 {
	return (h ^ uint64(b)) * prime64
}

func fingerprint(labels []*prompb.Label) uint64 {
	if len(labels) == 0 {
		return offset64
	}

	sum := offset64
	for _, l := range labels {
		sum = hashAdd(sum, l.Name)
		sum = hashAddByte(sum, separatorByte)
		sum = hashAdd(sum, l.Value)
		sum = hashAddByte(sum, separatorByte)
	}

	return sum
}

func marshalLabels(labels []*prompb.Label, byt []byte) []byte {
	if len(labels) == 0 {
		return append(byt, '{', '}')
	}

	byt = append(byt, '{')
	for _, l := range labels {
		byt = append(byt, '"')
		byt = append(byt, l.Name...)
		byt = append(byt, '"', ':', '"')

		for _, cha := range []byte(l.Value) {
			switch cha {
			case '\\', '"':
				byt = append(byt, '\\', cha)
			case '\n':
				byt = append(byt, '\\', 'n')
			case '\r':
				byt = append(byt, '\\', 'r')
			case '\t':
				byt = append(byt, '\\', 't')
			default:
				byt = append(byt, cha)
			}
		}

		byt = append(byt, '"', ',')
	}

	byt[len(byt)-1] = '}'

	return byt
}

func (s *signozExporter) convertMetrics( //nolint:funlen,gocognit,cyclop,gocyclo,maintidx
	ctx context.Context,
	metrics pmetric.Metrics,
) error {
	metricTemporality := map[string]pmetric.AggregationTemporality{}

	resourceMetricsSlice := metrics.ResourceMetrics()
	for i := 0; i < resourceMetricsSlice.Len(); i++ {
		resourceMetrics := resourceMetricsSlice.At(i)

		scopeMetricsSlice := resourceMetrics.ScopeMetrics()
		for j := 0; j < scopeMetricsSlice.Len(); j++ {
			scopeMetrics := scopeMetricsSlice.At(j)

			metricSlice := scopeMetrics.Metrics()
			for k := 0; k < metricSlice.Len(); k++ {
				metric := metricSlice.At(k)
				//nolint:exhaustive
				switch metric.Type() {
				case pmetric.MetricTypeSum:
					metricTemporality[namespace+"_"+metric.Name()] = metric.Sum().AggregationTemporality()
				case pmetric.MetricTypeHistogram:
					metricTemporality[namespace+"_"+metric.Name()] = metric.Histogram().AggregationTemporality()
				case pmetric.MetricTypeExponentialHistogram:
					metricTemporality[namespace+"_"+metric.Name()] = metric.ExponentialHistogram().AggregationTemporality()
				default:
					metricTemporality[namespace+"_"+metric.Name()] = pmetric.AggregationTemporalityUnspecified
				}
			}
		}
	}

	//nolint:exhaustruct
	tsMap, err := prometheusremotewrite.FromMetrics(metrics, prometheusremotewrite.Settings{Namespace: namespace})
	if err != nil {
		return fmt.Errorf("failed to convert metrics to prometheus format: %w", err)
	}

	fingerprints := []uint64{}
	timeSeries := make(map[uint64][]*prompb.Label, len(tsMap))
	fingerprintToName := make(map[uint64]map[string]string)
	timeSeriesList := make([]*prompb.TimeSeries, 0, len(tsMap))

	for _, timeSeriesInstance := range tsMap {
		timeSeriesList = append(timeSeriesList, timeSeriesInstance)

		var metricName string

		env := "default"

		labelsOverridden := make(map[string]*prompb.Label)
		for _, label := range timeSeriesInstance.Labels {
			//nolint:exhaustruct
			labelsOverridden[label.Name] = &prompb.Label{
				Name:  label.Name,
				Value: label.Value,
			}

			if label.Name == nameStr {
				metricName = label.Value
			}

			if label.Name == string(semconv.DeploymentEnvironmentKey) ||
				label.Name == prometheus.NormalizeLabel(string(semconv.DeploymentEnvironmentKey)) {
				env = label.Value
			}
		}

		var labels []*prompb.Label

		for _, l := range labelsOverridden {
			labels = append(labels, l)
		}

		if metricName != "" {
			if t, ok := metricTemporality[metricName]; ok {
				//nolint:exhaustruct
				labels = append(labels, &prompb.Label{
					Name:  temporalityStr,
					Value: t.String(),
				})
			}
		}

		sort.Slice(labels, func(i, j int) bool { return labels[i].Name < labels[j].Name })
		fingerprint := fingerprint(labels)
		fingerprints = append(fingerprints, fingerprint)
		timeSeries[fingerprint] = labels

		if _, ok := fingerprintToName[fingerprint]; !ok {
			fingerprintToName[fingerprint] = make(map[string]string)
		}

		fingerprintToName[fingerprint][nameStr] = metricName
		fingerprintToName[fingerprint][env] = env
	}

	newTimeSeries := make(map[uint64][]*prompb.Label)

	for fingerprint, labels := range timeSeries {
		_, ok := s.timeSeriesCache[fingerprint]
		if !ok {
			s.timeSeriesCache[fingerprint] = struct{}{}
			newTimeSeries[fingerprint] = labels
		}
	}

	err = func() error {
		statement, err := s.conn.PrepareBatch(
			ctx,
			fmt.Sprintf(
				"INSERT INTO %s.%s (metric_name, temporality, timestamp_ms, fingerprint, labels) VALUES (?, ?, ?, ?)",
				metricsDB,
				timeSeriesTable,
			),
		)
		if err != nil {
			return fmt.Errorf("failed preparing statement for time series: %w", err)
		}

		timestamp := model.Now().Time().UnixMilli()

		for fingerprint, labels := range newTimeSeries {
			//nolint:gomnd
			encodedLabels := string(marshalLabels(labels, make([]byte, 0, 128)))

			err = statement.Append(
				fingerprintToName[fingerprint][nameStr],
				metricTemporality[fingerprintToName[fingerprint][nameStr]].String(),
				timestamp,
				fingerprint,
				encodedLabels,
			)
			if err != nil {
				return fmt.Errorf("failed appending timeSeries statement: %w", err)
			}
		}

		err = statement.Send()
		if err != nil {
			return fmt.Errorf("failed sending timeSeries statement: %w", err)
		}

		return nil
	}()

	if err != nil {
		return fmt.Errorf("failed to write time series: %w", err)
	}

	err = func() error {
		ctx := context.Background()

		statement, err := s.conn.PrepareBatch(
			ctx,
			fmt.Sprintf(
				"INSERT INTO %s.%s (env, temporality, metric_name, fingerprint, timestamp_ms, labels) VALUES (?, ?, ?, ?, ?, ?)",
				metricsDB,
				timeSeriesTableV3,
			),
		)
		if err != nil {
			return fmt.Errorf("failed preparing statement for time series v3: %w", err)
		}

		timestamp := model.Now().Time().UnixMilli()

		for fingerprint, labels := range newTimeSeries {
			//nolint:gomnd
			encodedLabels := string(marshalLabels(labels, make([]byte, 0, 128)))

			err = statement.Append(
				fingerprintToName[fingerprint][envStr],
				metricTemporality[fingerprintToName[fingerprint][nameStr]].String(),
				fingerprintToName[fingerprint][nameStr],
				fingerprint,
				timestamp,
				encodedLabels,
			)
			if err != nil {
				return fmt.Errorf("failed appending timeSeries v3 statement: %w", err)
			}
		}

		err = statement.Send()
		if err != nil {
			return fmt.Errorf("failed sending timeSeries v3 statement: %w", err)
		}

		return nil
	}()

	if err != nil {
		return fmt.Errorf("failed to write time series v3: %w", err)
	}

	err = func() error {
		statement, err := s.conn.PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s.%s", metricsDB, samplesTable))
		if err != nil {
			return fmt.Errorf("failed preparing statement for samples table: %w", err)
		}

		for idx, timeSeriesInstance := range timeSeriesList {
			fingerprint := fingerprints[idx]
			for _, s := range timeSeriesInstance.Samples {
				err = statement.Append(
					fingerprintToName[fingerprint][nameStr],
					fingerprint,
					s.Timestamp,
					s.Value,
				)
				if err != nil {
					return fmt.Errorf("failed appending samples statement: %w", err)
				}
			}
		}

		err = statement.Send()
		if err != nil {
			return fmt.Errorf("failed sending samples statement: %w", err)
		}

		return nil
	}()
	if err != nil {
		return fmt.Errorf("failed to write samples: %w", err)
	}

	return nil
}
