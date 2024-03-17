package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"

	"gen/go/todo"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(); err != nil {
		log.Printf("failed to terminated server: %v", err)
		os.Exit(1)
	}
}

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

func NewJaegerExporter() (trace.SpanExporter, error) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
	if err != nil {
		return nil, err
	}
	return exporter, nil
}

func initTracerProvider() *trace.TracerProvider {
	// ctx := context.Background()
	// exporter, err := otlptracegrpc.New(ctx)
	// if err != nil {
	// 	log.Fatalf("OTLP Trace gRPC Creation: %v", err)
	// }
	exporter, err := NewJaegerExporter()
	if err != nil {
		log.Fatalf("OTLP Trace Creation: %v", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(initResource()), // 何が必要なのか要確認
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp
}

func run() error {
	tp := initTracerProvider()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
		log.Println("Shutdown tracer provider")
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", "8080"))
	if err != nil {
		return err
	}
	log.Printf("Server started at %v", l.Addr())

	mux, err := newHandler(ctx)
	if err != nil {
		return err
	}

	s := newServer(l, mux)
	return s.run(ctx)
}

func newHandler(ctx context.Context) (http.Handler, error) {
	grpcGateway := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := todo.RegisterTodoApiHandlerFromEndpoint(ctx, grpcGateway, "todo:8081", opts); err != nil {
		return nil, err
	}
	otelHandler := otelhttp.NewHandler(grpcGateway, "helloHandler")

	mux := http.NewServeMux()
	mux.Handle("/helthcheck", http.HandlerFunc(healthCheckHandler))
	mux.Handle("/", otelHandler)
	return mux, nil
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type Server struct {
	srv *http.Server
	l   net.Listener
}

func newServer(l net.Listener, mux http.Handler) *Server {
	return &Server{
		srv: &http.Server{Handler: mux},
		l:   l,
	}
}

func (s *Server) run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if err := s.srv.Serve(s.l); err != nil &&
			err != http.ErrServerClosed {
			log.Printf("failed to close: %+v", err)
			return err
		}
		return nil
	})

	<-ctx.Done()
	if err := s.srv.Shutdown(context.Background()); err != nil {
		log.Printf("failed to shutdown: %+v", err)
	}

	return eg.Wait()
}
