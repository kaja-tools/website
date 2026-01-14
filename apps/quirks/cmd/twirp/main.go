package main

import (
	"fmt"
	"net/http"

	"log/slog"

	"github.com/joho/godotenv"
	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/twitchtv/twirp"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Info(".env file not loaded", "error", err)
	}

	basicsServer := api.NewBasicsServer(&api.BasicsService{}, twirp.WithServerPathPrefix("/twirp-quirks/twirp"))
	quirksServer := api.NewQuirksServer(&api.QuirksService{}, twirp.WithServerPathPrefix("/twirp-quirks/twirp"))
	quirks_2Server := api.NewQuirks_2Server(&api.Quirks_2Service{}, twirp.WithServerPathPrefix("/twirp-quirks/twirp"))

	mux := http.NewServeMux()
	fmt.Printf("Handling BasicsServer on %s\n", basicsServer.PathPrefix())
	mux.Handle(basicsServer.PathPrefix(), basicsServer)
	fmt.Printf("Handling QuirksServer on %s\n", quirksServer.PathPrefix())
	mux.Handle(quirksServer.PathPrefix(), quirksServer)
	fmt.Printf("Handling Quirks_2Server on %s\n", quirks_2Server.PathPrefix())
	mux.Handle(quirks_2Server.PathPrefix(), quirks_2Server)
	http.ListenAndServe(":41523", mux)
}
