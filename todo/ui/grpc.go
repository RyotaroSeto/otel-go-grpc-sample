package ui

import (
	"context"
	greetPb "gen/go/greet"
	pb "gen/go/todo"
	"log"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCService struct {
	pb.UnimplementedTodoApiServer
}

func NewGRPCService() pb.TodoApiServer {
	return &GRPCService{}
}

func (s *GRPCService) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	conn, err := grpc.DialContext(
		ctx,
		"localhost:8082",
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

	return &pb.GetResponse{
		Id: res.Id,
	}, nil
}
