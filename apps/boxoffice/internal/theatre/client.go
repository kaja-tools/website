// Package theatre is the box office's client for the theatre catalog REST
// API — the same public API kaja users call, dogfooded service-to-service.
package theatre

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/twitchtv/twirp"
)

type Performance struct {
	ID             string    `json:"id"`
	EventID        string    `json:"eventId"`
	EventTitle     string    `json:"eventTitle"`
	StartsAt       time.Time `json:"startsAt"`
	BasePriceCents int       `json:"basePriceCents"`
	Status         string    `json:"status"`
}

type eventSummary struct {
	ID string `json:"id"`
}

type eventPage struct {
	Items []eventSummary `json:"items"`
}

type eventFull struct {
	Performances []Performance `json:"performances"`
}

type Client struct {
	baseURL string
	http    *http.Client

	mu            sync.Mutex
	perfCache     map[string]Performance
	upcoming      []Performance
	upcomingUntil time.Time
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:   baseURL,
		http:      &http.Client{Timeout: 5 * time.Second},
		perfCache: map[string]Performance{},
	}
}

// Performance validates a performance id and returns its details, cached
// until the show starts.
func (c *Client) Performance(id string) (Performance, error) {
	c.mu.Lock()
	cached, ok := c.perfCache[id]
	c.mu.Unlock()
	if ok {
		if time.Now().After(cached.StartsAt) {
			return Performance{}, twirp.NewError(twirp.FailedPrecondition,
				fmt.Sprintf("performance %s already happened", id))
		}
		return cached, nil
	}

	resp, err := c.http.Get(fmt.Sprintf("%s/performances/%s", c.baseURL, id))
	if err != nil {
		return Performance{}, twirp.NewError(twirp.Unavailable, "theatre catalog unreachable")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return Performance{}, twirp.NewError(twirp.NotFound,
			fmt.Sprintf("no performance %q in the theatre catalog", id))
	case http.StatusGone:
		return Performance{}, twirp.NewError(twirp.FailedPrecondition,
			fmt.Sprintf("performance %s already happened", id))
	default:
		return Performance{}, twirp.NewError(twirp.Unavailable,
			fmt.Sprintf("theatre catalog returned %d", resp.StatusCode))
	}

	var p Performance
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return Performance{}, twirp.NewError(twirp.Internal, "bad catalog response")
	}
	if p.Status != "onSale" {
		return Performance{}, twirp.NewError(twirp.FailedPrecondition,
			fmt.Sprintf("performance %s is not on sale", id))
	}

	c.mu.Lock()
	c.perfCache[id] = p
	c.mu.Unlock()
	return p, nil
}

// Upcoming lists every on-sale performance in the catalog's rolling window,
// cached for a few minutes. Used by the crowd to decide what to buy.
func (c *Client) Upcoming() ([]Performance, error) {
	c.mu.Lock()
	if time.Now().Before(c.upcomingUntil) {
		out := c.upcoming
		c.mu.Unlock()
		return out, nil
	}
	c.mu.Unlock()

	resp, err := c.http.Get(c.baseURL + "/events?perPage=50")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var page eventPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	var all []Performance
	for _, e := range page.Items {
		resp, err := c.http.Get(fmt.Sprintf("%s/events/%s", c.baseURL, e.ID))
		if err != nil {
			return nil, err
		}
		var full eventFull
		err = json.NewDecoder(resp.Body).Decode(&full)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, p := range full.Performances {
			if p.Status == "onSale" {
				all = append(all, p)
			}
		}
	}

	c.mu.Lock()
	c.upcoming = all
	c.upcomingUntil = time.Now().Add(5 * time.Minute)
	c.mu.Unlock()
	return all, nil
}
