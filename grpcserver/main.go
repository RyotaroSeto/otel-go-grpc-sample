package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	pb "otel-go-sample/proto"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
)

var (
	resc              *resource.Resource
	initResourcesOnce sync.Once
)

func initResource() *resource.Resource {
	initResourcesOnce.Do(func() {
		extraResources, _ := resource.New(
			context.Background(),
			resource.WithOS(),
			resource.WithProcess(),
			resource.WithContainer(),
			resource.WithHost(),
		)
		resc, _ = resource.Merge(
			resource.Default(),
			extraResources,
		)
	})
	return resc
}

func initTracerProvider() *trace.TracerProvider {
	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		log.Fatalf("OTLP Trace gRPC Creation: %v", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(initResource()), // 何が必要なのか要確認
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

func initMeterProvider() *metric.MeterProvider {
	ctx := context.Background()

	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		log.Fatalf("new otlp metric grpc exporter failed: %v", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(initResource()),
	)
	otel.SetMeterProvider(mp)
	return mp
}

func main() {
	tp := initTracerProvider()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
		log.Println("Shutdown tracer provider")
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	mp := initMeterProvider()
	defer func() {
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down meter provider: %v", err)
		}
		log.Println("Shutdown meter provider")
	}()
	openfeature.SetProvider(flagd.NewProvider())

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start server %v", err)
	}
	log.Printf("Server started at %v", ln.Addr())

	srv := setupServer()
	go func() {
		if err := srv.Serve(ln); err != nil {
			log.Fatalf("Failed to serve gRPC server, err: %v", err)
		}
	}()

	<-ctx.Done()

	srv.GracefulStop()
	log.Println("gRPC server stopped")
}

type helloServer struct {
	pb.GreetServiceServer
}

func setupServer() *grpc.Server {
	srv := grpc.NewServer(
		// grpc.ChainUnaryInterceptor(),
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // 分散トレーシングとメトリクスを有効にするためのgrpc.StatsHandlerを追加
	)

	pb.RegisterGreetServiceServer(srv, &helloServer{})

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)

	reflection.Register(srv)
	return srv
}
