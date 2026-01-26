package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaja-tools/website/v2/internal/api"
	v2 "github.com/kaja-tools/website/v2/internal/api/v2"
	"google.golang.org/grpc"
)

func main() {
	// Create gRPC server
	grpcServer := grpc.NewServer()
	api.RegisterBasicsServer(grpcServer, &api.BasicsService{})
	api.RegisterQuirksServer(grpcServer, &api.QuirksService{})
	api.RegisterQuirks_2Server(grpcServer, &api.Quirks_2Service{})
	v2.RegisterQuirksServer(grpcServer, &v2.QuirksService{})

	// Create TCP listener
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Handle shutdown gracefully
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	// Start server
	log.Printf("Starting gRPC server on %s", lis.Addr().String())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
