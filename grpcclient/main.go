package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	pb "otel-go-sample/proto"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Trace Providerのセット
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	// 後続のサービスにつなげるためにpropagaterを追加
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

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
