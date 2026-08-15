package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/crowd"
	"github.com/kaja-tools/website/v2/internal/server"
	"github.com/kaja-tools/website/v2/internal/store"
	"github.com/kaja-tools/website/v2/internal/theatre"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	theatreURL := os.Getenv("THEATRE_URL")
	if theatreURL == "" {
		theatreURL = "http://localhost:41530"
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":50053"
	}

	theatreClient := theatre.NewClient(theatreURL)
	seats := store.New(theatreClient)

	// The simulated crowd runs in-process, booking through the same store
	// the gRPC handlers use. Set CROWD=off for an empty, quiet theatre.
	if os.Getenv("CROWD") != "off" {
		go crowd.New(seats, theatreClient).Run(context.Background())
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterSeatingServer(grpcServer, server.New(seats))
	reflection.Register(grpcServer)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Starting seating gRPC server on %s (Theatre schedule at %s)", lis.Addr().String(), theatreURL)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
