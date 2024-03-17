package handlers

import (
	"context"
	"gen/go/todo"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewHandler(ctx context.Context) (http.Handler, error) {
	grpcGateway := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := todo.RegisterTodoApiHandlerFromEndpoint(ctx, grpcGateway, "localhost:8081", opts); err != nil {
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
