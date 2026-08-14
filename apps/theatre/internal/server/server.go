package server

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/kaja-tools/website/v2/internal/catalog"
	"github.com/kaja-tools/website/v2/openapi"
)

// Server exposes the theatre programme at the root of its host
// (e.g. https://theatre.kaja.tools).
//
// There is one operation. The whole programme comes back in one response, so
// a second call to fetch one screening of it would only be a way of asking
// for less.
type Server struct {
	now func() time.Time
}

func New() *Server {
	return &Server{now: time.Now}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.yaml", s.getSpec)
	mux.HandleFunc("GET /shows", s.listShows)
	return logRequests(compress(mux))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path)
	})
}

// compress gzips the programme for callers that asked for it. The whole week
// in one response is half a megabyte of JSON and about a fifth of that
// compressed, which is what keeps "no parameters" an easy promise to keep.
func compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		zip := gzip.NewWriter(w)
		defer zip.Close()
		next.ServeHTTP(gzipResponse{ResponseWriter: w, writer: zip}, r)
	})
}

type gzipResponse struct {
	http.ResponseWriter
	writer io.Writer
}

func (g gzipResponse) Write(b []byte) (int, error) { return g.writer.Write(b) }

type show struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Director       string    `json:"director"`
	Year           int       `json:"year"`
	RuntimeMinutes int       `json:"runtimeMinutes"`
	Genre          string    `json:"genre"`
	Language       string    `json:"language"`
	Synopsis       string    `json:"synopsis"`
	StartsAt       time.Time `json:"startsAt"`
	BasePriceCents int       `json:"basePriceCents"`
}

func render(c catalog.Show, now time.Time) show {
	return show{
		ID:             c.ID,
		Title:          c.Title,
		Director:       c.Director,
		Year:           c.Year,
		RuntimeMinutes: c.RuntimeMinutes,
		Genre:          c.Genre,
		Language:       c.Language,
		Synopsis:       c.Synopsis,
		StartsAt:       c.NextStart(now),
		BasePriceCents: c.BasePriceCents,
	}
}

func (s *Server) getSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openapi.Spec)
}

// listShows returns the whole programme, soonest first. No pagination, no
// filtering, no parameters: whatever the repertory has grown to, a caller
// gets all of it and needs to have read nothing first.
func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	now := s.now().UTC()
	out := []show{}
	for _, c := range catalog.Shows() {
		out = append(out, render(c, now))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(out)
}
