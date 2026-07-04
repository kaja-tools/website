package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/kaja-tools/website/v2/internal/server"
)

func main() {
	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://kaja.tools/theatre"
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":41530"
	}

	s := server.New(baseURL)
	slog.Info("theatre listening", "addr", addr)
	if err := http.ListenAndServe(addr, s.Handler()); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
