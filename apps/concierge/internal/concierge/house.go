// Package concierge is the front of house: it reads the programme from the
// theatre service and the live seat map from the seating service, and turns a
// sentence about what you feel like into a screening and a row.
//
// It owns no data of its own. Everything it says is something one of the other
// two services told it a moment ago, which is the whole point — a concierge
// knows the building, not the tickets.
package concierge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// How long the programme is cached. It changes at most once a week.
const programmeTTL = 5 * time.Minute

// Show is one weekly screening, as the theatre service publishes it.
type Show struct {
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

// House is the concierge's two neighbours: the programme over HTTP and the
// seat map over gRPC.
type House struct {
	theatreURL string
	http       *http.Client
	seating    api.SeatingClient

	mu    sync.Mutex
	shows []Show
	until time.Time
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

// Programme returns the whole week, cached for a few minutes.
func (h *House) Programme() ([]Show, error) {
	h.mu.Lock()
	if time.Now().Before(h.until) {
		shows := h.shows
		h.mu.Unlock()
		return shows, nil
	}
	h.mu.Unlock()

	var shows []Show
	for cursor := ""; ; {
		page, err := h.page(cursor)
		if err != nil {
			return nil, err
		}
		shows = append(shows, page.Shows...)
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	h.mu.Lock()
	h.shows, h.until = shows, time.Now().Add(programmeTTL)
	h.mu.Unlock()
	return shows, nil
}

// page is one response from the theatre service, which publishes the
// programme a page at a time. Ask for the largest page it will give: the
// concierge wants the whole week before it can have an opinion about it.
type page struct {
	Shows      []Show  `json:"shows"`
	NextCursor *string `json:"nextCursor"`
}

const pageSize = 500

func (h *House) page(cursor string) (page, error) {
	query := url.Values{"limit": {strconv.Itoa(pageSize)}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}

	resp, err := h.http.Get(h.theatreURL + "/shows?" + query.Encode())
	if err != nil {
		return page{}, fmt.Errorf("the programme is unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return page{}, fmt.Errorf("the programme returned %d", resp.StatusCode)
	}

	var out page
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return page{}, fmt.Errorf("the programme is not readable: %w", err)
	}
	return out, nil
}

// Show looks one screening up by the id the programme published.
func (h *House) Show(id string) (Show, error) {
	shows, err := h.Programme()
	if err != nil {
		return Show{}, err
	}
	for _, s := range shows {
		if s.ID == id {
			return s, nil
		}
	}
	return Show{}, fmt.Errorf("there is no screening called %q in this week's programme", id)
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
