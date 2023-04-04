package metric

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric/global"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrUndefindedMetricProto = fmt.Errorf("undefined metric protocol, available(http; grpc)")

type Config struct {
	Proto    string
	Endpoint string
}

func InitMeterProvider(ctx context.Context, serviceName, serviceVersion string, cnf Config) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	var metricExporter sdkmetric.Exporter

	switch cnf.Proto {
	case "http":
		// Set up a metric exporter
		if metricExporter, err = otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpoint(cnf.Endpoint)); err != nil {
			return nil, fmt.Errorf("failed to create metric exporter: %w", err)
		}
	case "grpc":
		// Set up a metric exporter
		conn, err := grpc.DialContext(ctx, cnf.Endpoint,
			// Note the use of insecure transport here. TLS is recommended in production.
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
		}

		// Set up a trace exporter
		if metricExporter, err = otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn)); err != nil {
			return nil, fmt.Errorf("failed to create trace exporter: %w", err)
		}
	default:
		return nil, ErrUndefindedMetricProto
	}

	pr := sdkmetric.NewPeriodicReader(metricExporter)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(pr),
		sdkmetric.WithResource(res),
	)

	global.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}

// // This is just an example, see the the contrib runtime instrumentation for real implementation.
// func computeGCPauses(ctx context.Context, recorder instrument.Float64Histogram, pauseBuff []uint64) {}
