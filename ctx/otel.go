package ctx

import (
	"github.com/DimmyJing/valise/attr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	otelTracerKey contextKey = "otelTracer"
	otelMeterKey  contextKey = "otelMeter"
	otelLogKey    contextKey = "otelLog"
)

func (c Context) WithOTelTracer(t trace.Tracer) Context {
	return c.WithValue(otelTracerKey, t)
}

func (c Context) OTelTracer() trace.Tracer { //nolint:ireturn
	if tracer, ok := Value[trace.Tracer](c, otelTracerKey); ok {
		return tracer
	}

	return nil
}

func (c Context) WithOTelMeter(m metric.Meter) Context {
	return c.WithValue(otelMeterKey, m)
}

func (c Context) OTelMeter() metric.Meter { //nolint:ireturn
	if meter, ok := Value[metric.Meter](c, otelMeterKey); ok {
		return meter
	}

	return nil
}

func (c Context) WithOTelLog(l log.Logger) Context {
	return c.WithValue(otelLogKey, l)
}

func (c Context) OTelLog() log.Logger { //nolint:ireturn
	if logger, ok := Value[log.Logger](c, otelLogKey); ok {
		return logger
	}

	return nil
}

func (c Context) Nest(name string, nestedFn func(ctx Context), attrs ...attr.Attr) {
	if tracer, ok := Value[trace.Tracer](c, otelTracerKey); ok {
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

func (c Context) Nested(name string, attrs ...attr.Attr) (Context, func()) {
	if tracer, ok := Value[trace.Tracer](c, otelTracerKey); ok {
		res := make([]attribute.KeyValue, len(attrs))
		for i, a := range attrs {
			res[i] = attr.OtelAttr(a)
		}

		ctx, span := tracer.Start(
			c,
			name,
			trace.WithAttributes(res...),
		)

		return From(ctx), func() { span.End() }
	}

	return c, func() { trace.SpanFromContext(c).End() }
}

func (c Context) NestedClient(name string, attrs ...attr.Attr) (Context, func()) {
	if tracer, ok := Value[trace.Tracer](c, otelTracerKey); ok {
		res := make([]attribute.KeyValue, len(attrs))
		for i, a := range attrs {
			res[i] = attr.OtelAttr(a)
		}

		ctx, span := tracer.Start(
			c,
			name,
			trace.WithAttributes(res...),
			trace.WithSpanKind(trace.SpanKindClient),
		)

		return From(ctx), func() { span.End() }
	}

	return c, func() { trace.SpanFromContext(c).End() }
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
