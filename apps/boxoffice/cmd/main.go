package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/crowd"
	"github.com/kaja-tools/website/v2/internal/office"
	"github.com/kaja-tools/website/v2/internal/theatre"
	"github.com/twitchtv/twirp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := env("ADDR", ":41531")
	theatreURL := env("THEATRE_URL", "http://localhost:41530/theatre")
	publicTheatreURL := env("PUBLIC_THEATRE_URL", "https://theatre.kaja.tools/theatre")
	seatingAddr := env("SEATING_ADDR", "localhost:50053")

	conn, err := grpc.NewClient(seatingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to create seating client", "error", err)
		os.Exit(1)
	}
	seating := api.NewSeatingClient(conn)
	theatreClient := theatre.NewClient(theatreURL)

	if env("CROWD", "on") != "off" {
		go crowd.New(seating, theatreClient).Run(context.Background())
	}

	boxOffice := office.New(seating, theatreClient, publicTheatreURL)
	server := api.NewBoxOfficeServer(boxOffice, twirp.WithServerPathPrefix("/boxoffice/twirp"))

	mux := http.NewServeMux()
	fmt.Printf("Handling BoxOfficeServer on %s\n", server.PathPrefix())
	mux.Handle(server.PathPrefix(), server)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
