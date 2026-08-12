package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/bakebook"
	"github.com/kaja-tools/website/v2/internal/crowd"
	"github.com/kaja-tools/website/v2/internal/server"
	"github.com/kaja-tools/website/v2/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	bakebookURL := os.Getenv("BAKEBOOK_URL")
	if bakebookURL == "" {
		bakebookURL = "http://localhost:41530"
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":50053"
	}

	book := bakebook.NewClient(bakebookURL)
	oven := store.New(book, store.DefaultPace())

	// The rest of the bakery runs in-process, baking through the same store
	// the gRPC handlers use. Set CROWD=off for an empty, quiet oven.
	if os.Getenv("CROWD") != "off" {
		go crowd.New(oven, book).Run(context.Background())
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterOvenServer(grpcServer, server.New(oven))
	reflection.Register(grpcServer)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Starting oven gRPC server on %s (bakebook at %s)", lis.Addr().String(), bakebookURL)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
