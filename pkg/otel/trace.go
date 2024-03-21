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

var otlpEndpoint string

func init() {
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	// エンドポイントがHTTPはデフォルト(localhost:4318)の場合は不要 エンドポイントがgRPCはデフォルト(localhost:4317)の場合は不要
	otlpEndpoint = os.Getenv("OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		log.Fatalln("You MUST set OTLP_ENDPOINT env variable!")
	}
}

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
		return newTracesJaegerExporter()
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

	r := newResource(serviceName, "1.0.0", "local")
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(r),
	)
	otel.SetTracerProvider(tp)

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
		log.Println("Shutdown tracer provider")
	}
}
