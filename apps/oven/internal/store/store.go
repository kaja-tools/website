// Package store owns the oven. Everything is in memory: a rack is hot for
// twenty seconds and a counter is cleared every night, so a restart simply
// means the bakery opened again.
//
// The oven runs on demo time. A recipe's minutes become seconds here, which is
// the only liberty this service takes — it is the difference between watching a
// bake finish and reading about one.
package store

import (
	crand "crypto/rand"
	"encoding/hex"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/bakebook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Four racks is the whole constraint the demo is built on: small enough
	// that two bakers genuinely collide, big enough to watch.
	Racks    = 4
	MaxTrays = 3
	// Even a good oven has a bad tray.
	scorchChance = 0.07
	// What the oven falls back to with nothing in it.
	idleCelsius = 170
	// Slow watchers get this much channel buffer before being dropped.
	watchBuffer = 256
)

// Pace is how fast the bakery runs. It is a parameter rather than a constant
// because the whole service turns on it: a bake you can watch finish is the
// difference between a demo and a description of one, and a test has no
// patience at all.
type Pace struct {
	// What one minute in the recipe is worth. A real oven would say a minute.
	BakeUnit time.Duration
	// How long a batch sits on the wire racks before it can be sold.
	CoolFor time.Duration
	// How long a claimed rack stays yours without a bake starting on it.
	ClaimTTL time.Duration
}

func DefaultPace() Pace {
	return Pace{
		BakeUnit: time.Second,
		CoolFor:  6 * time.Second,
		ClaimTTL: 45 * time.Second,
	}
}

type rack struct {
	number     int32
	state      api.RackState
	cookieID   string
	cookieName string
	trays      int32
	cookies    int32
	baker      string
	until      time.Time
	claimID    string
}

type claim struct {
	id         string
	rackNumber int32
	cookie     bakebook.Cookie
	trays      int32
	cookies    int32
	doughGrams int32
	expires    time.Time
	baker      string
}

type Store struct {
	book *bakebook.Client
	pace Pace

	mu            sync.Mutex
	racks         []*rack
	claims        map[string]*claim
	celsius       int32
	cooling       int32
	onCounter     int32
	bakedToday    int32
	scorchedToday int32
	watchers      map[chan *api.OvenUpdate]struct{}
}

func New(book *bakebook.Client, pace Pace) *Store {
	s := &Store{
		book:     book,
		pace:     pace,
		claims:   map[string]*claim{},
		celsius:  idleCelsius,
		watchers: map[chan *api.OvenUpdate]struct{}{},
	}
	for n := int32(1); n <= Racks; n++ {
		s.racks = append(s.racks, &rack{number: n, state: api.RackState_RACK_STATE_EMPTY})
	}
	return s
}

func (s *Store) snapshotLocked() *api.OvenState {
	o := &api.OvenState{
		Celsius:       s.celsius,
		Cooling:       s.cooling,
		OnCounter:     s.onCounter,
		BakedToday:    s.bakedToday,
		ScorchedToday: s.scorchedToday,
	}
	for _, r := range s.racks {
		o.Racks = append(o.Racks, rackProto(r))
		if r.state == api.RackState_RACK_STATE_EMPTY {
			o.FreeRacks++
		}
	}
	return o
}

// emitLocked fans a rack change out to every watcher. Watchers that cannot
// keep up are dropped so a stuck stream never blocks the oven.
func (s *Store) emitLocked(r *rack, reason api.ChangeReason, cookies int32) {
	update := &api.OvenUpdate{Update: &api.OvenUpdate_Change{Change: &api.RackChange{
		RackNumber: r.number,
		State:      r.state,
		Reason:     reason,
		CookieId:   r.cookieID,
		Baker:      r.baker,
		Cookies:    cookies,
		ChangedAt:  timestamppb.Now(),
	}}}
	for ch := range s.watchers {
		select {
		case ch <- update:
		default:
			delete(s.watchers, ch)
			close(ch)
		}
	}
}

// Oven returns a snapshot of every rack and both counters.
func (s *Store) Oven() *api.OvenState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

// Subscribe returns a consistent snapshot plus a channel of every change after
// it. Call the returned cancel function when done.
func (s *Store) Subscribe() (*api.OvenState, <-chan *api.OvenUpdate, func()) {
	ch := make(chan *api.OvenUpdate, watchBuffer)

	s.mu.Lock()
	snapshot := s.snapshotLocked()
	s.watchers[ch] = struct{}{}
	s.mu.Unlock()

	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := s.watchers[ch]; ok {
			delete(s.watchers, ch)
			close(ch)
		}
	}
	return snapshot, ch, cancel
}

// Claim takes a free rack for a cookie. The dough is not spent yet — that
// happens when the bake starts — so an abandoned claim costs nothing but the
// rack, and only for as long as the TTL.
func (s *Store) Claim(cookieID string, trays int32, baker string) (*api.Claim, error) {
	if trays == 0 {
		trays = 1
	}
	if trays < 1 || trays > MaxTrays {
		return nil, status.Errorf(codes.InvalidArgument, "a rack holds 1 to %d trays, not %d", MaxTrays, trays)
	}
	cookie, err := s.book.Cookie(cookieID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var free *rack
	for _, r := range s.racks {
		if r.state == api.RackState_RACK_STATE_EMPTY {
			free = r
			break
		}
	}
	if free == nil {
		return nil, status.Errorf(codes.ResourceExhausted,
			"all %d racks are busy — somebody got there first. Call GetOven to see when one frees up", Racks)
	}

	cookies := trays * int32(cookie.PerTray)
	c := &claim{
		id:         "claim_" + randomID(),
		rackNumber: free.number,
		cookie:     cookie,
		trays:      trays,
		cookies:    cookies,
		doughGrams: cookies * int32(cookie.DoughGramsPerCookie),
		expires:    time.Now().Add(s.pace.ClaimTTL),
		baker:      baker,
	}
	s.claims[c.id] = c

	free.state = api.RackState_RACK_STATE_CLAIMED
	free.cookieID = cookie.ID
	free.cookieName = cookie.Name
	free.trays = trays
	free.cookies = cookies
	free.baker = baker
	free.until = c.expires
	free.claimID = c.id
	s.emitLocked(free, api.ChangeReason_CHANGE_REASON_CLAIMED, cookies)

	time.AfterFunc(s.pace.ClaimTTL, func() { s.expire(c.id) })

	return claimProto(c), nil
}

// Start turns a claim into a batch in the oven. This is the irreversible one:
// the dough is spent here, and the only two ways out are cookies and a bin.
func (s *Store) Start(claimID string) (*api.Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.claims[claimID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "claim %q not found — it may have expired", claimID)
	}
	delete(s.claims, claimID)

	r := s.racks[c.rackNumber-1]
	bake := time.Duration(c.cookie.BakeMinutes) * s.pace.BakeUnit // demo time
	readyAt := time.Now().Add(bake)

	s.celsius = int32(c.cookie.OvenCelsius)
	r.state = api.RackState_RACK_STATE_BAKING
	r.until = readyAt
	r.claimID = ""
	s.emitLocked(r, api.ChangeReason_CHANGE_REASON_STARTED, c.cookies)

	time.AfterFunc(bake, func() { s.finish(r.number, c.cookies) })

	return &api.Batch{
		Id:          "batch_" + randomID(),
		RackNumber:  r.number,
		CookieId:    c.cookie.ID,
		Cookies:     c.cookies,
		Celsius:     int32(c.cookie.OvenCelsius),
		BakeSeconds: int32(c.cookie.BakeMinutes),
		ReadyAt:     timestamppb.New(readyAt),
	}, nil
}

// finish takes a batch out of the oven: onto the wire racks, or into the bin.
func (s *Store) finish(rackNumber, cookies int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := s.racks[rackNumber-1]
	if r.state != api.RackState_RACK_STATE_BAKING {
		return
	}

	if rand.Float64() < scorchChance {
		s.scorchedToday += cookies
		s.emitLocked(r, api.ChangeReason_CHANGE_REASON_SCORCHED, cookies)
		s.clearLocked(r)
		return
	}

	s.bakedToday += cookies
	s.cooling += cookies
	r.state = api.RackState_RACK_STATE_COOLING
	r.until = time.Now().Add(s.pace.CoolFor)
	s.emitLocked(r, api.ChangeReason_CHANGE_REASON_OUT, cookies)

	time.AfterFunc(s.pace.CoolFor, func() { s.cool(rackNumber, cookies) })
}

// cool moves a cooled batch off the rack and onto the counter.
func (s *Store) cool(rackNumber, cookies int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := s.racks[rackNumber-1]
	if r.state != api.RackState_RACK_STATE_COOLING {
		return
	}
	s.cooling -= cookies
	s.onCounter += cookies
	s.emitLocked(r, api.ChangeReason_CHANGE_REASON_COOLED, cookies)
	s.clearLocked(r)
}

// Release frees a claimed rack early. Not exposed over gRPC — claims expire on
// their own — but the other bakers use it to change their minds.
func (s *Store) Release(claimID string) error {
	return s.release(claimID, api.ChangeReason_CHANGE_REASON_RELEASED)
}

func (s *Store) expire(claimID string) {
	s.release(claimID, api.ChangeReason_CHANGE_REASON_EXPIRED) //nolint:errcheck // already-gone claims are fine
}

func (s *Store) release(claimID string, reason api.ChangeReason) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.claims[claimID]
	if !ok {
		if reason == api.ChangeReason_CHANGE_REASON_EXPIRED {
			return nil // baked or released before the TTL fired
		}
		return status.Errorf(codes.NotFound, "claim %q not found — it may have expired", claimID)
	}
	delete(s.claims, claimID)

	r := s.racks[c.rackNumber-1]
	if r.claimID != claimID {
		return nil
	}
	s.emitLocked(r, reason, c.cookies)
	s.clearLocked(r)
	return nil
}

// Sell takes cookies off the counter. The bakery's customers do this all day;
// it is what keeps the counter from only ever growing.
func (s *Store) Sell(n int32) int32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n > s.onCounter {
		n = s.onCounter
	}
	s.onCounter -= n
	return n
}

func (s *Store) clearLocked(r *rack) {
	r.state = api.RackState_RACK_STATE_EMPTY
	r.cookieID = ""
	r.cookieName = ""
	r.trays = 0
	r.cookies = 0
	r.baker = ""
	r.until = time.Time{}
	r.claimID = ""

	for _, other := range s.racks {
		if other.state == api.RackState_RACK_STATE_BAKING {
			return
		}
	}
	s.celsius = idleCelsius
}

func rackProto(r *rack) *api.Rack {
	p := &api.Rack{
		Number:     r.number,
		State:      r.state,
		CookieId:   r.cookieID,
		CookieName: r.cookieName,
		Trays:      r.trays,
		Cookies:    r.cookies,
		Baker:      r.baker,
	}
	if !r.until.IsZero() {
		p.Until = timestamppb.New(r.until)
	}
	return p
}

func claimProto(c *claim) *api.Claim {
	return &api.Claim{
		Id:         c.id,
		RackNumber: c.rackNumber,
		CookieId:   c.cookie.ID,
		Trays:      c.trays,
		Cookies:    c.cookies,
		DoughGrams: c.doughGrams,
		ExpiresAt:  timestamppb.New(c.expires),
	}
}

func randomID() string {
	b := make([]byte, 6)
	crand.Read(b)
	return hex.EncodeToString(b)
}
