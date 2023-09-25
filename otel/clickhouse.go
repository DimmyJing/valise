package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ClickhouseConfig struct {
	Endpoint         string
	Username         string
	Password         string
	Database         string
	ConnectionParams map[string]string
	LogsTableName    string
	TracesTableName  string
	MetricsTableName string
	TTLDays          uint

	// timeout settings
	Timeout time.Duration
	// retry settings
	Enabled             bool
	InitialInterval     time.Duration
	RandomizationFactor float64
	Multiplier          float64
	MaxInterval         time.Duration
	MaxElapsedTime      time.Duration
	// queue settings
	QueueSize int

	// exporter settings
	Name string
}

func (c *ClickhouseConfig) exportTo(config *clickhouseexporter.Config) { //nolint:cyclop,funlen
	if c.Endpoint != "" {
		config.Endpoint = c.Endpoint
	}

	if c.Username != "" {
		config.Username = c.Username
	}

	if c.Password != "" {
		config.Password = configopaque.String(c.Password)
	}

	if c.Database != "" {
		config.Database = c.Database
	}

	if c.ConnectionParams != nil {
		config.ConnectionParams = c.ConnectionParams
	}

	if c.LogsTableName != "" {
		config.LogsTableName = c.LogsTableName
	}

	if c.TracesTableName != "" {
		config.TracesTableName = c.TracesTableName
	}

	if c.MetricsTableName != "" {
		config.MetricsTableName = c.MetricsTableName
	}

	if c.TTLDays != 0 {
		config.TTLDays = c.TTLDays
	}

	if c.Timeout.Nanoseconds() != 0 {
		config.Timeout = c.Timeout
	}

	if c.Enabled {
		config.RetrySettings.Enabled = c.Enabled
	}

	if c.InitialInterval.Nanoseconds() != 0 {
		config.RetrySettings.InitialInterval = c.InitialInterval
	}

	if c.RandomizationFactor != 0 {
		config.RetrySettings.RandomizationFactor = c.RandomizationFactor
	}

	if c.Multiplier != 0 {
		config.RetrySettings.Multiplier = c.Multiplier
	}

	if c.MaxInterval.Nanoseconds() != 0 {
		config.RetrySettings.MaxInterval = c.MaxInterval
	}

	if c.MaxElapsedTime.Nanoseconds() != 0 {
		config.RetrySettings.MaxElapsedTime = c.MaxElapsedTime
	}

	if c.QueueSize != 0 {
		config.QueueSettings.QueueSize = c.QueueSize
	}
}

func GetClickhouseExporters( //nolint:ireturn,funlen
	ctx context.Context,
	config *ClickhouseConfig,
) (exporter.Traces, exporter.Metrics, exporter.Logs, error) {
	var (
		traces  exporter.Traces
		metrics exporter.Metrics
		logs    exporter.Logs
	)

	factory := clickhouseexporter.NewFactory()
	//nolint:forcetypeassert
	factoryCfg := factory.CreateDefaultConfig().(*clickhouseexporter.Config)
	config.exportTo(factoryCfg)

	loggerCfg := zap.NewProductionConfig()
	loggerCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)

	logger, err := loggerCfg.Build()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build logger config: %w", err)
	}

	telemetrySettings := component.TelemetrySettings{
		Logger:         logger,
		TracerProvider: trace.NewNoopTracerProvider(),
		MeterProvider:  nil,
		MetricsLevel:   configtelemetry.LevelNone,
		Resource:       pcommon.NewResource(),
	}
	buildInfo := component.NewDefaultBuildInfo()

	tracesSettings := exporter.CreateSettings{
		ID:                component.NewIDWithName(component.DataTypeTraces, config.Name),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         buildInfo,
	}

	traces, err = factory.CreateTracesExporter(ctx, tracesSettings, factoryCfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create traces exporter: %w", err)
	}

	metricsSettings := exporter.CreateSettings{
		ID:                component.NewIDWithName(component.DataTypeMetrics, config.Name),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         buildInfo,
	}

	metrics, err = factory.CreateMetricsExporter(ctx, metricsSettings, factoryCfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	logsSettings := exporter.CreateSettings{
		ID:                component.NewIDWithName(component.DataTypeLogs, config.Name),
		TelemetrySettings: telemetrySettings,
		BuildInfo:         buildInfo,
	}

	logs, err = factory.CreateLogsExporter(ctx, logsSettings, factoryCfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create logs exporter: %w", err)
	}

	return traces, metrics, logs, nil
}
