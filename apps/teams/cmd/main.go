package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/model"
	"github.com/kaja-tools/website/v2/internal/users"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf(".env file not loaded: %v", err)
	}

	// Get DB_DIR from environment
	dbDir := os.Getenv("DB_DIR")
	if dbDir == "" {
		log.Fatal("DB_DIR environment variable is required")
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Create PebbleDB storage
	db, err := model.OpenDB(dbDir)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer db.Close()

	// Create teams model
	teams := model.NewTeams(db)

	// Create users client for cross-service validation
	usersServiceURL := os.Getenv("USERS_SERVICE_URL")
	if usersServiceURL == "" {
		usersServiceURL = "http://users-service" // Default k8s service name
	}
	usersClient := users.NewClient(usersServiceURL)

	// Create gRPC server
	srv := api.NewTeamsServer(teams, usersClient)

	// Create TCP listener
	lis, err := net.Listen("tcp", ":50052") // Different port from users service
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	api.RegisterTeamsServer(grpcServer, srv)
	reflection.Register(grpcServer)

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
