package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "otel-go-sample/proto"

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
	"google.golang.org/grpc/credentials/insecure"
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

	grpcRequest()
}

func grpcRequest() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		"localhost:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewGreetServiceClient(conn)

	callSayHello(client)
}

// TraceProviderの追加
// TracerProviderとはTracerへのアクセスを提供する
// TracerとはStartメソッドを持つInterfaceであり、contextとNameを引数にSpanとContextを作成する機能を持つ

// TracerProviderへの追加はプロセスセーフではないといけないので、main関数の中でかくこと
// 間違ってもhandlerとか多数のスレッドで呼び出されるところでやってはいけない
// 後続のサービスへspanのContextを伝播するには、otel.SetTextMapPropagatorを追記すること。
// これはリクエスト送る側だけではなく、受け取る側にも必要