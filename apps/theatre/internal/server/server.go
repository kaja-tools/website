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
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kaja-tools/website/v2/internal/catalog"
	"github.com/kaja-tools/website/v2/openapi"
)

// Server publishes the Theatre's three lists at the root of its host
// (e.g. https://theatre.kaja.tools).
//
// The lists are the movie catalog, the theaters, and the schedule that
// relates them. The schedule carries the two ids and nothing else, so seeing
// what is on somewhere means joining it against the other two.
type Server struct {
	now func() time.Time
}

func New() *Server {
	return &Server{now: time.Now}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /openapi.yaml", s.getSpec)
	mux.HandleFunc("GET /movies", s.listMovies)
	mux.HandleFunc("GET /theaters", s.listTheaters)
	mux.HandleFunc("GET /shows", s.listShows)
	return logRequests(compress(mux))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path)
	})
}

// compress gzips a list for callers that asked for it. A page is small, but a
// caller walking the whole catalog fetches a dozen of them and gets about a
// fifth of the bytes this way.
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

func (s *Server) getSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openapi.Spec)
}

// A hundred rows is a page and five hundred is the most a caller can ask for
// in one, which is what makes walking a list of a few thousand a handful of
// requests rather than a hundred.
const (
	defaultLimit = 100
	maxLimit     = 500
	// The most ids one ?ids= lookup may name. A page of screenings names at
	// most a page of films, so this is the same number as maxLimit.
	maxIDs = maxLimit
)

// listMovies is the catalog: every film the chain has a print of, whether or
// not it is on this week. Everything a screening does not say about a film is
// here, which makes this the other half of every join.
func (s *Server) listMovies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, after, err := s.pageRequest(query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ids, err := parseIDs(query.Get("ids"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	genre := query.Get("genre")

	movies := []catalog.Movie{}
	for _, m := range catalog.Movies() {
		if ids != nil && !ids[m.ID] {
			continue
		}
		if genre != "" && m.Genre != genre {
			continue
		}
		movies = append(movies, m)
	}

	// The catalog is already in id order, which is the order a cursor walks.
	rows, next := paginate(movies, func(m catalog.Movie) string { return m.ID }, limit, after, s.anchor(after))
	writeJSON(w, moviePage{Movies: rows, Total: len(movies), NextCursor: next})
}

// listTheaters is the chain. A dozen houses is small enough to be one
// response, so this is the one list with no cursor: read it once and keep it
// as the lookup table it is.
func (s *Server) listTheaters(w http.ResponseWriter, r *http.Request) {
	city := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("city")))

	theaters := []catalog.Theater{}
	for _, t := range catalog.Theaters() {
		if city != "" && strings.ToLower(t.City) != city {
			continue
		}
		theaters = append(theaters, t)
	}
	writeJSON(w, theaterList{Theaters: theaters, Total: len(theaters)})
}

// listShows is the schedule and only the schedule: which film plays at which
// house, when, and for how much. What the film is called and where the house
// is are the other two lists' to say.
func (s *Server) listShows(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, after, err := s.pageRequest(query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	theaterID, movieID := query.Get("theaterId"), query.Get("movieId")
	city := strings.ToLower(strings.TrimSpace(query.Get("city")))

	// Every page of one walk is rendered against the time the walk started,
	// so a screening cannot roll over into next week half way through and be
	// served twice.
	anchor := s.anchor(after)

	shows := []show{}
	for _, c := range catalog.Shows() {
		if theaterID != "" && c.TheaterID != theaterID {
			continue
		}
		if movieID != "" && c.MovieID != movieID {
			continue
		}
		if city != "" {
			t, ok := catalog.TheaterByID(c.TheaterID)
			if !ok || strings.ToLower(t.City) != city {
				continue
			}
		}
		shows = append(shows, render(c, anchor))
	}
	sort.Slice(shows, func(i, j int) bool { return shows[i].key() < shows[j].key() })

	rows, next := paginate(shows, show.key, limit, after, anchor)
	writeJSON(w, showPage{Shows: rows, Total: len(shows), NextCursor: next})
}

// show is one screening as the schedule publishes it: the two ids it relates,
// the next time it starts, and what a seat costs.
type show struct {
	ID         string    `json:"id"`
	MovieID    string    `json:"movieId"`
	TheaterID  string    `json:"theaterId"`
	StartsAt   time.Time `json:"startsAt"`
	PriceCents int       `json:"priceCents"`
}

func render(c catalog.Show, now time.Time) show {
	return show{
		ID:         c.ID,
		MovieID:    c.MovieID,
		TheaterID:  c.TheaterID,
		StartsAt:   c.NextStart(now),
		PriceCents: c.PriceCents,
	}
}

// The schedule's order: soonest first, and by id where two screenings start
// in the same minute, so that a position in it is a screening rather than a
// number. Zero-padded because a cursor compares these as text.
func (s show) key() string {
	return fmt.Sprintf("%011d:%s", s.StartsAt.Unix(), s.ID)
}

type moviePage struct {
	Movies     []catalog.Movie `json:"movies"`
	Total      int             `json:"total"`
	NextCursor *string         `json:"nextCursor"`
}

type theaterList struct {
	Theaters []catalog.Theater `json:"theaters"`
	Total    int               `json:"total"`
}

type showPage struct {
	Shows      []show  `json:"shows"`
	Total      int     `json:"total"`
	NextCursor *string `json:"nextCursor"`
}

// paginate takes the rows after the cursor, up to limit, and says where to
// carry on from. items must already be sorted by key.
func paginate[T any](items []T, key func(T) string, limit int, after *cursor, anchor time.Time) ([]T, *string) {
	if after != nil {
		items = items[sort.Search(len(items), func(i int) bool { return key(items[i]) > after.key }):]
	}
	if len(items) <= limit {
		return items, nil
	}
	items = items[:limit]
	next := cursor{anchor: anchor, key: key(items[limit-1])}.encode()
	return items, &next
}

func (s *Server) pageRequest(query url.Values) (int, *cursor, error) {
	limit, err := parseLimit(query.Get("limit"))
	if err != nil {
		return 0, nil, err
	}
	after, err := parseCursor(query.Get("cursor"), s.now().UTC())
	if err != nil {
		return 0, nil, err
	}
	return limit, after, nil
}

// anchor is the moment a walk is rendered against: the one the cursor
// carries, or now for a walk that is just starting.
func (s *Server) anchor(after *cursor) time.Time {
	if after != nil {
		return after.anchor
	}
	return s.now().UTC()
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

// parseIDs reads the ?ids= filter. A nil set means the caller did not ask for
// particular rows, which is different from asking for none.
func parseIDs(raw string) (map[string]bool, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	ids := map[string]bool{}
	for _, id := range strings.Split(raw, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids[id] = true
		}
	}
	if len(ids) > maxIDs {
		return nil, fmt.Errorf("ids names %d films; %d is the most in one lookup", len(ids), maxIDs)
	}
	return ids, nil
}

// A cursor is where the last page stopped: the sort key of its final row,
// plus the moment that walk began. It is opaque on purpose — it says where a
// walk got to rather than how far in it got, so a list growing underneath a
// caller neither skips a row nor repeats one.
type cursor struct {
	anchor time.Time
	key    string
}

// How long a walk may take before it has to start again. A cursor carries the
// time its walk began, and past this the times in it would be stale.
const cursorTTL = time.Hour

func (c cursor) encode() string {
	raw := fmt.Sprintf("%d:%s", c.anchor.Unix(), c.key)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func parseCursor(raw string, now time.Time) (*cursor, error) {
	if raw == "" {
		return nil, nil
	}
	invalid := errors.New("cursor is not one this service issued — start again without one")
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, invalid
	}
	unix, key, found := strings.Cut(string(decoded), ":")
	if !found {
		return nil, invalid
	}
	anchor, err := strconv.ParseInt(unix, 10, 64)
	if err != nil {
		return nil, invalid
	}
	c := cursor{anchor: time.Unix(anchor, 0).UTC(), key: key}
	if now.Sub(c.anchor) > cursorTTL || c.anchor.After(now) {
		return nil, errors.New("cursor has expired — start again without one")
	}
	return &c, nil
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}
