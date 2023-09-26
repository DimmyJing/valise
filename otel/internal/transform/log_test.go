package transform_test

import (
	"testing"
	"time"

	"github.com/DimmyJing/valise/otel/internal/transform"
	"github.com/DimmyJing/valise/otel/otellog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
)

type readOnlyLog struct {
	resource               *resource.Resource
	instrumentationScope   instrumentation.Scope
	observedTime           time.Time
	time                   time.Time
	traceID                [16]byte
	spanID                 [8]byte
	flags                  otellog.LogFlags
	severityText           string
	severityNumber         otellog.SeverityNumber
	body                   attribute.Value
	attributes             []attribute.KeyValue
	droppedAttributesCount uint32
}

func (l readOnlyLog) Resource() *resource.Resource                { return l.resource }
func (l readOnlyLog) InstrumentationScope() instrumentation.Scope { return l.instrumentationScope }
func (l readOnlyLog) ObservedTime() time.Time                     { return l.observedTime }
func (l readOnlyLog) Time() time.Time                             { return l.time }
func (l readOnlyLog) TraceID() [16]byte                           { return l.traceID }
func (l readOnlyLog) SpanID() [8]byte                             { return l.spanID }
func (l readOnlyLog) Flags() otellog.LogFlags                     { return l.flags }
func (l readOnlyLog) SeverityText() string                        { return l.severityText }
func (l readOnlyLog) SeverityNumber() otellog.SeverityNumber      { return l.severityNumber }
func (l readOnlyLog) Body() attribute.Value                       { return l.body }
func (l readOnlyLog) Attributes() []attribute.KeyValue            { return l.attributes }

func (l readOnlyLog) DroppedAttributesCount() uint32 { return l.droppedAttributesCount }

type logBuilder struct {
	resource *resource.Resource
	scope    instrumentation.Scope

	logs          []readOnlyLog
	pLog          plog.Logs
	pResourceLogs plog.ResourceLogsSlice
	pResourceLog  plog.ResourceLogs
	pScopeLogs    plog.ScopeLogsSlice
	pScopeLog     plog.ScopeLogs
	pLogRecords   plog.LogRecordSlice
}

func newLogBuilder() *logBuilder {
	logs := plog.NewLogs()
	//nolint:exhaustruct
	return &logBuilder{
		logs:          nil,
		pLog:          logs,
		pResourceLogs: logs.ResourceLogs(),
	}
}

func (b *logBuilder) addResource(
	resSchemaURL string,
	resAttrs []attribute.KeyValue,
) {
	b.resource = resource.NewWithAttributes(resSchemaURL, resAttrs...)
	b.pResourceLog = b.pResourceLogs.AppendEmpty()
	transform.TransformKeyValues(resAttrs, b.pResourceLog.Resource().Attributes())
	b.pResourceLog.SetSchemaUrl(resSchemaURL)
	b.pScopeLogs = b.pResourceLog.ScopeLogs()
}

func (b *logBuilder) addScope(
	scopeName string,
	scopeVersion string,
	scopeSchemaURL string,
) {
	b.scope = instrumentation.Scope{
		Name:      scopeName,
		Version:   scopeVersion,
		SchemaURL: scopeSchemaURL,
	}
	b.pScopeLog = b.pScopeLogs.AppendEmpty()
	pScope := b.pScopeLog.Scope()
	pScope.SetName(scopeName)
	pScope.SetVersion(scopeVersion)
	b.pScopeLog.SetSchemaUrl(scopeSchemaURL)
	b.pLogRecords = b.pScopeLog.LogRecords()
}

func (b *logBuilder) addLog(
	tb testing.TB,
	observedTime time.Time,
	logTime time.Time,
	traceID byte,
	spanID byte,
	flags otellog.LogFlags,
	pFlags plog.LogRecordFlags,
	severityText string,
	severityNumber otellog.SeverityNumber,
	pSeverityNumber plog.SeverityNumber,
	body attribute.Value,
	attrs []attribute.KeyValue,
	droppedAttributesCount uint32,
) {
	tb.Helper()

	b.logs = append(b.logs, readOnlyLog{
		resource:               b.resource,
		instrumentationScope:   b.scope,
		observedTime:           observedTime,
		time:                   logTime,
		traceID:                getTraceID(traceID),
		spanID:                 getSpanID(spanID),
		flags:                  flags,
		severityText:           severityText,
		severityNumber:         severityNumber,
		body:                   body,
		attributes:             attrs,
		droppedAttributesCount: droppedAttributesCount,
	})

	pLogRecord := b.pLogRecords.AppendEmpty()
	pLogRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(observedTime))
	pLogRecord.SetTimestamp(pcommon.NewTimestampFromTime(logTime))
	pLogRecord.SetTraceID(pcommon.TraceID(getTraceID(traceID)))
	pLogRecord.SetSpanID(pcommon.SpanID(getSpanID(spanID)))
	pLogRecord.SetFlags(pFlags)
	pLogRecord.SetSeverityText(severityText)
	pLogRecord.SetSeverityNumber(pSeverityNumber)

	pMap := pcommon.NewMap()
	transform.TransformKeyValues([]attribute.KeyValue{{Key: "body", Value: body}}, pMap)

	pBody, found := pMap.Get("body")
	if !found {
		tb.Fatal("body not found")
	}

	pBody.CopyTo(pLogRecord.Body())
	transform.TransformKeyValues(attrs, pLogRecord.Attributes())
	pLogRecord.SetDroppedAttributesCount(droppedAttributesCount)
}

func (b *logBuilder) compare(t *testing.T) {
	t.Helper()

	logsSlice := make([]otellog.ReadOnlyLog, len(b.logs))
	for i, l := range b.logs {
		logsSlice[i] = l
	}

	want := b.pLog
	has := transform.Logs(logsSlice)
	assert.Equal(t, want, has)
}

func TestOneLog(t *testing.T) {
	t.Parallel()

	builder := newLogBuilder()
	builder.addResource("resource1", []attribute.KeyValue{
		attribute.String("resourcekey1", "resourcevalue1"),
		attribute.String("resourcekey2", "resourcevalue2"),
	})
	builder.addScope("scope1", "scope1version", "scope1schemaurl")
	builder.addLog(
		t,
		time.Unix(0, 1),
		time.Unix(2, 3),
		4,
		5,
		otellog.LogFlagsDefault,
		plog.DefaultLogRecordFlags,
		"severitytext1",
		otellog.SeverityNumberTrace,
		plog.SeverityNumberTrace,
		attribute.StringValue("body1"),
		[]attribute.KeyValue{
			attribute.String("key1", "value1"),
			attribute.String("key2", "value2"),
		},
		0,
	)
	builder.compare(t)
}

func TestMultipleLogs(t *testing.T) { //nolint:funlen
	t.Parallel()

	builder := newLogBuilder()
	builder.addResource("resource1", []attribute.KeyValue{
		attribute.String("resourcekey1", "resourcevalue1"),
		attribute.String("resourcekey2", "resourcevalue2"),
	})
	builder.addScope("scope1", "scope1version", "scope1schemaurl")
	builder.addLog(
		t,
		time.Unix(0, 1),
		time.Unix(2, 3),
		4,
		5,
		otellog.LogFlagsDefault,
		plog.DefaultLogRecordFlags,
		"severitytext1",
		otellog.SeverityNumberTrace,
		plog.SeverityNumberTrace,
		attribute.StringValue("body1"),
		[]attribute.KeyValue{
			attribute.String("key1", "value1"),
			attribute.String("key2", "value2"),
		},
		6,
	)
	builder.addLog(
		t,
		time.Unix(7, 8),
		time.Unix(9, 10),
		11,
		12,
		otellog.LogFlagsIsSampled,
		plog.DefaultLogRecordFlags.WithIsSampled(true),
		"severitytext2",
		otellog.SeverityNumberDebug,
		plog.SeverityNumberDebug,
		attribute.StringValue("body2"),
		[]attribute.KeyValue{
			attribute.String("key3", "value4"),
			attribute.String("key5", "value6"),
		},
		13,
	)
	builder.addResource("resource2", []attribute.KeyValue{
		attribute.String("resourcekey3", "resourcevalue3"),
		attribute.String("resourcekey4", "resourcevalue4"),
	})
	builder.addScope("scope2", "scope2version", "scope2schemaurl")
	builder.addLog(
		t,
		time.Unix(14, 15),
		time.Unix(16, 17),
		18,
		19,
		otellog.LogFlagsDefault,
		plog.DefaultLogRecordFlags,
		"severitytext3",
		otellog.SeverityNumberInfo,
		plog.SeverityNumberInfo,
		attribute.StringValue("body3"),
		[]attribute.KeyValue{
			attribute.String("key7", "value8"),
			attribute.String("key9", "value10"),
		},
		20,
	)
	builder.addLog(
		t,
		time.Unix(21, 22),
		time.Unix(23, 24),
		25,
		26,
		otellog.LogFlagsDefault,
		plog.DefaultLogRecordFlags,
		"severitytext4",
		otellog.SeverityNumberWarn,
		plog.SeverityNumberWarn,
		attribute.StringValue("body4"),
		[]attribute.KeyValue{
			attribute.String("key11", "value12"),
			attribute.String("key13", "value14"),
		},
		27,
	)
	builder.addScope("scope3", "scope3version", "scope3schemaurl")
	builder.addLog(
		t,
		time.Unix(28, 29),
		time.Unix(30, 31),
		32,
		33,
		otellog.LogFlagsDefault,
		plog.DefaultLogRecordFlags,
		"severitytext5",
		otellog.SeverityNumberError,
		plog.SeverityNumberError,
		attribute.StringValue("body5"),
		[]attribute.KeyValue{
			attribute.String("key15", "value16"),
			attribute.String("key17", "value18"),
		},
		34,
	)
	builder.addLog(
		t,
		time.Unix(35, 36),
		time.Unix(37, 38),
		39,
		40,
		otellog.LogFlagsDefault,
		plog.DefaultLogRecordFlags,
		"severitytext6",
		otellog.SeverityNumberFatal,
		plog.SeverityNumberFatal,
		attribute.StringValue("body6"),
		[]attribute.KeyValue{
			attribute.String("key19", "value20"),
			attribute.String("key21", "value22"),
		},
		41,
	)
	builder.addLog(
		t,
		time.Unix(42, 43),
		time.Unix(44, 45),
		46,
		47,
		otellog.LogFlags(2),
		plog.DefaultLogRecordFlags,
		"severitytext7",
		otellog.SeverityNumber(100),
		plog.SeverityNumberUnspecified,
		attribute.StringValue("body7"),
		[]attribute.KeyValue{
			attribute.String("key23", "value24"),
			attribute.String("key25", "value26"),
		},
		48,
	)
	builder.compare(t)
}

func TestEmptyLogs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, plog.NewLogs(), transform.Logs(nil))
	assert.Equal(t, plog.NewLogs(), transform.Logs([]otellog.ReadOnlyLog{nil}))
}
