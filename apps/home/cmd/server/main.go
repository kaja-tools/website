package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	staticDir := "./static"
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		staticDir = dir
	}

	fs := http.FileServer(http.Dir(staticDir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)

		// Check if the file exists
		fullPath := filepath.Join(staticDir, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if path != "/" && filepath.Ext(path) == "" {
				// Check if a matching .html file exists
				htmlPath := filepath.Join(staticDir, path+".html")
				if _, err := os.Stat(htmlPath); err == nil {
					http.ServeFile(w, r, htmlPath)
					return
				}
				// Serve index.html for SPA-like behavior
				http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
				return
			}
		}

		fs.ServeHTTP(w, r)
	})

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
