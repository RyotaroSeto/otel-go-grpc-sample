package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	todoPb "gen/go/todo"

	"todo/ui"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	mp := initMeterProvider()
	defer func() {
		if err := mp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down meter provider: %v", err)
		}
		log.Println("Shutdown meter provider")
	}()
	openfeature.SetProvider(flagd.NewProvider())

	err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Failed to start server %v", err)
	}
	log.Printf("Server started at %v", ln.Addr())

	srv := setupServer()
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		log.Println("gRPC server stopped")
	}()

	if err := srv.Serve(ln); err != nil {
		log.Fatalf("Failed to serve gRPC server, err: %v", err)
	}
}

func setupServer() *grpc.Server {
	srv := grpc.NewServer(
		// grpc.ChainUnaryInterceptor(),
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // 分散トレーシングとメトリクスを有効にするためのgrpc.StatsHandlerを追加
	)

	todoPb.RegisterTodoApiServer(srv, ui.NewGRPCService())
	reflection.Register(srv)

	return srv
}

// TraceProviderの追加
// TracerProviderとはTracerへのアクセスを提供する
// TracerとはStartメソッドを持つInterfaceであり、contextとNameを引数にSpanとContextを作成する機能を持つ

// TracerProviderへの追加はプロセスセーフではないといけないので、main関数の中でかくこと
// 間違ってもhandlerとか多数のスレッドで呼び出されるところでやってはいけない
// 後続のサービスへspanのContextを伝播するには、otel.SetTextMapPropagatorを追記すること。
// これはリクエスト送る側だけではなく、受け取る側にも必要
