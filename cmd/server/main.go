package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://github.com/wham/kaja", http.StatusFound)
	})

	http.HandleFunc("/demo/", func(w http.ResponseWriter, r *http.Request) {
		// Remove /demo prefix from path before proxying
		r.URL.Path = r.URL.Path[len("/demo"):]
		proxy := &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Scheme = "http"
				r.URL.Host = "localhost:41520"
			},
		}
		proxy.ServeHTTP(w, r)
	})

	fmt.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
