package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/wham/website/apps/teams/internal/api"
	"github.com/wham/website/apps/teams/internal/model"
	"google.golang.org/grpc"
)

func main() {
	// Create PebbleDB storage
	dbPath := filepath.Join("data", "teams.db")
	db, err := model.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer db.Close()

	// Create teams model
	teams := model.NewTeams(db)

	// Create gRPC server
	srv := api.NewTeamsServer(teams)

	// Create TCP listener
	lis, err := net.Listen("tcp", ":50052") // Different port from users service
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	api.RegisterTeamsServer(grpcServer, srv)

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
