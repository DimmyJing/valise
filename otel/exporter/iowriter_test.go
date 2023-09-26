package exporter_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/DimmyJing/valise/log"
	"github.com/DimmyJing/valise/otel/exporter"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestIOWriterExporter(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	logger := log.New(log.WithCharm(), log.WithWriter(buf))
	traces, metrics, logs := exporter.NewIOWriterExporters(context.Background(), logger)
	err := traces.Start(context.Background(), nil)
	assert.NoError(t, err)
	err = metrics.Start(context.Background(), nil)
	assert.NoError(t, err)
	err = logs.Start(context.Background(), nil)
	assert.NoError(t, err)
	err = traces.ConsumeTraces(context.Background(), ptrace.NewTraces())
	assert.NoError(t, err)
	err = metrics.ConsumeMetrics(context.Background(), pmetric.NewMetrics())
	assert.NoError(t, err)
	err = logs.ConsumeLogs(context.Background(), plog.NewLogs())
	assert.NoError(t, err)
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, traces.Capabilities())
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, metrics.Capabilities())
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, logs.Capabilities())

	err = traces.Shutdown(context.Background())
	assert.NoError(t, err)
	err = metrics.Shutdown(context.Background())
	assert.NoError(t, err)
	err = logs.Shutdown(context.Background())
	assert.NoError(t, err)
}
