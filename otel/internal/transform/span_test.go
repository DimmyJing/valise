package transform_test

import (
	"testing"
	"time"

	"github.com/DimmyJing/valise/otel/internal/transform"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func getTraceID(id byte) trace.TraceID {
	var traceID trace.TraceID
	for i := 0; i < len(traceID); i++ {
		traceID[i] = id
	}

	return traceID
}

func getSpanID(id byte) trace.SpanID {
	var spanID trace.SpanID
	for i := 0; i < len(spanID); i++ {
		spanID[i] = id
	}

	return spanID
}

type spanBuilder struct {
	resource *resource.Resource
	scope    instrumentation.Scope

	spans          tracetest.SpanStubs
	pTrace         ptrace.Traces
	pResourceSpans ptrace.ResourceSpansSlice
	pResourceSpan  ptrace.ResourceSpans
	pScopeSpans    ptrace.ScopeSpansSlice
	pScopeSpan     ptrace.ScopeSpans
	pSpans         ptrace.SpanSlice
	pSpan          ptrace.Span
}

func newSpanBuilder() *spanBuilder {
	trace := ptrace.NewTraces()
	//nolint:exhaustruct
	return &spanBuilder{
		spans:          nil,
		pTrace:         trace,
		pResourceSpans: trace.ResourceSpans(),
	}
}

func (b *spanBuilder) addResource(
	resSchemaURL string,
	resAttrs []attribute.KeyValue,
) {
	b.resource = resource.NewWithAttributes(resSchemaURL, resAttrs...)
	b.pResourceSpan = b.pResourceSpans.AppendEmpty()
	transform.TransformKeyValues(resAttrs, b.pResourceSpan.Resource().Attributes())
	b.pResourceSpan.SetSchemaUrl(resSchemaURL)
	b.pScopeSpans = b.pResourceSpan.ScopeSpans()
}

func (b *spanBuilder) addEmptyResource() {
	b.resource = nil
	b.pResourceSpan = b.pResourceSpans.AppendEmpty()
	b.pScopeSpans = b.pResourceSpan.ScopeSpans()
}

func (b *spanBuilder) addScope(
	scopeName string,
	scopeVersion string,
	scopeSchemaURL string,
) {
	b.scope = instrumentation.Scope{
		Name:      scopeName,
		Version:   scopeVersion,
		SchemaURL: scopeSchemaURL,
	}
	b.pScopeSpan = b.pScopeSpans.AppendEmpty()
	pScope := b.pScopeSpan.Scope()
	pScope.SetName(scopeName)
	pScope.SetVersion(scopeVersion)
	b.pScopeSpan.SetSchemaUrl(scopeSchemaURL)
	b.pSpans = b.pScopeSpan.Spans()
}

func (b *spanBuilder) addSpan( //nolint:funlen
	tb testing.TB,
	name string,
	traceID byte,
	spanID byte,
	traceState string,
	parentSpanID byte,
	spanKind trace.SpanKind,
	pSpanKind ptrace.SpanKind,
	startTime time.Time,
	endTime time.Time,
	attributes []attribute.KeyValue,
	statusCode codes.Code,
	statusDescription string,
	pStatusCode ptrace.StatusCode,
	droppedAttributes int,
	droppedEvents int,
	droppedLinks int,
) {
	tb.Helper()

	tState, err := trace.ParseTraceState(traceState)
	if err != nil {
		tb.Fatal(err)
	}

	span := tracetest.SpanStub{
		Name: name,
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    getTraceID(traceID),
			SpanID:     getSpanID(spanID),
			TraceFlags: 0,
			TraceState: tState,
			Remote:     false,
		}),
		Parent: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    getTraceID(0),
			SpanID:     getSpanID(parentSpanID),
			TraceFlags: 0,
			TraceState: trace.TraceState{},
			Remote:     false,
		}),
		SpanKind:   spanKind,
		StartTime:  startTime,
		EndTime:    endTime,
		Attributes: attributes,
		Events:     []tracesdk.Event{},
		Links:      []tracesdk.Link{},
		Status: tracesdk.Status{
			Code:        statusCode,
			Description: statusDescription,
		},
		DroppedAttributes:      droppedAttributes,
		DroppedEvents:          droppedEvents,
		DroppedLinks:           droppedLinks,
		ChildSpanCount:         0,
		Resource:               b.resource,
		InstrumentationLibrary: b.scope,
	}

	b.pSpan = b.pSpans.AppendEmpty()
	b.pSpan.SetTraceID(pcommon.TraceID(getTraceID(traceID)))
	b.pSpan.SetSpanID(pcommon.SpanID(getSpanID(spanID)))
	ts := b.pSpan.TraceState()
	ts.FromRaw(traceState)
	b.pSpan.SetParentSpanID(pcommon.SpanID(getSpanID(parentSpanID)))
	b.pSpan.SetName(name)
	b.pSpan.SetKind(pSpanKind)
	b.pSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(startTime))
	b.pSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(endTime))
	transform.TransformKeyValues(attributes, b.pSpan.Attributes())
	b.pSpan.SetDroppedAttributesCount(uint32(droppedAttributes))
	b.pSpan.SetDroppedEventsCount(uint32(droppedEvents))
	b.pSpan.SetDroppedLinksCount(uint32(droppedLinks))
	pStatus := b.pSpan.Status()
	pStatus.SetCode(pStatusCode)
	pStatus.SetMessage(statusDescription)

	b.spans = append(b.spans, span)
}

func (b *spanBuilder) addEvent(
	name string,
	attrs []attribute.KeyValue,
	droppedCount int,
	eventTime time.Time,
) {
	span := &b.spans[len(b.spans)-1]
	span.Events = append(span.Events, tracesdk.Event{
		Name:                  name,
		Attributes:            attrs,
		DroppedAttributeCount: droppedCount,
		Time:                  eventTime,
	})
	pEvent := b.pSpan.Events().AppendEmpty()
	pEvent.SetName(name)
	transform.TransformKeyValues(attrs, pEvent.Attributes())
	pEvent.SetDroppedAttributesCount(uint32(droppedCount))
	pEvent.SetTimestamp(pcommon.NewTimestampFromTime(eventTime))
}

func (b *spanBuilder) addLink(
	tb testing.TB,
	traceID byte,
	spanID byte,
	traceState string,
	attrs []attribute.KeyValue,
	droppedCount int,
) {
	tb.Helper()

	tState, err := trace.ParseTraceState(traceState)
	if err != nil {
		tb.Fatal(err)
	}

	span := &b.spans[len(b.spans)-1]
	span.Links = append(span.Links, tracesdk.Link{
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    getTraceID(traceID),
			SpanID:     getSpanID(spanID),
			TraceFlags: 0,
			TraceState: tState,
			Remote:     false,
		}),
		Attributes:            attrs,
		DroppedAttributeCount: droppedCount,
	})
	pLink := b.pSpan.Links().AppendEmpty()
	pLink.SetTraceID(pcommon.TraceID(getTraceID(traceID)))
	pLink.SetSpanID(pcommon.SpanID(getSpanID(spanID)))
	pLink.TraceState().FromRaw(traceState)
	transform.TransformKeyValues(attrs, pLink.Attributes())
	pLink.SetDroppedAttributesCount(uint32(droppedCount))
}

func (b *spanBuilder) compare(t *testing.T) {
	t.Helper()

	readOnlySpans := b.spans.Snapshots()
	want := b.pTrace
	has := transform.Spans(readOnlySpans)
	assert.Equal(t, want, has)
}

func TestSpan(t *testing.T) {
	t.Parallel()

	spanBuilder := newSpanBuilder()
	spanBuilder.addResource("resourceschemaurl1", []attribute.KeyValue{
		attribute.String("resourceattrfirstkey1", "resourceattrfirstvalue1"),
		attribute.String("resourceattrfirstkey1", "resourceattrsecondvalue1"),
	})
	spanBuilder.addScope("scopename1", "scopeversion1", "scopeschemaurl1")
	spanBuilder.addSpan(
		t,
		"spanname1",
		1,
		2,
		"statefirstkey1=statefirstvalue1,statesecondkey1=statesecondvalue1",
		3,
		trace.SpanKindInternal,
		ptrace.SpanKindInternal,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey1", "spanattrfirstvalue1"),
			attribute.String("spanattrsecondkey1", "spanattrsecondvalue1"),
		},
		codes.Unset,
		"UNSET",
		ptrace.StatusCodeUnset,
		1,
		2,
		3,
	)
	spanBuilder.compare(t)
}

func TestEmptySpans(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ptrace.NewTraces(), transform.Spans(nil))
}

func TestNilSpans(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ptrace.NewTraces(), transform.Spans([]tracesdk.ReadOnlySpan{nil}))
}

func TestEmptyScope(t *testing.T) {
	t.Parallel()

	spanBuilder := newSpanBuilder()
	spanBuilder.addEmptyResource()
	spanBuilder.addScope("", "", "")
	spanBuilder.addSpan(
		t,
		"spanname1",
		1,
		2,
		"statefirstkey1=statefirstvalue1,statesecondkey1=statesecondvalue1",
		3,
		trace.SpanKindInternal,
		ptrace.SpanKindInternal,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey1", "spanattrfirstvalue1"),
			attribute.String("spanattrsecondkey1", "spanattrsecondvalue1"),
		},
		codes.Unset,
		"UNSET",
		ptrace.StatusCodeUnset,
		1,
		2,
		3,
	)
	spanBuilder.compare(t)
}

func TestStatus(t *testing.T) {
	t.Parallel()

	spanBuilder := newSpanBuilder()
	spanBuilder.addResource("resourceschemaurl1", []attribute.KeyValue{
		attribute.String("resourceattrfirstkey1", "resourceattrfirstvalue1"),
		attribute.String("resourceattrfirstkey1", "resourceattrsecondvalue1"),
	})
	spanBuilder.addScope("scopename1", "scopeversion1", "scopeschemaurl1")
	spanBuilder.addSpan(
		t,
		"spanname1",
		1,
		2,
		"statefirstkey1=statefirstvalue1,statesecondkey1=statesecondvalue1",
		3,
		trace.SpanKindInternal,
		ptrace.SpanKindInternal,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey1", "spanattrfirstvalue1"),
			attribute.String("spanattrsecondkey1", "spanattrsecondvalue1"),
		},
		codes.Ok,
		"OK",
		ptrace.StatusCodeOk,
		1,
		2,
		3,
	)
	spanBuilder.addSpan(
		t,
		"spanname2",
		4,
		5,
		"statefirstkey2=statefirstvalue2,statesecondkey2=statesecondvalue2",
		6,
		trace.SpanKindInternal,
		ptrace.SpanKindInternal,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey2", "spanattrfirstvalue2"),
			attribute.String("spanattrsecondkey2", "spanattrsecondvalue2"),
		},
		codes.Error,
		"ERROR",
		ptrace.StatusCodeError,
		4,
		5,
		6,
	)
	spanBuilder.compare(t)
}

func TestSpanKind(t *testing.T) { //nolint:funlen
	t.Parallel()

	spanBuilder := newSpanBuilder()
	spanBuilder.addResource("resourceschemaurl1", []attribute.KeyValue{
		attribute.String("resourceattrfirstkey1", "resourceattrfirstvalue1"),
		attribute.String("resourceattrfirstkey1", "resourceattrsecondvalue1"),
	})
	spanBuilder.addScope("scopename1", "scopeversion1", "scopeschemaurl1")
	spanBuilder.addSpan(
		t,
		"spanname1",
		1,
		2,
		"statefirstkey1=statefirstvalue1,statesecondkey1=statesecondvalue1",
		3,
		trace.SpanKindInternal,
		ptrace.SpanKindInternal,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey1", "spanattrfirstvalue1"),
			attribute.String("spanattrsecondkey1", "spanattrsecondvalue1"),
		},
		codes.Ok,
		"OK",
		ptrace.StatusCodeOk,
		1,
		2,
		3,
	)
	spanBuilder.addSpan(
		t,
		"spanname2",
		4,
		5,
		"statefirstkey2=statefirstvalue2,statesecondkey2=statesecondvalue2",
		6,
		trace.SpanKindClient,
		ptrace.SpanKindClient,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey2", "spanattrfirstvalue2"),
			attribute.String("spanattrsecondkey2", "spanattrsecondvalue2"),
		},
		codes.Error,
		"ERROR",
		ptrace.StatusCodeError,
		4,
		5,
		6,
	)
	spanBuilder.addScope("scopename2", "scopeversion2", "scopeschemaurl2")
	spanBuilder.addSpan(
		t,
		"spanname3",
		7,
		8,
		"statefirstkey3=statefirstvalue3,statesecondkey3=statesecondvalue3",
		9,
		trace.SpanKindServer,
		ptrace.SpanKindServer,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey3", "spanattrfirstvalue3"),
			attribute.String("spanattrsecondkey3", "spanattrsecondvalue3"),
		},
		codes.Ok,
		"OK",
		ptrace.StatusCodeOk,
		10,
		11,
		12,
	)
	spanBuilder.addSpan(
		t,
		"spanname4",
		13,
		14,
		"statefirstkey4=statefirstvalue4,statesecondkey4=statesecondvalue4",
		15,
		trace.SpanKindProducer,
		ptrace.SpanKindProducer,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey4", "spanattrfirstvalue4"),
			attribute.String("spanattrsecondkey4", "spanattrsecondvalue4"),
		},
		codes.Error,
		"ERROR",
		ptrace.StatusCodeError,
		16,
		17,
		18,
	)
	spanBuilder.addResource("resourceschemaurl2", []attribute.KeyValue{
		attribute.String("resourceattrfirstkey2", "resourceattrfirstvalue2"),
		attribute.String("resourceattrfirstkey2", "resourceattrsecondvalue2"),
	})
	spanBuilder.addScope("scopename3", "scopeversion3", "scopeschemaurl3")
	spanBuilder.addSpan(
		t,
		"spanname5",
		19,
		20,
		"statefirstkey5=statefirstvalue5,statesecondkey5=statesecondvalue5",
		21,
		trace.SpanKindConsumer,
		ptrace.SpanKindConsumer,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey5", "spanattrfirstvalue5"),
			attribute.String("spanattrsecondkey5", "spanattrsecondvalue5"),
		},
		codes.Ok,
		"OK",
		ptrace.StatusCodeOk,
		22,
		23,
		24,
	)
	spanBuilder.addSpan(
		t,
		"spanname6",
		25,
		26,
		"statefirstkey6=statefirstvalue6,statesecondkey6=statesecondvalue6",
		27,
		trace.SpanKindUnspecified,
		ptrace.SpanKindUnspecified,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey6", "spanattrfirstvalue6"),
			attribute.String("spanattrsecondkey6", "spanattrsecondvalue6"),
		},
		codes.Error,
		"ERROR",
		ptrace.StatusCodeError,
		28,
		29,
		30,
	)
	spanBuilder.compare(t)
}

func TestEvent(t *testing.T) {
	t.Parallel()

	spanBuilder := newSpanBuilder()
	spanBuilder.addResource("resourceschemaurl1", []attribute.KeyValue{})
	spanBuilder.addScope("scopename1", "scopeversion1", "scopeschemaurl1")
	spanBuilder.addSpan(
		t,
		"spanname1",
		1,
		2,
		"statefirstkey1=statefirstvalue1,statesecondkey1=statesecondvalue1",
		3,
		trace.SpanKindInternal,
		ptrace.SpanKindInternal,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey1", "spanattrfirstvalue1"),
			attribute.String("spanattrsecondkey1", "spanattrsecondvalue1"),
		},
		codes.Unset,
		"UNSET",
		ptrace.StatusCodeUnset,
		1,
		2,
		3,
	)
	spanBuilder.addEvent("eventname1", []attribute.KeyValue{
		attribute.String("eventattrfirstkey1", "eventattrfirstvalue1"),
		attribute.String("eventattrsecondkey1", "eventattrsecondvalue1"),
	}, 1, time.Unix(1, 2))
	spanBuilder.addEvent("eventname2", []attribute.KeyValue{
		attribute.String("eventattrfirstkey2", "eventattrfirstvalue2"),
		attribute.String("eventattrsecondkey2", "eventattrsecondvalue2"),
	}, 2, time.Unix(3, 4))
	spanBuilder.compare(t)
}

func TestLink(t *testing.T) {
	t.Parallel()

	spanBuilder := newSpanBuilder()
	spanBuilder.addResource("resourceschemaurl1", []attribute.KeyValue{
		attribute.String("resourceattrfirstkey1", "resourceattrfirstvalue1"),
		attribute.String("resourceattrfirstkey1", "resourceattrsecondvalue1"),
	})
	spanBuilder.addScope("scopename1", "scopeversion1", "scopeschemaurl1")
	spanBuilder.addSpan(
		t,
		"spanname1",
		1,
		2,
		"statefirstkey1=statefirstvalue1,statesecondkey1=statesecondvalue1",
		3,
		trace.SpanKindInternal,
		ptrace.SpanKindInternal,
		time.Unix(0, 0),
		time.Unix(1, 1),
		[]attribute.KeyValue{
			attribute.String("spanattrfirstkey1", "spanattrfirstvalue1"),
			attribute.String("spanattrsecondkey1", "spanattrsecondvalue1"),
		},
		codes.Unset,
		"UNSET",
		ptrace.StatusCodeUnset,
		1,
		2,
		3,
	)
	spanBuilder.addLink(
		t,
		4,
		5,
		"statefirstkey1=statefirstvalue1,statesecondkey1=statesecondvalue1",
		[]attribute.KeyValue{
			attribute.String("eventattrfirstkey1", "eventattrfirstvalue1"),
			attribute.String("eventattrsecondkey1", "eventattrsecondvalue1"),
		},
		1,
	)
	spanBuilder.addLink(
		t,
		6,
		7,
		"",
		[]attribute.KeyValue{
			attribute.String("eventattrfirstkey2", "eventattrfirstvalue2"),
			attribute.String("eventattrsecondkey2", "eventattrsecondvalue2"),
		},
		2,
	)
	spanBuilder.compare(t)
}
