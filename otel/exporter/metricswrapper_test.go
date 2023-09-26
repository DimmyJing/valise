package exporter_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/DimmyJing/valise/log"
	"github.com/DimmyJing/valise/otel/exporter"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestMetricsExporter(t *testing.T) { //nolint:funlen
	t.Parallel()

	buf := new(bytes.Buffer)
	logger := log.New(log.WithCharm(), log.WithWriter(buf))
	_, metrics, _ := exporter.NewIOWriterExporters(context.Background(), logger)

	exp, err := exporter.NewMetricsWrapper(context.Background(), metrics)
	assert.NoError(t, err)
	assert.Equal(t, metricdata.DeltaTemporality, exp.Temporality(metric.InstrumentKindCounter))
	assert.Equal(t, metric.AggregationSum{}, exp.Aggregation(metric.InstrumentKindCounter))

	err = exp.Export(
		context.Background(),
		&metricdata.ResourceMetrics{Resource: resource.Default(), ScopeMetrics: nil},
	)
	assert.NoError(t, err)

	err = exp.Export(
		context.Background(),
		&metricdata.ResourceMetrics{
			Resource: resource.Default(),
			//nolint:exhaustruct
			ScopeMetrics: []metricdata.ScopeMetrics{{}},
		},
	)
	assert.NoError(t, err)

	err = exp.Export(
		context.Background(),
		&metricdata.ResourceMetrics{
			Resource: resource.Default(),
			ScopeMetrics: []metricdata.ScopeMetrics{
				{
					Scope: instrumentation.Scope{Name: "1", Version: "2", SchemaURL: "3"},
					//nolint:exhaustruct
					Metrics: []metricdata.Metrics{
						{
							Data: metricdata.Sum[int64]{
								Temporality: metricdata.Temporality(100),
							},
						},
					},
				},
			},
		},
	)
	assert.Error(t, err)

	err = exp.ForceFlush(context.Background())
	assert.NoError(t, err)

	exp, err = exporter.NewMetricsWrapper(
		context.Background(),
		metrics,
		exporter.WithTemporality(func(_ metric.InstrumentKind) metricdata.Temporality {
			return metricdata.CumulativeTemporality
		}),
		exporter.WithAggregation(func(_ metric.InstrumentKind) metric.Aggregation {
			return metric.AggregationLastValue{}
		}),
	)
	assert.NoError(t, err)
	assert.Equal(t, metricdata.CumulativeTemporality, exp.Temporality(metric.InstrumentKindCounter))
	assert.Equal(t, metric.AggregationLastValue{}, exp.Aggregation(metric.InstrumentKindCounter))
	err = exp.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestFailMetrics(t *testing.T) {
	t.Parallel()

	_, err := exporter.NewMetricsWrapper(context.Background(), &failExporter{failStart: true})
	assert.Error(t, err)

	exp, err := exporter.NewMetricsWrapper(context.Background(), &failExporter{failStart: false})
	assert.NoError(t, err)

	err = exp.Export(
		context.Background(),
		&metricdata.ResourceMetrics{
			Resource: resource.Default(),
			//nolint:exhaustruct
			ScopeMetrics: []metricdata.ScopeMetrics{{}},
		},
	)
	assert.Error(t, err)

	err = exp.Shutdown(context.Background())
	assert.Error(t, err)
}
