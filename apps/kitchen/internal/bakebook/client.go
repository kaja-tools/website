// Package bakebook is a small client for the bakebook catalog's REST API. The
// kitchen trusts the book for which cookies exist and how big a batch is; the
// recipes themselves are its own.
package bakebook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// How long the book is cached. It never changes.
const cacheTTL = 5 * time.Minute

// ErrUnknownCookie is a question about something we don't bake — an answer, not
// a failure, so a tool reports it as a result rather than as a broken call.
var ErrUnknownCookie = errors.New("unknown cookie")

type Cookie struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Texture             string   `json:"texture"`
	BakeMinutes         int      `json:"bakeMinutes"`
	PerTray             int      `json:"perTray"`
	DoughGramsPerCookie int      `json:"doughGramsPerCookie"`
	Allergens           []string `json:"allergens"`
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

func (c *Client) BaseURL() string { return c.baseURL }

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
		return nil, fmt.Errorf("bakebook unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bakebook returned %d", resp.StatusCode)
	}

	var cookies []Cookie
	if err := json.NewDecoder(resp.Body).Decode(&cookies); err != nil {
		return nil, fmt.Errorf("bad bakebook response: %w", err)
	}

	c.mu.Lock()
	c.cookies = cookies
	c.until = time.Now().Add(cacheTTL)
	c.mu.Unlock()
	return cookies, nil
}

// Cookie looks one recipe up, returning ErrUnknownCookie for an id the book
// does not have.
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
	return Cookie{}, fmt.Errorf("%w: %s", ErrUnknownCookie, id)
}

// IDs is every id in the book, for an answer that can say what there is.
func (c *Client) IDs() []string {
	cookies, err := c.Cookies()
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		out = append(out, cookie.ID)
	}
	return out
}
