// Package theatre is a small client for the Theatre service's REST API.
// The seating service trusts it for which screenings exist, when they start
// and what they cost; it owns everything about seats itself.
//
// Only the schedule is read. What the film is called and where the house is
// are the other two lists' to say, and a seat map needs neither.
package theatre

import (
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// How long the schedule is cached. It changes at most once a week.
const cacheTTL = 5 * time.Minute

// The biggest page the Theatre service will hand over, so that walking a
// schedule of a few thousand screenings is a handful of requests.
const pageSize = 500

type Show struct {
	ID         string    `json:"id"`
	MovieID    string    `json:"movieId"`
	TheaterID  string    `json:"theaterId"`
	StartsAt   time.Time `json:"startsAt"`
	PriceCents int       `json:"priceCents"`
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

// Shows returns the whole schedule, cached for a few minutes.
func (c *Client) Shows() ([]Show, error) {
	c.mu.Lock()
	if time.Now().Before(c.until) {
		out := c.shows
		c.mu.Unlock()
		return out, nil
	}
	c.mu.Unlock()

	var shows []Show
	for cursor := ""; ; {
		page, err := c.page(cursor)
		if err != nil {
			return nil, err
		}
		if shows == nil {
			shows = make([]Show, 0, page.Total)
		}
		shows = append(shows, page.Shows...)
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	c.mu.Lock()
	c.shows = shows
	c.until = time.Now().Add(cacheTTL)
	c.mu.Unlock()
	return shows, nil
}

// page is one response from the schedule, which paginates. The client asks
// for the biggest page it can and follows the cursor; total is only there to
// size the slice up front.
type page struct {
	Shows      []Show  `json:"shows"`
	Total      int     `json:"total"`
	NextCursor *string `json:"nextCursor"`
}

func (c *Client) page(cursor string) (page, error) {
	url := fmt.Sprintf("%s/shows?limit=%d", c.baseURL, pageSize)
	if cursor != "" {
		url += "&cursor=" + neturl.QueryEscape(cursor)
	}

	resp, err := c.http.Get(url)
	if err != nil {
		return page{}, status.Errorf(codes.Unavailable, "the Theatre schedule is unreachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return page{}, status.Errorf(codes.Unavailable, "the Theatre schedule returned %d", resp.StatusCode)
	}

	var out page
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return page{}, status.Errorf(codes.Internal, "the Theatre schedule is not readable: %v", err)
	}
	return out, nil
}

// Show validates a screening id against the schedule and returns its details.
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
		"no screening %q — list them all at %s/shows", id, c.baseURL)
}
