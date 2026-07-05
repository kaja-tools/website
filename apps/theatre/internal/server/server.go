package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/kaja-tools/website/v2/internal/catalog"
	"github.com/kaja-tools/website/v2/openapi"
)

// Server exposes the theatre catalog under the /theatre path prefix, so the
// same paths work whether reached directly or via the public hostname.
type Server struct {
	// baseURL is the public URL of the /theatre prefix, used to build
	// absolute posterUrl links.
	baseURL string
	now     func() time.Time
}

func New(baseURL string) *Server {
	return &Server{baseURL: baseURL, now: time.Now}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /theatre/openapi.yaml", s.getSpec)
	mux.HandleFunc("GET /theatre/venue", s.getVenue)
	mux.HandleFunc("GET /theatre/events", s.listEvents)
	mux.HandleFunc("GET /theatre/events/{eventId}", s.getEvent)
	mux.HandleFunc("GET /theatre/events/{eventId}/poster.svg", s.getPoster)
	mux.HandleFunc("GET /theatre/performances/{performanceId}", s.getPerformance)
	return logRequests(mux)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path)
	})
}

type eventSummary struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Genre             string  `json:"genre"`
	Tagline           string  `json:"tagline"`
	PosterURL         string  `json:"posterUrl"`
	NextPerformanceID *string `json:"nextPerformanceId"`
}

type eventFull struct {
	eventSummary
	Description  string                `json:"description"`
	Details      catalog.Details       `json:"details"`
	Performances []catalog.Performance `json:"performances"`
}

type eventPage struct {
	Items      []eventSummary `json:"items"`
	Page       int            `json:"page"`
	PerPage    int            `json:"perPage"`
	TotalItems int            `json:"totalItems"`
	TotalPages int            `json:"totalPages"`
}

func (s *Server) summary(e catalog.Event, now time.Time) eventSummary {
	return eventSummary{
		ID:                e.ID,
		Title:             e.Title,
		Genre:             e.Genre,
		Tagline:           e.Tagline,
		PosterURL:         fmt.Sprintf("%s/events/%s/poster.svg", s.baseURL, e.ID),
		NextPerformanceID: catalog.NextPerformanceID(e, now),
	}
}

func (s *Server) getSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openapi.Spec)
}

func (s *Server) getVenue(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, catalog.TheVenue())
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	now := s.now().UTC()
	q := r.URL.Query()

	genre := q.Get("genre")
	switch genre {
	case "", "concert", "play", "comedy", "opera":
	default:
		problem(w, http.StatusBadRequest, "Invalid genre",
			fmt.Sprintf("%q is not one of: concert, play, comedy, opera", genre))
		return
	}

	var date *time.Time
	if d := q.Get("date"); d != "" {
		parsed, err := time.ParseInLocation("2006-01-02", d, time.UTC)
		if err != nil {
			problem(w, http.StatusBadRequest, "Invalid date", "date must be formatted YYYY-MM-DD")
			return
		}
		date = &parsed
	}

	page, err := positiveInt(q.Get("page"), 1)
	if err != nil {
		problem(w, http.StatusBadRequest, "Invalid page", "page must be a positive integer")
		return
	}
	perPage, err := positiveInt(q.Get("perPage"), 10)
	if err != nil || perPage > 50 {
		problem(w, http.StatusBadRequest, "Invalid perPage", "perPage must be between 1 and 50")
		return
	}

	var filtered []catalog.Event
	for _, e := range catalog.Events() {
		if genre != "" && e.Genre != genre {
			continue
		}
		if date != nil && !performsOn(e, *date, now) {
			continue
		}
		filtered = append(filtered, e)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Title < filtered[j].Title })

	totalItems := len(filtered)
	totalPages := (totalItems + perPage - 1) / perPage
	start := (page - 1) * perPage
	end := min(start+perPage, totalItems)
	items := []eventSummary{}
	if start < totalItems {
		for _, e := range filtered[start:end] {
			items = append(items, s.summary(e, now))
		}
	}

	links := ""
	if page < totalPages {
		links = fmt.Sprintf(`<%s/events?page=%d&perPage=%d>; rel="next"`, s.baseURL, page+1, perPage)
	}
	if page > 1 {
		if links != "" {
			links += ", "
		}
		links += fmt.Sprintf(`<%s/events?page=%d&perPage=%d>; rel="prev"`, s.baseURL, page-1, perPage)
	}
	if links != "" {
		w.Header().Set("Link", links)
	}

	writeJSON(w, http.StatusOK, eventPage{
		Items:      items,
		Page:       page,
		PerPage:    perPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
	})
}

func performsOn(e catalog.Event, date time.Time, now time.Time) bool {
	for _, p := range catalog.PerformancesFor(e, now) {
		y1, m1, d1 := p.StartsAt.Date()
		y2, m2, d2 := date.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 {
			return true
		}
	}
	return false
}

func (s *Server) getEvent(w http.ResponseWriter, r *http.Request) {
	now := s.now().UTC()
	e, ok := catalog.EventByID(r.PathValue("eventId"))
	if !ok {
		problem(w, http.StatusNotFound, "Event not found",
			fmt.Sprintf("no event %q — list them at %s/events", r.PathValue("eventId"), s.baseURL))
		return
	}
	writeJSON(w, http.StatusOK, eventFull{
		eventSummary: s.summary(e, now),
		Description:  e.Description,
		Details:      e.Details,
		Performances: catalog.PerformancesFor(e, now),
	})
}

func (s *Server) getPoster(w http.ResponseWriter, r *http.Request) {
	e, ok := catalog.EventByID(r.PathValue("eventId"))
	if !ok {
		problem(w, http.StatusNotFound, "Event not found", "")
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(poster(e))
}

func (s *Server) getPerformance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("performanceId")
	p, result := catalog.PerformanceByID(id, s.now())
	switch result {
	case catalog.LookupFound:
		writeJSON(w, http.StatusOK, p)
	case catalog.LookupGone:
		problem(w, http.StatusGone, "Performance already happened",
			fmt.Sprintf("%s is in the past; pick an upcoming one from %s/events", id, s.baseURL))
	default:
		problem(w, http.StatusNotFound, "Performance not found",
			fmt.Sprintf("%q does not match any showtime in the next %d days", id, catalog.WindowDays))
	}
}

func positiveInt(raw string, def int) (int, error) {
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid: %q", raw)
	}
	return n, nil
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
