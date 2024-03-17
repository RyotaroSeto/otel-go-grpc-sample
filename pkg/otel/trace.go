package otel

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func newResource(serviceName, version, env string) *resource.Resource {
	resc, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			attribute.String("environment", env),
		),
	)
	return resc
}

func newExporter(ctx context.Context) (trace.SpanExporter, error) {
	switch os.Getenv("OTEL_EXPORTER_PROTOCOL") {
	case "http":
		return newTracesHttpExporter(ctx)
	case "grpc":
		return newTracesGrpcExporter(ctx)
	case "jaeger":
		return newTracesJaegerExporter(ctx)
	default:
		return newTracesWriterExporter(os.Stdout)
	}
}

func NewTracerProvider(serviceName string) func() {
	ctx := context.Background()
	exporter, err := newExporter(ctx)
	if err != nil {
		log.Fatalf("OTLP Trace Creation: %v", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(newResource(serviceName, "1.0.0", "local")),
	)
	otel.SetTracerProvider(tp)

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
		log.Println("Shutdown tracer provider")
	}
}

func init() {
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
}