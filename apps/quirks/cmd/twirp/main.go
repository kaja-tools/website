package main

import (
	"fmt"
	"net/http"

	v1 "github.com/kaja-tools/website/v2/internal/api/v1"
	v2 "github.com/kaja-tools/website/v2/internal/api/v2"
	"github.com/twitchtv/twirp"
)

// headerMiddleware wraps an http.Handler and injects HTTP headers into the Twirp context
func headerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Filter out reserved headers that Twirp doesn't allow
		headers := make(http.Header)
		for key, values := range r.Header {
			// Skip reserved headers (Content-Type, Accept, Twirp-specific headers)
			switch key {
			case "Content-Type", "Accept", "Content-Length", "Transfer-Encoding":
				continue
			}
			headers[key] = values
		}
		ctx, err := twirp.WithHTTPRequestHeaders(r.Context(), headers)
		if err != nil {
			fmt.Printf("Error setting headers: %v\n", err)
			ctx = r.Context()
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	basicsServer := v1.NewBasicsServer(&v1.BasicsService{}, twirp.WithServerPathPrefix("/twirp-quirks/twirp"))
	quirksServer := v1.NewQuirksServer(&v1.QuirksService{}, twirp.WithServerPathPrefix("/twirp-quirks/twirp"))
	quirks_2Server := v1.NewQuirks_2Server(&v1.Quirks_2Service{}, twirp.WithServerPathPrefix("/twirp-quirks/twirp"))
	quirksV2Server := v2.NewQuirksServer(&v2.QuirksService{}, twirp.WithServerPathPrefix("/twirp-quirks/twirp"))

	mux := http.NewServeMux()
	fmt.Printf("Handling BasicsServer on %s\n", basicsServer.PathPrefix())
	mux.Handle(basicsServer.PathPrefix(), headerMiddleware(basicsServer))
	fmt.Printf("Handling QuirksServer on %s\n", quirksServer.PathPrefix())
	mux.Handle(quirksServer.PathPrefix(), headerMiddleware(quirksServer))
	fmt.Printf("Handling Quirks_2Server on %s\n", quirks_2Server.PathPrefix())
	mux.Handle(quirks_2Server.PathPrefix(), headerMiddleware(quirks_2Server))
	fmt.Printf("Handling QuirksV2Server on %s\n", quirksV2Server.PathPrefix())
	mux.Handle(quirksV2Server.PathPrefix(), headerMiddleware(quirksV2Server))
	http.ListenAndServe(":41523", mux)
}
