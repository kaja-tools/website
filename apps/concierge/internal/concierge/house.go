// Package concierge is the front of house: it reads the Theatre service's
// three lists over HTTP and the live seat map from the seating service over
// gRPC, and turns a sentence about what you feel like into a screening and a
// row.
//
// It owns no data of its own. Everything it says is something one of the
// other two services told it a moment ago, which is the whole point — a
// concierge knows the building, not the tickets.
//
// The one thing it does own is the join. The schedule publishes a movie id, a
// theater id and a time; a person asking what is on wants a title, a
// neighbourhood and a time, so the three lists are read and put back together
// here, once every few minutes, and everything downstream sees screenings.
package concierge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"sync"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// How long the programme is cached. It changes at most once a week.
const programmeTTL = 5 * time.Minute

// The biggest page the Theatre service will hand over, so that walking a list
// of a few thousand rows is a handful of requests.
const pageSize = 500

// Movie is one film, as the catalog publishes it.
type Movie struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Director       string `json:"director"`
	Year           int    `json:"year"`
	RuntimeMinutes int    `json:"runtimeMinutes"`
	Genre          string `json:"genre"`
	Language       string `json:"language"`
	Synopsis       string `json:"synopsis"`
}

// Theater is one house, as the chain publishes it.
type Theater struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	City     string `json:"city"`
	State    string `json:"state"`
	Address  string `json:"address"`
	TimeZone string `json:"timeZone"`
}

// Where is the house written the way somebody says it out loud.
func (t Theater) Where() string { return fmt.Sprintf("%s, %s", t.Name, t.City) }

// show is one row of the schedule: two ids, a time and a price, and nothing
// else. It is never handed out — it is the left side of the join.
type show struct {
	ID         string    `json:"id"`
	MovieID    string    `json:"movieId"`
	TheaterID  string    `json:"theaterId"`
	StartsAt   time.Time `json:"startsAt"`
	PriceCents int       `json:"priceCents"`
}

// Screening is a row of the schedule with its film and its house filled in —
// the join, and the only thing the concierge has opinions about.
type Screening struct {
	ID         string
	StartsAt   time.Time
	PriceCents int
	Movie      Movie
	Theater    Theater
}

// House is the concierge's two neighbours: the Theatre service over HTTP and
// the seat map over gRPC.
type House struct {
	theatreURL string
	http       *http.Client
	seating    api.SeatingClient

	mu         sync.Mutex
	screenings []Screening
	until      time.Time
}

func NewHouse(theatreURL, seatingTarget string) (*House, error) {
	// The seating service is on the private network, which is why this is
	// plaintext: nothing outside the organization can dial it.
	conn, err := grpc.NewClient(seatingTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dialing seating at %s: %w", seatingTarget, err)
	}
	return &House{
		theatreURL: theatreURL,
		http:       &http.Client{Timeout: 5 * time.Second},
		seating:    api.NewSeatingClient(conn),
	}, nil
}

// Programme is the whole week joined up, cached for a few minutes.
func (h *House) Programme() ([]Screening, error) {
	h.mu.Lock()
	if time.Now().Before(h.until) {
		screenings := h.screenings
		h.mu.Unlock()
		return screenings, nil
	}
	h.mu.Unlock()

	screenings, err := h.readProgramme()
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	h.screenings, h.until = screenings, time.Now().Add(programmeTTL)
	h.mu.Unlock()
	return screenings, nil
}

// readProgramme fetches the three lists and joins them. A screening whose
// film or house is missing from the other two is dropped rather than
// answered for: the concierge would have nothing to say about it.
func (h *House) readProgramme() ([]Screening, error) {
	movies := map[string]Movie{}
	if err := walk(h, "/movies", func(p listPage) {
		for _, m := range p.Movies {
			movies[m.ID] = m
		}
	}); err != nil {
		return nil, err
	}

	theaters := map[string]Theater{}
	if err := walk(h, "/theaters", func(p listPage) {
		for _, t := range p.Theaters {
			theaters[t.ID] = t
		}
	}); err != nil {
		return nil, err
	}

	var screenings []Screening
	if err := walk(h, "/shows", func(p listPage) {
		for _, s := range p.Shows {
			movie, haveMovie := movies[s.MovieID]
			theater, haveTheater := theaters[s.TheaterID]
			if !haveMovie || !haveTheater {
				continue
			}
			screenings = append(screenings, Screening{
				ID:         s.ID,
				StartsAt:   s.StartsAt,
				PriceCents: s.PriceCents,
				Movie:      movie,
				Theater:    theater,
			})
		}
	}); err != nil {
		return nil, err
	}
	return screenings, nil
}

// listPage is one response from the Theatre service. The three lists differ
// only in which array they fill and whether they carry a cursor at all — the
// theaters come back in one response — so one shape reads all three.
type listPage struct {
	Movies     []Movie   `json:"movies"`
	Theaters   []Theater `json:"theaters"`
	Shows      []show    `json:"shows"`
	Total      int       `json:"total"`
	NextCursor *string   `json:"nextCursor"`
}

// walk follows a list to its end, handing each page to read. A list with no
// cursor is one page, which is how /theaters comes back.
func walk(h *House, path string, read func(listPage)) error {
	for cursor := ""; ; {
		url := fmt.Sprintf("%s%s?limit=%d", h.theatreURL, path, pageSize)
		if cursor != "" {
			url += "&cursor=" + neturl.QueryEscape(cursor)
		}

		resp, err := h.http.Get(url)
		if err != nil {
			return fmt.Errorf("%s is unreachable: %w", path, err)
		}
		var page listPage
		err = json.NewDecoder(resp.Body).Decode(&page)
		status := resp.StatusCode
		resp.Body.Close()
		if status != http.StatusOK {
			return fmt.Errorf("%s returned %d", path, status)
		}
		if err != nil {
			return fmt.Errorf("%s is not readable: %w", path, err)
		}

		read(page)
		if page.NextCursor == nil {
			return nil
		}
		cursor = *page.NextCursor
	}
}

// Screening looks one screening up by the id the schedule published.
func (h *House) Screening(id string) (Screening, error) {
	screenings, err := h.Programme()
	if err != nil {
		return Screening{}, err
	}
	for _, s := range screenings {
		if s.ID == id {
			return s, nil
		}
	}
	return Screening{}, fmt.Errorf("there is no screening called %q in this week's programme", id)
}

// SeatMap reads the live house for a screening.
func (h *House) SeatMap(id string) (*api.SeatMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.seating.GetSeatMap(ctx, &api.GetSeatMapRequest{ShowId: id})
	if err != nil {
		return nil, fmt.Errorf("the seat map is unreachable: %w", err)
	}
	return resp.SeatMap, nil
}
