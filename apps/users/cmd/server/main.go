package main

import (
	"fmt"
	"net/http"
	"os"

	"log/slog"

	"github.com/joho/godotenv"
	users "github.com/kaja-tools/website/v2/internal/users"
	"github.com/twitchtv/twirp"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Info(".env file not loaded", "error", err)
	}

	usersServer := users.NewUsersServer(users.NewUsersServerPebble(os.Getenv("DB_DIR")), twirp.WithServerHooks(users.NewLoggingServerHooks()))
	mux := http.NewServeMux()
	fmt.Printf("Handling UsersServer on %s\n", usersServer.PathPrefix())
	mux.Handle(usersServer.PathPrefix(), usersServer)
	http.ListenAndServe(":41521", mux)
}
