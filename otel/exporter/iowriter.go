package exporter

import (
	"context"

	"github.com/DimmyJing/valise/log"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type ioWriterExporter struct {
	logger *log.Logger
}

func (e *ioWriterExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{
		MutatesData: false,
	}
}

func (e *ioWriterExporter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	e.logger.InfoContext(ctx, td)

	return nil
}

func (e *ioWriterExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	e.logger.InfoContext(ctx, md)

	return nil
}

func (e *ioWriterExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	e.logger.InfoContext(ctx, ld)

	return nil
}

func (e *ioWriterExporter) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (e *ioWriterExporter) Shutdown(context.Context) error {
	return nil
}

func NewIOWriterExporters( //nolint:ireturn
	ctx context.Context,
	logger *log.Logger,
) (exporter.Traces, exporter.Metrics, exporter.Logs) {
	exporter := &ioWriterExporter{logger: logger}

	return exporter, exporter, exporter
}
