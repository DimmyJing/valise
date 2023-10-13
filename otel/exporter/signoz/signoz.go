package signoz

import (
	"context"
	"fmt"
	"sync"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/SigNoz/signoz-otel-collector/processor/signozspanmetricsprocessor"
	"github.com/segmentio/ksuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

const (
	traceDB           = "signoz_traces"
	metricsDB         = "signoz_metrics"
	logsDB            = "signoz_logs"
	indexTable        = "signoz_index_v2"
	attributeTable    = "span_attributes"
	attributeKeyTable = "span_attributes_keys"
	errorTable        = "signoz_error_index_v2"
	spansTable        = "signoz_spans"
	timeSeriesTable   = "time_series_v2"
	timeSeriesTableV3 = "time_series_v3"
	samplesTable      = "samples_v2"
	logsTable         = "logs"
	tagAttributeTable = "tag_attributes"
)

type signozExporter struct {
	dsn             string
	conn            driver.Conn
	timeSeriesCache map[uint64]struct{}
	ksuid           ksuid.KSUID
	connClosed      bool
	mutex           sync.Mutex
	tracesProcessor processor.Traces
}

var (
	_ exporter.Traces  = (*signozExporter)(nil)
	_ exporter.Metrics = (*signozExporter)(nil)
	_ exporter.Logs    = (*signozExporter)(nil)
)

func NewSignozExporter(dsn string) (*signozExporter, error) {
	return &signozExporter{
		dsn:             dsn,
		conn:            nil,
		timeSeriesCache: make(map[uint64]struct{}),
		ksuid:           ksuid.New(),
		connClosed:      false,
		mutex:           sync.Mutex{},
		tracesProcessor: nil,
	}, nil
}

func (s *signozExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type signozHost struct {
	exporters map[component.DataType]map[component.ID]component.Component
}

func (s *signozHost) ReportFatalError(err error) {}

func (s *signozHost) GetFactory(kind component.Kind, componentType component.Type) component.Factory { //nolint:ireturn
	return nil
}

func (s *signozHost) GetExtensions() map[component.ID]component.Component {
	return nil
}

func (s *signozHost) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return s.exporters
}

type noopTraceExporter struct{}

func (n noopTraceExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (n noopTraceExporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	return nil
}

func (s *signozExporter) initConnection(ctx context.Context) error { //nolint:funlen
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.conn != nil {
		return nil
	}

	opts, err := clickhouse.ParseDSN(s.dsn)
	if err != nil {
		return fmt.Errorf("error parsing clickhouse dsn: %w", err)
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return fmt.Errorf("error opening clickhouse: %w", err)
	}

	err = conn.Ping(context.Background())
	if err != nil {
		return fmt.Errorf("error pinging clickhouse: %w", err)
	}

	s.conn = conn

	factory := signozspanmetricsprocessor.NewFactory()
	config := factory.CreateDefaultConfig()

	loggerCfg := zap.NewProductionConfig()
	loggerCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	logger, _ := loggerCfg.Build()

	traces, err := factory.CreateTracesProcessor(ctx, processor.CreateSettings{
		ID: component.NewID(component.DataTypeTraces),
		TelemetrySettings: component.TelemetrySettings{
			Logger:                logger,
			TracerProvider:        nil,
			MeterProvider:         nil,
			MetricsLevel:          configtelemetry.LevelNone,
			Resource:              pcommon.NewResource(),
			ReportComponentStatus: nil,
		},
		BuildInfo: component.NewDefaultBuildInfo(),
	}, config, noopTraceExporter{})
	if err != nil {
		return fmt.Errorf("error creating traces processor: %w", err)
	}

	err = traces.Start(ctx, &signozHost{
		exporters: map[component.Type]map[component.ID]component.Component{
			component.DataTypeMetrics: {
				component.NewID(component.DataType("")): s,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error starting traces processor: %w", err)
	}

	s.tracesProcessor = traces

	return nil
}

func (s *signozExporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	err := s.initConnection(ctx)
	if err != nil {
		return fmt.Errorf("error initializing clickhouse connection: %w", err)
	}

	err = s.convertTraces(ctx, traces)
	if err != nil {
		return fmt.Errorf("error while consuming traces: %w", err)
	}

	// TODO: figure out why this is not working on dev
	_ = s.tracesProcessor.ConsumeTraces(ctx, traces)
	/*
		if err != nil {
			return fmt.Errorf("error while consuming traces metrics: %w", err)
		}
	*/

	return nil
}

func (s *signozExporter) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	err := s.initConnection(ctx)
	if err != nil {
		return fmt.Errorf("error initializing clickhouse connection: %w", err)
	}

	err = s.convertMetrics(ctx, metrics)
	if err != nil {
		return fmt.Errorf("error while consuming metrics: %w", err)
	}

	return nil
}

func (s *signozExporter) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	err := s.initConnection(ctx)
	if err != nil {
		return fmt.Errorf("error initializing clickhouse connection: %w", err)
	}

	err = s.convertLogs(ctx, logs)
	if err != nil {
		return fmt.Errorf("error while consuming logs: %w", err)
	}

	return nil
}

func (s *signozExporter) Shutdown(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.connClosed {
		return nil
	}

	if s.conn != nil {
		err := s.conn.Close()
		if err != nil {
			return fmt.Errorf("error closing clickhouse connection: %w", err)
		}
	}

	if s.tracesProcessor != nil {
		err := s.tracesProcessor.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("error shutting down traces processor: %w", err)
		}
	}

	s.connClosed = true

	return nil
}

func (s *signozExporter) Start(ctx context.Context, host component.Host) error {
	return nil
}
