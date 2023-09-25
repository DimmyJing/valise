package transform

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlptrace/internal/tracetransform

type scopeKey struct {
	r  attribute.Distinct
	is instrumentation.Scope
}

func Spans(spans []tracesdk.ReadOnlySpan) ptrace.Traces {
	traces := ptrace.NewTraces()
	if len(spans) == 0 {
		return traces
	}

	resourceSpansMap := make(map[attribute.Distinct]ptrace.ResourceSpans)
	scopeSpansMap := make(map[scopeKey]ptrace.ScopeSpans)
	resourceSpans := traces.ResourceSpans()

	for _, spanData := range spans {
		if spanData == nil {
			continue
		}

		resourceKey := spanData.Resource().Equivalent()
		scopeKey := scopeKey{
			r:  resourceKey,
			is: spanData.InstrumentationScope(),
		}

		resourceSpan, rOk := resourceSpansMap[resourceKey]
		if !rOk {
			// The resource was unknown.
			resourceSpan = resourceSpans.AppendEmpty()
			spanResource := spanData.Resource()
			transformResource(spanResource, resourceSpan.Resource())
			resourceSpan.SetSchemaUrl(spanResource.SchemaURL())
			resourceSpansMap[resourceKey] = resourceSpan
		}

		scopeSpan, iOk := scopeSpansMap[scopeKey]
		if !iOk {
			// Either the resource or instrumentation scope were unknown.
			scopeSpan = resourceSpan.ScopeSpans().AppendEmpty()
			instrumentationScope := spanData.InstrumentationScope()
			transformInstrumentationScope(instrumentationScope, scopeSpan.Scope())
			scopeSpan.SetSchemaUrl(instrumentationScope.SchemaURL)
			scopeSpansMap[scopeKey] = scopeSpan
		}

		transformSpan(spanData, scopeSpan.Spans().AppendEmpty())
	}

	return traces
}

func transformInstrumentationScope(
	scope instrumentation.Scope,
	pScope pcommon.InstrumentationScope,
) {
	//nolint:exhaustruct
	if scope == (instrumentation.Scope{}) {
		return
	}

	pScope.SetName(scope.Name)
	pScope.SetVersion(scope.Version)
}

func transformResource(resource *resource.Resource, pResource pcommon.Resource) {
	if resource == nil {
		return
	}

	TransformKeyValues(resource.Attributes(), pResource.Attributes())
}

func transformSpan(spanData tracesdk.ReadOnlySpan, traceSpan ptrace.Span) {
	traceID := spanData.SpanContext().TraceID()
	spanID := spanData.SpanContext().SpanID()

	traceSpan.SetTraceID(pcommon.TraceID(traceID[:]))
	traceSpan.SetSpanID(pcommon.SpanID(spanID[:]))
	traceSpan.TraceState().FromRaw(spanData.SpanContext().TraceState().String())
	transformStatus(spanData.Status().Code, spanData.Status().Description, traceSpan.Status())
	traceSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(spanData.StartTime()))
	traceSpan.SetEndTimestamp(pcommon.NewTimestampFromTime(spanData.EndTime()))
	transformLinks(spanData.Links(), traceSpan.Links())
	traceSpan.SetKind(spanKind(spanData.SpanKind()))
	traceSpan.SetName(spanData.Name())
	TransformKeyValues(spanData.Attributes(), traceSpan.Attributes())
	transformSpanEvents(spanData.Events(), traceSpan.Events())
	traceSpan.SetDroppedAttributesCount(uint32(spanData.DroppedAttributes()))
	traceSpan.SetDroppedEventsCount(uint32(spanData.DroppedEvents()))
	traceSpan.SetDroppedLinksCount(uint32(spanData.DroppedLinks()))

	if parentSpanID := spanData.Parent().SpanID(); parentSpanID.IsValid() {
		traceSpan.SetParentSpanID(pcommon.SpanID(parentSpanID[:]))
	}
}

func transformStatus(status codes.Code, message string, pStatus ptrace.Status) {
	var code ptrace.StatusCode

	//nolint:exhaustive
	switch status {
	case codes.Ok:
		code = ptrace.StatusCodeOk
	case codes.Error:
		code = ptrace.StatusCodeError
	default:
		code = ptrace.StatusCodeUnset
	}

	pStatus.SetCode(code)
	pStatus.SetMessage(message)
}

func transformLinks(links []tracesdk.Link, spanLinks ptrace.SpanLinkSlice) {
	if len(links) == 0 {
		return
	}

	spanLinks.EnsureCapacity(len(links))

	for _, link := range links {
		// This redefinition is necessary to prevent link.*ID[:] copies
		// being reused -- in short we need a new link per iteration.
		link := link

		tid := link.SpanContext.TraceID()
		sid := link.SpanContext.SpanID()

		spanLink := spanLinks.AppendEmpty()
		spanLink.SetTraceID(pcommon.TraceID(tid[:]))
		spanLink.SetSpanID(pcommon.SpanID(sid[:]))
		spanLink.TraceState().FromRaw(link.SpanContext.TraceState().String())
		TransformKeyValues(link.Attributes, spanLink.Attributes())
		spanLink.SetDroppedAttributesCount(uint32(link.DroppedAttributeCount))
	}
}

func transformSpanEvents(events []tracesdk.Event, spanEvents ptrace.SpanEventSlice) {
	if len(events) == 0 {
		return
	}

	spanEvents.EnsureCapacity(len(events))
	// Transform message events
	for _, event := range events {
		spanEvent := spanEvents.AppendEmpty()
		spanEvent.SetName(event.Name)
		spanEvent.SetTimestamp(pcommon.NewTimestampFromTime(event.Time))
		TransformKeyValues(event.Attributes, spanEvent.Attributes())
		spanEvent.SetDroppedAttributesCount(uint32(event.DroppedAttributeCount))
	}
}

func spanKind(kind trace.SpanKind) ptrace.SpanKind {
	//nolint:exhaustive
	switch kind {
	case trace.SpanKindInternal:
		return ptrace.SpanKindInternal
	case trace.SpanKindClient:
		return ptrace.SpanKindClient
	case trace.SpanKindServer:
		return ptrace.SpanKindServer
	case trace.SpanKindProducer:
		return ptrace.SpanKindProducer
	case trace.SpanKindConsumer:
		return ptrace.SpanKindConsumer
	default:
		return ptrace.SpanKindUnspecified
	}
}
