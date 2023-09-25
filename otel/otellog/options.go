package otellog

import "go.opentelemetry.io/otel/sdk/resource"

type LogProviderOptions interface {
	option()
}

type withResource struct {
	r *resource.Resource
}

func (w withResource) option() {}

func WithResource(r *resource.Resource) withResource {
	return withResource{r: r}
}

type withSyncer struct {
	exporter LogExporter
}

func (w withSyncer) option() {}

func WithSyncer(exporter LogExporter) withSyncer {
	return withSyncer{exporter: exporter}
}

type withBatcher struct {
	exporter LogExporter
}

func (w withBatcher) option() {}

func WithBatcher(exporter LogExporter) withBatcher {
	return withBatcher{exporter: newBatcher(exporter)}
}
