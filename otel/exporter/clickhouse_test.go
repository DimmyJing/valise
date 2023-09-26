package exporter_test

import (
	"context"
	"testing"
	"time"

	"github.com/DimmyJing/valise/otel/exporter"
	"github.com/stretchr/testify/assert"
)

func TestGetClickhouseExporters(t *testing.T) {
	t.Parallel()

	config := exporter.ClickhouseConfig{
		Endpoint: "localhost:8123",
		Username: "default",
		Password: "password",
		Database: "otel",
		ConnectionParams: map[string]string{
			"secure":      "true",
			"skip_verify": "true",
		},
		LogsTableName:    "otel_logs",
		TracesTableName:  "otel_traces",
		MetricsTableName: "otel_metrics",
		TTLDays:          7,

		Timeout: time.Second * 5,

		Enabled:             true,
		InitialInterval:     time.Second * 1,
		RandomizationFactor: 0.5,
		Multiplier:          2.0,
		MaxInterval:         time.Second * 30,
		MaxElapsedTime:      time.Minute * 5,

		QueueSize: 2048,

		Name: "otel-clickhouse-exporter",
	}

	_, _, _, err := exporter.NewClickhouseExporters(context.Background(), &config)
	assert.NoError(t, err)

	config.Endpoint = ":"
	_, _, _, err = exporter.NewClickhouseExporters(context.Background(), &config)
	assert.Error(t, err)
}
