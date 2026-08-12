package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kaja-tools/website/v2/internal/recipes"
	"github.com/kaja-tools/website/v2/openapi"
)

// Server serves the bakery's book at the root of its host
// (e.g. https://bakebook.kaja.tools).
type Server struct {
	// baseURL is the public base URL of the service, used in error details so
	// a 404 can say where the full list is.
	baseURL string
}

func New(baseURL string) *Server {
	return &Server{baseURL: baseURL}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.yaml", s.getSpec)
	mux.HandleFunc("GET /cookies", s.listCookies)
	mux.HandleFunc("GET /cookies/{cookieId}", s.getCookie)
	return logRequests(mux)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path)
	})
}

type cookie struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Texture             string   `json:"texture"`
	Tagline             string   `json:"tagline"`
	Description         string   `json:"description"`
	BakeMinutes         int      `json:"bakeMinutes"`
	OvenCelsius         int      `json:"ovenCelsius"`
	PerTray             int      `json:"perTray"`
	DoughGramsPerCookie int      `json:"doughGramsPerCookie"`
	PriceCents          int      `json:"priceCents"`
	Allergens           []string `json:"allergens"`
}

func render(c recipes.Cookie) cookie {
	return cookie{
		ID:                  c.ID,
		Name:                c.Name,
		Texture:             string(c.Texture),
		Tagline:             c.Tagline,
		Description:         c.Description,
		BakeMinutes:         c.BakeMinutes,
		OvenCelsius:         c.OvenCelsius,
		PerTray:             c.PerTray,
		DoughGramsPerCookie: c.DoughGramsPerCookie,
		PriceCents:          c.PriceCents,
		Allergens:           c.Allergens,
	}
}

func (s *Server) getSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openapi.Spec)
}

// listCookies returns the whole book. There is no pagination and no
// filtering: seven recipes fit in one response.
func (s *Server) listCookies(w http.ResponseWriter, r *http.Request) {
	out := []cookie{}
	for _, c := range recipes.Cookies() {
		out = append(out, render(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getCookie(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("cookieId")
	c, ok := recipes.ByID(id)
	if !ok {
		problem(w, http.StatusNotFound, "We don't bake that",
			fmt.Sprintf("no cookie %q — the whole book is at %s/cookies", id, s.baseURL))
		return
	}
	writeJSON(w, http.StatusOK, render(c))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// problem writes an RFC 7807 problem+json error response.
func problem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  title,
		"status": status,
		"detail": detail,
	})
}
