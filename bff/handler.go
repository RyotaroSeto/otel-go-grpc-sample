package main

import (
	"context"
	"net/http"
	"otel-go-sample/gen/go/todo"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewHandler(ctx context.Context) (http.Handler, error) {
	grpcGateway := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := todo.RegisterTodoApiHandlerFromEndpoint(ctx, grpcGateway, "end", opts); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/helthcheck", http.HandlerFunc(healthCheckHandler))
	mux.Handle("/", grpcGateway)
	return mux, nil
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
