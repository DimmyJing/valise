package otellog

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
)

type LogFlags uint32

const (
	LogFlagsDefault LogFlags = iota
	LogFlagsIsSampled
)

type SeverityNumber int32

const (
	SeverityNumberTrace SeverityNumber = -8
	SeverityNumberDebug SeverityNumber = -4
	SeverityNumberInfo  SeverityNumber = 0
	SeverityNumberWarn  SeverityNumber = 4
	SeverityNumberError SeverityNumber = 8
	SeverityNumberFatal SeverityNumber = 12
)

//nolint:interfacebloat
type ReadOnlyLog interface {
	Resource() *resource.Resource
	InstrumentationScope() instrumentation.Scope
	ObservedTime() time.Time
	Time() time.Time
	TraceID() [16]byte
	SpanID() [8]byte
	Flags() LogFlags
	SeverityText() string
	SeverityNumber() SeverityNumber
	Body() attribute.Value
	Attributes() []attribute.KeyValue
	DroppedAttributesCount() uint32
}

type readOnlyLog struct {
	resource               *resource.Resource
	instrumentationScope   instrumentation.Scope
	observedTime           time.Time
	time                   time.Time
	traceID                [16]byte
	spanID                 [8]byte
	flags                  LogFlags
	severityText           string
	severityNumber         SeverityNumber
	body                   attribute.Value
	attributes             []attribute.KeyValue
	droppedAttributesCount uint32
}

func (l readOnlyLog) Resource() *resource.Resource {
	return l.resource
}

func (l readOnlyLog) InstrumentationScope() instrumentation.Scope {
	return l.instrumentationScope
}

func (l readOnlyLog) ObservedTime() time.Time {
	return l.observedTime
}

func (l readOnlyLog) Time() time.Time {
	return l.time
}

func (l readOnlyLog) TraceID() [16]byte {
	return l.traceID
}

func (l readOnlyLog) SpanID() [8]byte {
	return l.spanID
}

func (l readOnlyLog) Flags() LogFlags {
	return l.flags
}

func (l readOnlyLog) SeverityText() string {
	return l.severityText
}

func (l readOnlyLog) SeverityNumber() SeverityNumber {
	return l.severityNumber
}

func (l readOnlyLog) Body() attribute.Value {
	return l.body
}

func (l readOnlyLog) Attributes() []attribute.KeyValue {
	return l.attributes
}

func (l readOnlyLog) DroppedAttributesCount() uint32 {
	return l.droppedAttributesCount
}
