package main

import (
	"bff/handlers"
	"bff/server"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
)

func main() {
	if err := run(); err != nil {
		log.Printf("failed to terminated server: %v", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", "8080"))
	if err != nil {
		return err
	}
	log.Printf("Server started at %v", l.Addr())

	mux, err := handlers.NewHandler(ctx)
	if err != nil {
		return err
	}

	s := server.NewServer(l, mux)
	return s.Run(ctx)
}
