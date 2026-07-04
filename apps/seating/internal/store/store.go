// Package store owns live seat state for every performance. Everything is
// in memory: seat state is disposable by design (holds expire, performances
// roll off the schedule), so a restart simply resets the house.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/theatre"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	HoldTTL         = 2 * time.Minute
	MaxSeatsPerHold = 6
	// Slow watchers get this much channel buffer before being dropped.
	watchBuffer = 256
	// How many recent changes we keep per performance for pollers to catch
	// up on. A polling client that falls further behind than this gets a
	// reset and re-fetches the snapshot.
	changeLogSize = 512
)

// The house layout. This mirrors GET /venue in the theatre catalog; the two
// services share the contract, not the code.
type layoutSection struct {
	section  api.Section
	rows     string // row letters
	seats    int    // per row
	priceNum int    // price = base * priceNum / priceDen
	priceDen int
}

var layout = []layoutSection{
	{api.Section_SECTION_ORCHESTRA, "ABCDEFGH", 14, 1, 1},
	{api.Section_SECTION_BALCONY, "JKLM", 12, 3, 4},
}

type seat struct {
	id      string
	section api.Section
	row     string
	number  int32
	status  api.SeatStatus
	price   int32
	holdID  string
}

type hold struct {
	id      string
	perfID  string
	seatIDs []string
	total   int32
	expires time.Time
}

type performance struct {
	id       string
	seats    map[string]*seat
	watchers map[chan *api.SeatUpdate]struct{}
	// seq is a monotonic counter; log holds the last changeLogSize changes
	// in ascending seq order so unary pollers can catch up without a stream.
	seq int64
	log []*api.SeatChange
}

type Store struct {
	theatre *theatre.Client

	mu    sync.Mutex
	perfs map[string]*performance
	holds map[string]*hold
}

func New(theatreClient *theatre.Client) *Store {
	return &Store{
		theatre: theatreClient,
		perfs:   map[string]*performance{},
		holds:   map[string]*hold{},
	}
}

// perf returns the live state for a performance, lazily building the seat
// map the first time a valid performance is touched. The catalog lookup
// happens outside the lock.
func (s *Store) perf(perfID string) (*performance, error) {
	s.mu.Lock()
	p, ok := s.perfs[perfID]
	s.mu.Unlock()
	if ok {
		return p, nil
	}

	info, err := s.theatre.Performance(perfID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.perfs[perfID]; ok {
		return p, nil
	}
	p = &performance{
		id:       perfID,
		seats:    map[string]*seat{},
		watchers: map[chan *api.SeatUpdate]struct{}{},
	}
	for _, sec := range layout {
		price := int32(info.BasePriceCents * sec.priceNum / sec.priceDen)
		for _, letter := range strings.Split(sec.rows, "") {
			for n := 1; n <= sec.seats; n++ {
				id := fmt.Sprintf("%s%d", letter, n)
				p.seats[id] = &seat{
					id:      id,
					section: sec.section,
					row:     letter,
					number:  int32(n),
					status:  api.SeatStatus_SEAT_STATUS_AVAILABLE,
					price:   price,
				}
			}
		}
	}
	s.perfs[perfID] = p
	return p, nil
}

func (p *performance) snapshotLocked() *api.SeatMap {
	m := &api.SeatMap{
		PerformanceId:      p.id,
		AvailableBySection: map[string]int32{},
	}
	for _, sec := range layout {
		sectionMap := &api.SectionMap{Section: sec.section}
		sectionName := strings.TrimPrefix(sec.section.String(), "SECTION_")
		m.AvailableBySection[sectionName] = 0
		for _, letter := range strings.Split(sec.rows, "") {
			row := &api.Row{Letter: letter}
			for n := 1; n <= sec.seats; n++ {
				st := p.seats[fmt.Sprintf("%s%d", letter, n)]
				row.Seats = append(row.Seats, &api.Seat{
					Id:         st.id,
					Section:    st.section,
					Row:        st.row,
					Number:     st.number,
					Status:     st.status,
					PriceCents: st.price,
				})
				switch st.status {
				case api.SeatStatus_SEAT_STATUS_AVAILABLE:
					m.Available++
					m.AvailableBySection[sectionName]++
				case api.SeatStatus_SEAT_STATUS_HELD:
					m.Held++
				case api.SeatStatus_SEAT_STATUS_SOLD:
					m.Sold++
				}
			}
			sectionMap.Rows = append(sectionMap.Rows, row)
		}
		m.Sections = append(m.Sections, sectionMap)
	}
	m.Cursor = p.seq
	return m
}

// emitLocked records a seat change in the poll log and fans it out to every
// watcher. Watchers that cannot keep up are dropped so a stuck stream never
// blocks the house.
func (p *performance) emitLocked(seatID string, st api.SeatStatus, reason api.ChangeReason) {
	p.seq++
	change := &api.SeatChange{
		PerformanceId: p.id,
		SeatId:        seatID,
		Status:        st,
		Reason:        reason,
		ChangedAt:     timestamppb.Now(),
		Seq:           p.seq,
	}

	p.log = append(p.log, change)
	if len(p.log) > changeLogSize {
		p.log = p.log[len(p.log)-changeLogSize:]
	}

	update := &api.SeatUpdate{Update: &api.SeatUpdate_Change{Change: change}}
	for ch := range p.watchers {
		select {
		case ch <- update:
		default:
			delete(p.watchers, ch)
			close(ch)
		}
	}
}

// SeatMap returns a snapshot of a performance's seats.
func (s *Store) SeatMap(perfID string) (*api.SeatMap, error) {
	p, err := s.perf(perfID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return p.snapshotLocked(), nil
}

// Changes returns the seat changes after cursor, the current cursor to poll
// from next, and whether the caller fell behind the retained log (in which
// case it should re-fetch a snapshot). A cursor of 0 starts from the latest
// change, returning none. This is the unary, web-friendly counterpart to
// WatchSeats.
func (s *Store) Changes(perfID string, cursor int64) (changes []*api.SeatChange, next int64, reset bool, err error) {
	p, err := s.perf(perfID)
	if err != nil {
		return nil, 0, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if cursor == 0 || len(p.log) == 0 {
		return nil, p.seq, false, nil
	}
	// log seqs are contiguous and ascending; if the requested cursor sits
	// before the oldest retained change, some were evicted — signal a reset.
	if cursor < p.log[0].Seq-1 {
		return nil, p.seq, true, nil
	}
	for _, c := range p.log {
		if c.Seq > cursor {
			changes = append(changes, c)
		}
	}
	return changes, p.seq, false, nil
}

// Subscribe returns a consistent snapshot plus a channel of every change
// after it. Call the returned cancel function when done.
func (s *Store) Subscribe(perfID string) (*api.SeatMap, <-chan *api.SeatUpdate, func(), error) {
	p, err := s.perf(perfID)
	if err != nil {
		return nil, nil, nil, err
	}
	ch := make(chan *api.SeatUpdate, watchBuffer)

	s.mu.Lock()
	snapshot := p.snapshotLocked()
	p.watchers[ch] = struct{}{}
	s.mu.Unlock()

	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := p.watchers[ch]; ok {
			delete(p.watchers, ch)
			close(ch)
		}
	}
	return snapshot, ch, cancel, nil
}

// Hold places a hold on the given seats. All seats must be available;
// otherwise the caller is told exactly which ones were lost.
func (s *Store) Hold(perfID string, seatIDs []string) (*api.Hold, error) {
	if len(seatIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "pick at least one seat")
	}
	if len(seatIDs) > MaxSeatsPerHold {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d seats per hold", MaxSeatsPerHold)
	}

	p, err := s.perf(perfID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(seatIDs))
	seen := map[string]bool{}
	for _, raw := range seatIDs {
		id := strings.ToUpper(strings.TrimSpace(raw))
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, ok := p.seats[id]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "there is no seat %q in the house", raw)
		}
		ids = append(ids, id)
	}

	var taken []string
	var total int32
	for _, id := range ids {
		if p.seats[id].status != api.SeatStatus_SEAT_STATUS_AVAILABLE {
			taken = append(taken, id)
		}
		total += p.seats[id].price
	}
	if len(taken) > 0 {
		verb := "was"
		if len(taken) > 1 {
			verb = "were"
		}
		return nil, status.Errorf(codes.AlreadyExists, "seat %s %s just taken", strings.Join(taken, ", "), verb)
	}

	h := &hold{
		id:      "hold_" + randomID(),
		perfID:  perfID,
		seatIDs: ids,
		total:   total,
		expires: time.Now().Add(HoldTTL),
	}
	s.holds[h.id] = h
	for _, id := range ids {
		p.seats[id].status = api.SeatStatus_SEAT_STATUS_HELD
		p.seats[id].holdID = h.id
		p.emitLocked(id, api.SeatStatus_SEAT_STATUS_HELD, api.ChangeReason_CHANGE_REASON_HOLD)
	}
	time.AfterFunc(HoldTTL, func() { s.expire(h.id) })

	return holdProto(h), nil
}

// Confirm turns a hold into sold seats.
func (s *Store) Confirm(holdID string) ([]*api.Seat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	h, ok := s.holds[holdID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "hold %q not found — it may have expired", holdID)
	}
	p := s.perfs[h.perfID]
	delete(s.holds, holdID)

	var seats []*api.Seat
	for _, id := range h.seatIDs {
		st := p.seats[id]
		st.status = api.SeatStatus_SEAT_STATUS_SOLD
		st.holdID = ""
		p.emitLocked(id, api.SeatStatus_SEAT_STATUS_SOLD, api.ChangeReason_CHANGE_REASON_PURCHASE)
		seats = append(seats, &api.Seat{
			Id:         st.id,
			Section:    st.section,
			Row:        st.row,
			Number:     st.number,
			Status:     st.status,
			PriceCents: st.price,
		})
	}
	return seats, nil
}

// Release frees a hold's seats early.
func (s *Store) Release(holdID string) error {
	return s.release(holdID, api.ChangeReason_CHANGE_REASON_RELEASE)
}

func (s *Store) expire(holdID string) {
	s.release(holdID, api.ChangeReason_CHANGE_REASON_EXPIRE) //nolint:errcheck // already-gone holds are fine
}

func (s *Store) release(holdID string, reason api.ChangeReason) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	h, ok := s.holds[holdID]
	if !ok {
		if reason == api.ChangeReason_CHANGE_REASON_EXPIRE {
			return nil // purchased or released before the TTL fired
		}
		return status.Errorf(codes.NotFound, "hold %q not found — it may have expired", holdID)
	}
	p := s.perfs[h.perfID]
	delete(s.holds, holdID)

	for _, id := range h.seatIDs {
		p.seats[id].status = api.SeatStatus_SEAT_STATUS_AVAILABLE
		p.seats[id].holdID = ""
		p.emitLocked(id, api.SeatStatus_SEAT_STATUS_AVAILABLE, reason)
	}
	return nil
}

func holdProto(h *hold) *api.Hold {
	return &api.Hold{
		Id:              h.id,
		PerformanceId:   h.perfID,
		SeatIds:         h.seatIDs,
		TotalPriceCents: h.total,
		ExpiresAt:       timestamppb.New(h.expires),
	}
}

func randomID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}
