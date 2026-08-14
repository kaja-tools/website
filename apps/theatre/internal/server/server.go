package server

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kaja-tools/website/v2/internal/catalog"
	"github.com/kaja-tools/website/v2/openapi"
)

// Server exposes the theatre programme at the root of its host
// (e.g. https://theatre.kaja.tools).
//
// There is one operation. It comes back a page at a time: call it with no
// parameters for the first page and follow nextCursor until it is null for
// the rest.
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

// compress gzips the programme for callers that asked for it. A page of it is
// small, but a caller walking the whole week asks for the largest page it can
// and gets about a fifth of the bytes this way.
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

// How much of the programme comes back at once, and the most a caller can ask
// for in one page.
const (
	defaultLimit = 100
	maxLimit     = 500
)

// page is one response: a slice of the programme and the cursor that continues
// it. nextCursor is null on the last page, which is the only stopping
// condition a caller needs.
type page struct {
	Shows      []show  `json:"shows"`
	NextCursor *string `json:"nextCursor"`
}

// listShows returns the programme a page at a time, soonest first. Called with
// no parameters it answers the first page, so a caller still needs to have
// read nothing first; ?limit= sizes the page and ?cursor= continues it.
func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := s.now().UTC()
	after, err := parseCursor(r.URL.Query().Get("cursor"), now)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Every page of one walk is rendered against the time the walk started, so
	// a screening cannot roll over into next week half way through and be
	// served twice.
	anchor := now
	if after != nil {
		anchor = after.anchor
	}

	shows := []show{}
	for _, c := range catalog.Shows() {
		shows = append(shows, render(c, anchor))
	}
	sort.Slice(shows, func(i, j int) bool { return before(shows[i], shows[j]) })

	if after != nil {
		shows = shows[sort.Search(len(shows), func(i int) bool { return after.precedes(shows[i]) }):]
	}
	out := page{Shows: shows}
	if len(shows) > limit {
		out.Shows = shows[:limit]
		next := cursor{anchor: anchor, startsAt: out.Shows[limit-1].StartsAt, id: out.Shows[limit-1].ID}.encode()
		out.NextCursor = &next
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(out)
}

// The programme's order: soonest first, and by id where two films start at the
// same minute, so that a position in it is a screening rather than a number.
func before(a, b show) bool {
	if !a.StartsAt.Equal(b.StartsAt) {
		return a.StartsAt.Before(b.StartsAt)
	}
	return a.ID < b.ID
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, fmt.Errorf("limit must be a whole number between 1 and %d", maxLimit)
	}
	return limit, nil
}

// A cursor is the screening the last page ended on, plus the time that page
// was rendered against. It is opaque on purpose: it says where a walk got to,
// not how far in it got, so the programme growing underneath a caller neither
// skips a screening nor repeats one.
type cursor struct {
	anchor   time.Time
	startsAt time.Time
	id       string
}

// How long a walk may take before it has to start again. A cursor carries the
// time its walk began, and past this the times in it would be stale.
const cursorTTL = time.Hour

func (c cursor) encode() string {
	raw := fmt.Sprintf("%d:%d:%s", c.anchor.Unix(), c.startsAt.Unix(), c.id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// precedes reports whether s comes after the cursor in the programme's order.
func (c cursor) precedes(s show) bool {
	if !s.StartsAt.Equal(c.startsAt) {
		return s.StartsAt.After(c.startsAt)
	}
	return s.ID > c.id
}

func parseCursor(raw string, now time.Time) (*cursor, error) {
	if raw == "" {
		return nil, nil
	}
	invalid := errors.New("cursor is not one this service issued — start again at /shows")
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, invalid
	}
	parts := strings.SplitN(string(decoded), ":", 3)
	if len(parts) != 3 {
		return nil, invalid
	}
	anchor, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, invalid
	}
	startsAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, invalid
	}
	c := cursor{anchor: time.Unix(anchor, 0).UTC(), startsAt: time.Unix(startsAt, 0).UTC(), id: parts[2]}
	if now.Sub(c.anchor) > cursorTTL || c.anchor.After(now) {
		return nil, errors.New("cursor has expired — start again at /shows")
	}
	return &c, nil
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
