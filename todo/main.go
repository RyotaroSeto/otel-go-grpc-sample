package main

import (
	"context"
	greetPb "gen/go/greet"
	todoPb "gen/go/todo"
	"log"
	"net"
	"os/signal"
	"pkg/otel"
	"syscall"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// TracerProviderへの追加はプロセスセーフではないといけないので、main関数の中でかくこと
// 間違ってもhandlerとか多数のスレッドで呼び出されるところでやってはいけない
// 後続のサービスへspanのContextを伝播するには、otel.SetTextMapPropagatorを追記すること。
// これはリクエスト送る側だけではなく、受け取る側にも必要

func main() {
	close := otel.NewTracerProvider("todo")
	defer close()

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

type todoServer struct {
	todoPb.TodoApiServer
}

func setupServer() *grpc.Server {
	srv := grpc.NewServer(
		// grpc.ChainUnaryInterceptor(),
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // 分散トレーシングとメトリクスを有効にするためのgrpc.StatsHandlerを追加
	)

	todoPb.RegisterTodoApiServer(srv, &todoServer{})
	reflection.Register(srv)

	return srv
}

func (s *todoServer) Get(ctx context.Context, req *todoPb.GetRequest) (*todoPb.GetResponse, error) {
	conn, err := grpc.DialContext(
		ctx,
		"greet:8082",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := greetPb.NewGreetServiceClient(conn)
	res, err := client.SayHello(ctx, &greetPb.NoParam{})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	return &todoPb.GetResponse{
		Id: res.Id,
	}, nil
}
