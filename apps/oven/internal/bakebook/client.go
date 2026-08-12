// Package bakebook is a small client for the bakebook catalog's REST API.
// The oven trusts the book for which cookies exist and how they bake; it owns
// everything about racks, dough and timers itself.
package bakebook

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// How long the book is cached. It never changes.
const cacheTTL = 5 * time.Minute

type Cookie struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	BakeMinutes         int    `json:"bakeMinutes"`
	OvenCelsius         int    `json:"ovenCelsius"`
	PerTray             int    `json:"perTray"`
	DoughGramsPerCookie int    `json:"doughGramsPerCookie"`
}

type Client struct {
	baseURL string
	http    *http.Client

	mu      sync.Mutex
	cookies []Cookie
	until   time.Time
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Cookies returns the whole book, cached for a few minutes.
func (c *Client) Cookies() ([]Cookie, error) {
	c.mu.Lock()
	if time.Now().Before(c.until) {
		out := c.cookies
		c.mu.Unlock()
		return out, nil
	}
	c.mu.Unlock()

	resp, err := c.http.Get(c.baseURL + "/cookies")
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "bakebook unreachable: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.Unavailable, "bakebook returned %d", resp.StatusCode)
	}

	var cookies []Cookie
	if err := json.NewDecoder(resp.Body).Decode(&cookies); err != nil {
		return nil, status.Errorf(codes.Internal, "bad bakebook response: %v", err)
	}

	c.mu.Lock()
	c.cookies = cookies
	c.until = time.Now().Add(cacheTTL)
	c.mu.Unlock()
	return cookies, nil
}

// Cookie validates a cookie id against the book and returns its recipe.
func (c *Client) Cookie(id string) (Cookie, error) {
	cookies, err := c.Cookies()
	if err != nil {
		return Cookie{}, err
	}
	for _, cookie := range cookies {
		if cookie.ID == id {
			return cookie, nil
		}
	}
	return Cookie{}, status.Errorf(codes.NotFound,
		"we don't bake %q — the whole book is at %s/cookies", id, c.baseURL)
}
