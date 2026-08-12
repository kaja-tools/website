package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/kaja-tools/website/v2/internal/bakebook"
	"github.com/kaja-tools/website/v2/internal/mcp"
	"github.com/kaja-tools/website/v2/internal/tools"
)

const version = "1.0.0"

func main() {
	bakebookURL := os.Getenv("BAKEBOOK_URL")
	if bakebookURL == "" {
		bakebookURL = "http://localhost:41530"
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":41540"
	}

	book := bakebook.NewClient(bakebookURL)
	server := mcp.NewServer("The Kaja Bakery Kitchen", version, tools.All(book))

	mux := http.NewServeMux()
	mux.Handle("/mcp", server)
	// Anybody who opens the host in a browser gets told where the server is,
	// rather than a 404 that reads as "nothing is running here".
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("The Kaja Bakery kitchen speaks MCP at /mcp.\n" +
			"Recipes are at https://bakebook.kaja.tools/cookies.\n"))
	})

	slog.Info("kitchen listening", "addr", addr, "bakebook", bakebookURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
