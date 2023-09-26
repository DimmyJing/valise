package otellog

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

type LogExporter interface {
	ExportLogs(ctx context.Context, logs []ReadOnlyLog) error
	Shutdown(ctx context.Context) error
}

type LogProvider interface {
	Logger(name string) Logger
}

type DefaultLogProvider struct {
	resource *resource.Resource
	exporter LogExporter
}

func NewLogProvider(options ...LogProviderOptions) *DefaultLogProvider {
	provider := DefaultLogProvider{resource: nil, exporter: nil}

	for _, option := range options {
		switch option := option.(type) {
		case withResource:
			provider.resource = option.r
		case withSyncer:
			provider.exporter = option.exporter
		case withBatcher:
			provider.exporter = option.exporter
		}
	}

	return &provider
}

func (p *DefaultLogProvider) Logger(name string) Logger { //nolint:ireturn
	return &logger{
		parent: p,
		name:   name,
	}
}

func (p *DefaultLogProvider) Shutdown(ctx context.Context) error {
	if p.exporter != nil {
		err := p.exporter.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("error shutdown exporter: %w", err)
		}
	}

	return nil
}

type Logger interface {
	Log(
		ctx context.Context,
		severityText string,
		severityNumber SeverityNumber,
		body attribute.Value,
		attributes []attribute.KeyValue,
	) error
	RawLog(ctx context.Context, log ReadOnlyLog) error
}

type logger struct {
	parent *DefaultLogProvider
	name   string
}

func (l *logger) Log(
	ctx context.Context,
	severityText string,
	severityNumber SeverityNumber,
	body attribute.Value,
	attributes []attribute.KeyValue,
) error {
	if l.parent.exporter == nil {
		return nil
	}

	span := trace.SpanFromContext(ctx)

	var (
		traceID [16]byte
		spanID  [8]byte
	)

	if span != nil {
		traceID = span.SpanContext().TraceID()
		spanID = span.SpanContext().SpanID()
	}

	now := time.Now()
	result := readOnlyLog{
		resource: l.parent.resource,
		instrumentationScope: instrumentation.Scope{
			Name:      l.name,
			Version:   "", // TODO: add this as option
			SchemaURL: "", // TODO: add this as option
		},
		observedTime:           now,
		time:                   now,
		traceID:                traceID,
		spanID:                 spanID,
		flags:                  LogFlagsIsSampled, // TODO: add this as option
		severityText:           severityText,
		severityNumber:         severityNumber,
		body:                   body,
		attributes:             attributes,
		droppedAttributesCount: 0, // TODO: add attribute dropping
	}

	err := l.parent.exporter.ExportLogs(ctx, []ReadOnlyLog{result})
	if err != nil {
		return fmt.Errorf("error export logs: %w", err)
	}

	return nil
}

func (l *logger) RawLog(ctx context.Context, log ReadOnlyLog) error {
	if l.parent.exporter == nil {
		return nil
	}

	err := l.parent.exporter.ExportLogs(ctx, []ReadOnlyLog{log})
	if err != nil {
		return fmt.Errorf("error export logs: %w", err)
	}

	return nil
}
