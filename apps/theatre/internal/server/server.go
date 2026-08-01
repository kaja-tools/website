package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/kaja-tools/website/v2/internal/catalog"
	"github.com/kaja-tools/website/v2/openapi"
)

// Server exposes the theatre catalog at the root of its host
// (e.g. https://theatre.kaja.tools).
type Server struct {
	// baseURL is the public base URL of the service, used to build absolute
	// posterUrl links.
	baseURL string
	now     func() time.Time
}

func New(baseURL string) *Server {
	return &Server{baseURL: baseURL, now: time.Now}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.yaml", s.getSpec)
	mux.HandleFunc("GET /shows", s.listShows)
	mux.HandleFunc("GET /shows/{showId}", s.getShow)
	mux.HandleFunc("GET /shows/{showId}/poster.svg", s.getPoster)
	return logRequests(mux)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path)
	})
}

type show struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Genre          string    `json:"genre"`
	Tagline        string    `json:"tagline"`
	Description    string    `json:"description"`
	StartsAt       time.Time `json:"startsAt"`
	BasePriceCents int       `json:"basePriceCents"`
	PosterURL      string    `json:"posterUrl"`
}

func (s *Server) render(c catalog.Show, now time.Time) show {
	return show{
		ID:             c.ID,
		Title:          c.Title,
		Genre:          c.Genre,
		Tagline:        c.Tagline,
		Description:    c.Description,
		StartsAt:       c.NextStart(now),
		BasePriceCents: c.BasePriceCents,
		PosterURL:      fmt.Sprintf("%s/shows/%s/poster.svg", s.baseURL, c.ID),
	}
}

func (s *Server) getSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openapi.Spec)
}

// listShows returns the whole repertoire, soonest first. There is no
// pagination and no filtering: seven shows fit in one response.
func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	now := s.now().UTC()
	out := []show{}
	for _, c := range catalog.Shows() {
		out = append(out, s.render(c, now))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getShow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("showId")
	c, ok := catalog.ByID(id)
	if !ok {
		problem(w, http.StatusNotFound, "Show not found",
			fmt.Sprintf("no show %q — list them all at %s/shows", id, s.baseURL))
		return
	}
	writeJSON(w, http.StatusOK, s.render(c, s.now().UTC()))
}

func (s *Server) getPoster(w http.ResponseWriter, r *http.Request) {
	c, ok := catalog.ByID(r.PathValue("showId"))
	if !ok {
		problem(w, http.StatusNotFound, "Show not found", "")
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(poster(c))
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
