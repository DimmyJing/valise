package signoz

import (
	"context"
	"fmt"
	"sync"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/segmentio/ksuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
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
	}, nil
}

func (s *signozExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (s *signozExporter) initConnection() error {
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

	return nil
}

func (s *signozExporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	err := s.initConnection()
	if err != nil {
		return fmt.Errorf("error initializing clickhouse connection: %w", err)
	}

	err = s.convertTraces(ctx, traces)
	if err != nil {
		return fmt.Errorf("error while consuming traces: %w", err)
	}

	return nil
}

func (s *signozExporter) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	err := s.initConnection()
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
	err := s.initConnection()
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

	if !s.connClosed {
		if s.conn != nil {
			err := s.conn.Close()
			if err != nil {
				return fmt.Errorf("error closing clickhouse connection: %w", err)
			}
		}

		s.connClosed = true
	}

	return nil
}

func (s *signozExporter) Start(ctx context.Context, host component.Host) error {
	return nil
}
