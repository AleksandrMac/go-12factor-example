package trace

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Proto    string
	Endpoint string
}

var (
	tracer Tracer

	ErrUndefindedTraceProto = fmt.Errorf("undefined trace protocol, available(http; grpc)")
)

type Tracer struct {
	t trace.Tracer
}

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func InitTraceProvider(serviceName, serviceVersion string, cnf Config) (func(context.Context) error, error) {
	ctx := context.Background()

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

	var traceExporter *otlptrace.Exporter

	switch cnf.Proto {
	case "http":
		// Set up a trace exporter
		if traceExporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(cnf.Endpoint)); err != nil {
			return nil, fmt.Errorf("failed to create trace exporter: %w", err)
		}
	case "grpc":
		conn, err := grpc.DialContext(ctx, cnf.Endpoint,
			// Note the use of insecure transport here. TLS is recommended in production.
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
		}

		// Set up a trace exporter
		if traceExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn)); err != nil {
			return nil, fmt.Errorf("failed to create trace exporter: %w", err)
		}
	default:
		return nil, ErrUndefindedTraceProto
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

func GetTracer(name string, opts ...trace.TracerOption) *Tracer {
	tracer.t = otel.Tracer(name, opts...)
	return &tracer
}

func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t.t == nil {
		return ctx, nil
	}
	ctx, span := t.t.Start(ctx, spanName, opts...)
	return ctx, &Spaner{s: span}
}
