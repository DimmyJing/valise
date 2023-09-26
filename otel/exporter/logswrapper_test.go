package exporter_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/DimmyJing/valise/log"
	"github.com/DimmyJing/valise/otel/exporter"
	"github.com/DimmyJing/valise/otel/otellog"
	"github.com/stretchr/testify/assert"
)

func TestLogsExporter(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	logger := log.New(log.WithCharm(), log.WithWriter(buf))
	_, _, logs := exporter.NewIOWriterExporters(context.Background(), logger)

	exp, err := exporter.NewLogsWrapper(context.Background(), logs)
	assert.NoError(t, err)

	err = exp.ExportLogs(context.Background(), nil)
	assert.NoError(t, err)

	err = exp.ExportLogs(context.Background(), []otellog.ReadOnlyLog{nil})
	assert.NoError(t, err)

	err = exp.Shutdown(context.Background())
	assert.NoError(t, err)

	_, err = exporter.NewLogsWrapper(context.Background(), &failExporter{failStart: true})
	assert.Error(t, err)

	exp, err = exporter.NewLogsWrapper(context.Background(), &failExporter{failStart: false})
	assert.NoError(t, err)

	err = exp.ExportLogs(context.Background(), []otellog.ReadOnlyLog{nil})
	assert.Error(t, err)

	err = exp.Shutdown(context.Background())
	assert.Error(t, err)
}
