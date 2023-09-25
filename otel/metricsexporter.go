package otel

import (
	"context"
	"fmt"

	"github.com/DimmyJing/valise/otel/internal/transform"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type metricsExporter struct {
	metrics       exporter.Metrics
	temporalityFn func(metric.InstrumentKind) metricdata.Temporality
	aggregationFn func(metric.InstrumentKind) metric.Aggregation
}

func (m metricsExporter) Temporality(kind metric.InstrumentKind) metricdata.Temporality {
	if m.temporalityFn != nil {
		return m.temporalityFn(kind)
	} else {
		return metricdata.DeltaTemporality
	}
}

func (m metricsExporter) Aggregation( //nolint:ireturn
	kind metric.InstrumentKind,
) metric.Aggregation {
	if m.aggregationFn != nil {
		return m.aggregationFn(kind)
	} else {
		return metric.DefaultAggregationSelector(kind)
	}
}

func (m metricsExporter) Export(
	ctx context.Context,
	resourceMetrics *metricdata.ResourceMetrics,
) error {
	if len(resourceMetrics.ScopeMetrics) == 0 {
		return nil
	}

	metrics, err := transform.ResourceMetrics(resourceMetrics)
	if err != nil {
		return fmt.Errorf("error transforming resource metrics: %w", err)
	}

	err = m.metrics.ConsumeMetrics(ctx, metrics)
	if err != nil {
		return fmt.Errorf("error consuming %d metrics: %w", len(resourceMetrics.ScopeMetrics), err)
	}

	return nil
}

func (m metricsExporter) ForceFlush(_ context.Context) error {
	return nil
}

func (m metricsExporter) Shutdown(ctx context.Context) error {
	err := m.metrics.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("error shutting down metrics exporter: %w", err)
	}

	return nil
}

type MetricsExporterOption interface {
	option()
}

type withTemporalityFn struct {
	fn func(metric.InstrumentKind) metricdata.Temporality
}

func (withTemporalityFn) option() {}

func WithTemporality(fn func(metric.InstrumentKind) metricdata.Temporality) withTemporalityFn {
	return withTemporalityFn{fn: fn}
}

type withAggregationFn struct {
	fn func(metric.InstrumentKind) metric.Aggregation
}

func (withAggregationFn) option() {}

func WithAggregation(fn func(metric.InstrumentKind) metric.Aggregation) withAggregationFn {
	return withAggregationFn{fn: fn}
}

func NewMetricsExporter(
	ctx context.Context,
	metrics exporter.Metrics,
	options ...MetricsExporterOption,
) (*metricsExporter, error) {
	err := metrics.Start(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting metrics: %w", err)
	}

	exporter := &metricsExporter{metrics: metrics, temporalityFn: nil, aggregationFn: nil}

	for _, option := range options {
		switch option := option.(type) {
		case withTemporalityFn:
			exporter.temporalityFn = option.fn
		case withAggregationFn:
			exporter.aggregationFn = option.fn
		}
	}

	return exporter, nil
}
