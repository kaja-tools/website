// Package theatre is a small client for the theatre catalog's REST API.
// The seating service trusts the catalog for which performances exist and
// what their base price is; it owns everything about seats itself.
package theatre

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Performance struct {
	ID             string    `json:"id"`
	EventID        string    `json:"eventId"`
	EventTitle     string    `json:"eventTitle"`
	StartsAt       time.Time `json:"startsAt"`
	BasePriceCents int       `json:"basePriceCents"`
	Status         string    `json:"status"`
}

type Client struct {
	baseURL string
	http    *http.Client

	mu    sync.Mutex
	cache map[string]Performance
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
		cache:   map[string]Performance{},
	}
}

// Performance validates a performance id against the catalog and returns
// its details. Successful lookups are cached until the show starts.
func (c *Client) Performance(id string) (Performance, error) {
	c.mu.Lock()
	cached, ok := c.cache[id]
	c.mu.Unlock()
	if ok {
		if time.Now().After(cached.StartsAt) {
			return Performance{}, status.Errorf(codes.FailedPrecondition, "performance %s already happened", id)
		}
		return cached, nil
	}

	resp, err := c.http.Get(fmt.Sprintf("%s/performances/%s", c.baseURL, id))
	if err != nil {
		return Performance{}, status.Errorf(codes.Unavailable, "theatre catalog unreachable: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return Performance{}, status.Errorf(codes.NotFound, "no performance %q in the theatre catalog", id)
	case http.StatusGone:
		return Performance{}, status.Errorf(codes.FailedPrecondition, "performance %s already happened", id)
	default:
		return Performance{}, status.Errorf(codes.Unavailable, "theatre catalog returned %d", resp.StatusCode)
	}

	var p Performance
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return Performance{}, status.Errorf(codes.Internal, "bad catalog response: %v", err)
	}
	if p.Status != "onSale" {
		return Performance{}, status.Errorf(codes.FailedPrecondition, "performance %s is not on sale", id)
	}

	c.mu.Lock()
	c.cache[id] = p
	c.mu.Unlock()
	return p, nil
}
