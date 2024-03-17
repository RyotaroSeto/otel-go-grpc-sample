package main

import (
	"log"
	"os"
)

func main() {
	if err := run(); err != nil {
		log.Printf("failed to terminated server: %v", err)
		os.Exit(1)
	}
}

func run() error {
	return nil
	// ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	// defer stop()

	// l, err := net.Listen("tcp", fmt.Sprintf(":%s", "8080"))
	// if err != nil {
	// 	return err
	// }

	// mux, err := NewHandler(ctx)
	// if err != nil {
	// 	return err
	// }

	// s := NewServer(l, mux)
	// return s.Run(ctx)
}
