package otel //nolint:dupl

import (
	"context"
	"fmt"

	"github.com/DimmyJing/valise/otel/internal/transform"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/otel/sdk/trace"
)

type tracesExporter struct {
	traces exporter.Traces
}

func (t *tracesExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}

	err := t.traces.ConsumeTraces(ctx, transform.Spans(spans))
	if err != nil {
		return fmt.Errorf("error consuming %d traces: %w", len(spans), err)
	}

	return nil
}

func (t *tracesExporter) Shutdown(ctx context.Context) error {
	err := t.traces.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("error shutting down traces exporter: %w", err)
	}

	return nil
}

func NewTracesExporter(ctx context.Context, traces exporter.Traces) (*tracesExporter, error) {
	err := traces.Start(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting traces: %w", err)
	}

	exporter := &tracesExporter{traces: traces}

	return exporter, nil
}
