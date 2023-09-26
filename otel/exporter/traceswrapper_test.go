package exporter_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/DimmyJing/valise/log"
	"github.com/DimmyJing/valise/otel/exporter"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

type failExporter struct{ failStart bool }

var errFailExporter = errors.New("fail exporter")

func (f *failExporter) Start(context.Context, component.Host) error {
	if f.failStart {
		return errFailExporter
	}

	return nil
}

func (f *failExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (f *failExporter) ConsumeTraces(_ context.Context, _ ptrace.Traces) error {
	return errFailExporter
}

func (f *failExporter) ConsumeMetrics(_ context.Context, _ pmetric.Metrics) error {
	return errFailExporter
}

func (f *failExporter) ConsumeLogs(_ context.Context, _ plog.Logs) error {
	return errFailExporter
}

func (f *failExporter) Shutdown(_ context.Context) error {
	return errFailExporter
}

func TestTracesExporter(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	logger := log.New(log.WithCharm(), log.WithWriter(buf))
	traces, _, _ := exporter.NewIOWriterExporters(context.Background(), logger)

	exp, err := exporter.NewTracesWrapper(context.Background(), traces)
	assert.NoError(t, err)

	err = exp.ExportSpans(context.Background(), nil)
	assert.NoError(t, err)

	err = exp.ExportSpans(context.Background(), []trace.ReadOnlySpan{nil})
	assert.NoError(t, err)

	err = exp.Shutdown(context.Background())
	assert.NoError(t, err)

	_, err = exporter.NewTracesWrapper(context.Background(), &failExporter{failStart: true})
	assert.Error(t, err)

	exp, err = exporter.NewTracesWrapper(context.Background(), &failExporter{failStart: false})
	assert.NoError(t, err)

	err = exp.ExportSpans(context.Background(), []trace.ReadOnlySpan{nil})
	assert.Error(t, err)

	err = exp.Shutdown(context.Background())
	assert.Error(t, err)
}
