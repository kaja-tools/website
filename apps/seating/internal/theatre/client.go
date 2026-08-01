// Package theatre is a small client for the theatre catalog's REST API.
// The seating service trusts the catalog for which shows exist and what
// they cost; it owns everything about seats itself.
package theatre

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// How long the programme is cached. It changes at most once a week.
const cacheTTL = 5 * time.Minute

type Show struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	StartsAt       time.Time `json:"startsAt"`
	BasePriceCents int       `json:"basePriceCents"`
}

type Client struct {
	baseURL string
	http    *http.Client

	mu    sync.Mutex
	shows []Show
	until time.Time
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Shows returns the catalog's whole programme, cached for a few minutes.
func (c *Client) Shows() ([]Show, error) {
	c.mu.Lock()
	if time.Now().Before(c.until) {
		out := c.shows
		c.mu.Unlock()
		return out, nil
	}
	c.mu.Unlock()

	resp, err := c.http.Get(c.baseURL + "/shows")
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "theatre catalog unreachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Unavailable, "theatre catalog returned %d", resp.StatusCode)
	}

	var shows []Show
	if err := json.NewDecoder(resp.Body).Decode(&shows); err != nil {
		return nil, status.Errorf(codes.Internal, "bad catalog response: %v", err)
	}

	c.mu.Lock()
	c.shows = shows
	c.until = time.Now().Add(cacheTTL)
	c.mu.Unlock()
	return shows, nil
}

// Show validates a show id against the catalog and returns its details.
func (c *Client) Show(id string) (Show, error) {
	shows, err := c.Shows()
	if err != nil {
		return Show{}, err
	}
	for _, s := range shows {
		if s.ID == id {
			return s, nil
		}
	}
	return Show{}, status.Errorf(codes.NotFound,
		"no show %q — list them all at %s/shows", id, c.baseURL)
}
