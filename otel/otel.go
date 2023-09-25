package otel

import "go.opentelemetry.io/otel/attribute"

type OTelOptions struct {
	ServiceName           string
	ServiceVersion        string
	DeploymentEnvironment string
	ExtraAttributes       []attribute.KeyValue

	Disable bool
}

func Init(options OTelOptions) error {
	initResource(
		options.ServiceName,
		options.ServiceVersion,
		options.DeploymentEnvironment,
		options.ExtraAttributes...,
	)

	return nil
}
