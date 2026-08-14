// Package theatre is a small client for the theatre catalog's REST API.
// The seating service trusts the catalog for which shows exist and what
// they cost; it owns everything about seats itself.
package theatre

import (
	"encoding/json"
	"net/http"
	neturl "net/url"
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

// page is one response from the catalog, which paginates. The client takes
// whatever page size the catalog defaults to and follows the cursor; total is
// only there to size the slice up front.
type page struct {
	Shows      []Show  `json:"shows"`
	Total      int     `json:"total"`
	NextCursor *string `json:"nextCursor"`
}

func (c *Client) page(cursor string) (page, error) {
	url := c.baseURL + "/shows"
	if cursor != "" {
		url += "?cursor=" + neturl.QueryEscape(cursor)
	}

	resp, err := c.http.Get(url)
	if err != nil {
		return page{}, status.Errorf(codes.Unavailable, "theatre catalog unreachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return page{}, status.Errorf(codes.Unavailable, "theatre catalog returned %d", resp.StatusCode)
	}

	var out page
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return page{}, status.Errorf(codes.Internal, "bad catalog response: %v", err)
	}
	return out, nil
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
