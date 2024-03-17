package main

import (
	"context"
	pb "gen/go/greet"
	"log"
	"net"
	"os"
	"os/signal"
	"pkg/otel"
	"syscall"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Failed to serve gRPC server, err: %v", err)
	}
}

func run() error {
	close := otel.NewTracerProvider("greet")
	defer close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	ln, err := net.Listen("tcp", ":8082")
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

	return srv.Serve(ln)
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

func (s *helloServer) SayHello(ctx context.Context, req *pb.NoParam) (*pb.HelloResponse, error) {
	// span := trace.SpanFromContext(ctx)
	// span.SetAttributes(
	// 	attribute.String("app.product.id", req.Id),
	// )

	return &pb.HelloResponse{
		Id: 1,
	}, nil
}
