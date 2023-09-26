package exporter

import (
	"context"
	"fmt"

	"github.com/DimmyJing/valise/otel/internal/transform"
	"github.com/DimmyJing/valise/otel/otellog"
	"go.opentelemetry.io/collector/exporter"
)

type logsExporter struct {
	logs exporter.Logs
}

func (l *logsExporter) ExportLogs(ctx context.Context, logs []otellog.ReadOnlyLog) error {
	if len(logs) == 0 {
		return nil
	}

	err := l.logs.ConsumeLogs(ctx, transform.Logs(logs))
	if err != nil {
		return fmt.Errorf("error consuming %d logs: %w", len(logs), err)
	}

	return nil
}

func (l *logsExporter) Shutdown(ctx context.Context) error {
	err := l.logs.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("error shutting down logs exporter: %w", err)
	}

	return nil
}

func NewLogsWrapper(ctx context.Context, logs exporter.Logs) (*logsExporter, error) {
	err := logs.Start(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting logs: %w", err)
	}

	exporter := &logsExporter{logs: logs}

	return exporter, nil
}
