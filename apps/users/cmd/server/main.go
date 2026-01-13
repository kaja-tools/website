package main

import (
	"fmt"
	"net/http"
	"os"

	"log/slog"

	"github.com/joho/godotenv"
	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/model"
	"github.com/twitchtv/twirp"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Info(".env file not loaded", "error", err)
	}

	db, err := model.OpenDB(os.Getenv("DB_DIR"))
	if err != nil {
		slog.Error("failed to open db", "error", err)
		os.Exit(1)
	}

	model := model.NewUsers(db)

	usersServer := api.NewUsersServer(api.NewUsersHandler(model), twirp.WithServerPathPrefix("/users/twirp"), twirp.WithServerHooks(api.NewLoggingServerHooks()))
	mux := http.NewServeMux()
	fmt.Printf("Handling UsersServer on %s\n", usersServer.PathPrefix())
	mux.Handle(usersServer.PathPrefix(), usersServer)
	http.ListenAndServe(":41521", mux)
}
