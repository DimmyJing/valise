package otel

import (
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var otelResource *resource.Resource

func initResource(
	serviceName string,
	serviceVersion string,
	deploymentEnvironment string,
	extras ...attribute.KeyValue,
) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	extras = append(
		extras,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.DeploymentEnvironment(deploymentEnvironment),
		semconv.HostName(hostname),
	)

	otelResource, err = resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(resource.Default().SchemaURL(), extras...),
	)
	if err != nil {
		otelResource = resource.Default()
	}
}
