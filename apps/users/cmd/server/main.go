package main

import (
	"fmt"
	"net/http"

	users "github.com/kaja-tools/website/v2/internal/users"
	"github.com/twitchtv/twirp"
)

func main() {
	usersServer := users.NewUsersServer(users.NewUsersServerPebble("../build/users.db"), twirp.WithServerHooks(users.NewLoggingServerHooks()))
	mux := http.NewServeMux()
	fmt.Printf("Handling UsersServer on %s\n", usersServer.PathPrefix())
	mux.Handle(usersServer.PathPrefix(), usersServer)
	http.ListenAndServe(":41521", mux)
}
