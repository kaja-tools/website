package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/kaja-tools/website/v2/internal/concierge"
	"github.com/kaja-tools/website/v2/internal/mcp"
)

func main() {
	theatreURL := os.Getenv("THEATRE_URL")
	if theatreURL == "" {
		theatreURL = "http://localhost:41530"
	}
	seatingTarget := os.Getenv("SEATING_TARGET")
	if seatingTarget == "" {
		seatingTarget = "localhost:50053"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":41540"
	}

	house, err := concierge.NewHouse(theatreURL, seatingTarget)
	if err != nil {
		slog.Error("could not reach the house", "error", err)
		os.Exit(1)
	}

	server := mcp.NewServer("The Kaja Theatre Concierge", "1.0.0", concierge.Instructions(), concierge.Tools(house))

	mux := http.NewServeMux()
	// The endpoint is /mcp because that is where every client looks first.
	mux.Handle("/mcp", server)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://kaja.tools", http.StatusFound)
	})

	slog.Info("concierge listening", "addr", addr, "theatre", theatreURL, "seating", seatingTarget)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
