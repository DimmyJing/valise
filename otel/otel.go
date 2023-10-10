package otel

import (
	"context"
	"fmt"
	"os"

	"github.com/DimmyJing/valise/log"
	"github.com/DimmyJing/valise/otel/exporter"
	"github.com/DimmyJing/valise/otel/exporter/signoz"
	"github.com/DimmyJing/valise/otel/otellog"
	colexporter "go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type OTelOptions struct {
	ServiceName           string
	ServiceVersion        string
	DeploymentEnvironment string
	ExtraAttributes       []attribute.KeyValue

	UseClickhouse bool
	ClickhouseDSN string

	Disable bool
}

func Logger(name string) otellog.Logger { //nolint:ireturn
	return logProvider.Logger(name)
}

func initResource(
	serviceName string,
	serviceVersion string,
	deploymentEnvironment string,
	extras ...attribute.KeyValue,
) (*resource.Resource, error) {
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

	otelResource, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(resource.Default().SchemaURL(), extras...),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating resource: %w", err)
	}

	return otelResource, nil
}

//nolint:gochecknoglobals
var (
	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
	logProvider    otellog.LogProvider
)

func Init(ctx context.Context, options OTelOptions) error { //nolint:funlen,cyclop
	defer func() {
		if tracerProvider == nil {
			tracerProvider = trace.NewNoopTracerProvider()
		}

		otel.SetTracerProvider(tracerProvider)

		if meterProvider == nil {
			meterProvider = noop.NewMeterProvider()
		}

		otel.SetMeterProvider(meterProvider)

		if logProvider == nil {
			logProvider = otellog.NewLogProvider()
		}
	}()

	//nolint:nestif
	if !options.Disable {
		otelResource, err := initResource(
			options.ServiceName,
			options.ServiceVersion,
			options.DeploymentEnvironment,
			options.ExtraAttributes...,
		)
		if err != nil {
			return err
		}

		var (
			traces  colexporter.Traces
			metrics colexporter.Metrics
			logs    colexporter.Logs
		)

		if options.UseClickhouse && options.ClickhouseDSN != "" {
			exporter, err := signoz.NewSignozExporter(options.ClickhouseDSN)
			if err != nil {
				return fmt.Errorf("failed to initialize signoz exporter: %w", err)
			}

			traces, metrics, logs = exporter, exporter, exporter
		} else {
			traces, metrics, logs = exporter.NewIOWriterExporters(ctx, log.Default())
			if err != nil {
				return fmt.Errorf("failed to initialize stdout exporters: %w", err)
			}
		}

		tracesExporter, err := exporter.NewTracesWrapper(ctx, traces)
		if err != nil {
			return fmt.Errorf("failed to create traces wrapper: %w", err)
		}

		metricsExporter, err := exporter.NewMetricsWrapper(ctx, metrics)
		if err != nil {
			return fmt.Errorf("failed to create metrics wrapper: %w", err)
		}

		logsExporter, err := exporter.NewLogsWrapper(ctx, logs)
		if err != nil {
			return fmt.Errorf("failed to create logs wrapper: %w", err)
		}

		tracerProvider = tracesdk.NewTracerProvider(
			tracesdk.WithResource(otelResource),
			tracesdk.WithBatcher(tracesExporter),
			tracesdk.WithSampler(tracesdk.AlwaysSample()),
		)

		meterProvider = metricsdk.NewMeterProvider(
			metricsdk.WithResource(otelResource),
			metricsdk.WithReader(metricsdk.NewPeriodicReader(metricsExporter)),
		)

		logProvider = otellog.NewLogProvider(
			otellog.WithResource(otelResource),
			//nolint:contextcheck
			otellog.WithBatcher(logsExporter),
		)
	}

	return nil
}

func Shutdown(ctx context.Context) error {
	var traceErr, metricErr, logErr error

	if provider, ok := tracerProvider.(*tracesdk.TracerProvider); ok {
		traceErr = provider.Shutdown(ctx)
	}

	if provider, ok := meterProvider.(*metricsdk.MeterProvider); ok {
		metricErr = provider.Shutdown(ctx)
	}

	if provider, ok := logProvider.(*otellog.DefaultLogProvider); ok {
		logErr = provider.Shutdown(ctx)
	}

	if traceErr != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", traceErr)
	}

	if metricErr != nil {
		return fmt.Errorf("failed to shutdown metric provider: %w", metricErr)
	}

	if logErr != nil {
		return fmt.Errorf("failed to shutdown log provider: %w", logErr)
	}

	return nil
}
