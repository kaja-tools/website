package main

import (
	"log/slog"
	"net/http"
	"os"

	// Showtimes are published in each house's own local time, and the runtime
	// image has no zone database of its own.
	_ "time/tzdata"

	"github.com/kaja-tools/website/v2/internal/server"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":41530"
	}

	s := server.New()
	slog.Info("theatre listening", "addr", addr)
	if err := http.ListenAndServe(addr, s.Handler()); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
