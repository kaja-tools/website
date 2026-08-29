// Package ratelimit caps how much of the seating service one caller may use.
//
// It is a fixed window per client, held in memory: the service has no storage,
// and a budget nobody remembers across a restart is the honest shape for one
// process serving a demo. A window that has run out is the whole state, so an
// idle caller is forgotten rather than kept.
//
// What it says about the budget is read off every response, refused or not,
// in both spellings that are in the wild: the RateLimit and RateLimit-Policy
// structured fields of draft-ietf-httpapi-ratelimit-headers, and the
// RateLimit-Limit / -Remaining / -Reset triple that predates them.
package ratelimit

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// The name the policy is reported under. A service with one budget still has to
// name it: the draft keys every field on the policy it belongs to.
const Policy = "default"

// Big enough that reading a programme, a house and a booking never comes near
// it; small enough that a script written to hit it does, in a few seconds.
const (
	Quota  = 120
	Window = 30 * time.Second
)

// How many idle clients are kept before the expired ones are swept. In-memory
// means the map is the only thing that can grow, and a client whose window has
// turned over is not a client at all.
const sweepAt = 1024

// Result is one client's budget at the moment a call arrived.
type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	ResetAfter time.Duration
	Window     time.Duration
}

type window struct {
	used    int
	resetAt time.Time
}

// Limiter counts calls per client over a fixed window.
type Limiter struct {
	quota  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	clients map[string]*window
}

func New(quota int, length time.Duration) *Limiter {
	return &Limiter{quota: quota, window: length, now: time.Now, clients: map[string]*window{}}
}

// Take spends one call of the client's budget and reports what is left. A
// refused call is not counted against the budget it was refused by, so a caller
// hammering a spent window still gets in as soon as it turns over.
func (l *Limiter) Take(client string) Result {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.clients) >= sweepAt {
		for key, w := range l.clients {
			if !now.Before(w.resetAt) {
				delete(l.clients, key)
			}
		}
	}

	w, ok := l.clients[client]
	if !ok || !now.Before(w.resetAt) {
		w = &window{resetAt: now.Add(l.window)}
		l.clients[client] = w
	}

	if w.used >= l.quota {
		return Result{Limit: l.quota, ResetAfter: w.resetAt.Sub(now), Window: l.window}
	}

	w.used++
	return Result{Allowed: true, Limit: l.quota, Remaining: l.quota - w.used, ResetAfter: w.resetAt.Sub(now), Window: l.window}
}

// Headers is what a response says about the budget, keyed by the lowercase
// names gRPC metadata uses. Retry-After is only on a refusal: RFC 9110 defines
// it as how long to wait before trying again, which is a thing to say to a
// caller that has just been turned away.
func (r Result) Headers() map[string]string {
	reset := int(math.Ceil(r.ResetAfter.Seconds()))
	if reset < 0 {
		reset = 0
	}

	headers := map[string]string{
		"ratelimit-policy":    fmt.Sprintf("%q;q=%d;w=%d", Policy, r.Limit, int(r.Window.Seconds())),
		"ratelimit":           fmt.Sprintf("%q;r=%d;t=%d", Policy, r.Remaining, reset),
		"ratelimit-limit":     fmt.Sprint(r.Limit),
		"ratelimit-remaining": fmt.Sprint(r.Remaining),
		"ratelimit-reset":     fmt.Sprint(reset),
	}
	if !r.Allowed {
		headers["retry-after"] = fmt.Sprint(reset)
	}
	return headers
}
