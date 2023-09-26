package ctx

import (
	"github.com/DimmyJing/valise/attr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerKey contextKey = "otelTracer"

func (c Context) WithTracer(t trace.Tracer) Context {
	return c.WithValue(tracerKey, t)
}

func (c Context) Nest(name string, nestedFn func(ctx Context), attrs ...attr.Attr) {
	if tracer, ok := Value[trace.Tracer](c, tracerKey); ok {
		res := make([]attribute.KeyValue, len(attrs))
		for i, a := range attrs {
			res[i] = attr.OtelAttr(a)
		}

		ctx, span := tracer.Start(
			c,
			name,
			trace.WithAttributes(res...),
		)
		defer span.End()

		nestedFn(From(ctx))
	} else {
		nestedFn(c)
	}
}

func (c Context) Nested(name string, attrs ...attr.Attr) (Context, trace.Span) { //nolint:ireturn
	if tracer, ok := Value[trace.Tracer](c, tracerKey); ok {
		res := make([]attribute.KeyValue, len(attrs))
		for i, a := range attrs {
			res[i] = attr.OtelAttr(a)
		}

		ctx, span := tracer.Start(
			c,
			name,
			trace.WithAttributes(res...),
		)

		return From(ctx), span
	}

	return c, trace.SpanFromContext(c)
}

func (c Context) SetAttributes(attrs ...attr.Attr) {
	span := trace.SpanFromContext(c)

	res := make([]attribute.KeyValue, len(attrs))
	for i, a := range attrs {
		res[i] = attr.OtelAttr(a)
	}

	span.SetAttributes(res...)
}

func (c Context) SetAnyAttribute(key string, val any) {
	span := trace.SpanFromContext(c)

	span.SetAttributes(attribute.KeyValue{Key: attribute.Key(key), Value: attr.AnyToOtelValue(val)})
}

func (c Context) fail(msg string) {
	span := trace.SpanFromContext(c)
	span.SetStatus(codes.Error, msg)
	span.End()
}

func (c Context) recordEvent(msg any, args []attr.Attr) {
	span := trace.SpanFromContext(c)
	if span.IsRecording() {
		res := make([]attribute.KeyValue, len(args))
		for i, a := range args {
			res[i] = attr.OtelAttr(a)
		}

		if err, ok := msg.(error); ok {
			span.RecordError(
				err,
				trace.WithAttributes(res...),
				trace.WithStackTrace(true),
			)
		} else if msg, ok := msg.(string); ok {
			span.AddEvent(msg, trace.WithAttributes(res...))
		}
	}
}
