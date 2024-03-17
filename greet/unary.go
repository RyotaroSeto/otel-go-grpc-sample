package main

import (
	"context"
	pb "otel-go-sample/proto"
)

func (s *helloServer) SayHello(ctx context.Context, req *pb.NoParam) (*pb.HelloResponse, error) {
	// span := trace.SpanFromContext(ctx)
	// span.SetAttributes(
	// 	attribute.String("app.product.id", req.Id),
	// )

	return &pb.HelloResponse{
		Message: "Hello",
	}, nil
}
