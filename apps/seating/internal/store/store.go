// Package store owns live seat state for every show. Everything is in
// memory: seat state is disposable by design (holds expire, houses reopen
// empty each week), so a restart simply resets the theatre.
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
)

// The house layout. The catalog does not publish it — a seat map describes
// itself, so there is nothing to look up first.
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
	showID  string
	seatIDs []string
	total   int32
	expires time.Time
}

type show struct {
	id    string
	seats map[string]*seat
	// startsAt is the curtain-up this house was built for. When the catalog
	// moves it on to next week, the house reopens empty.
	startsAt time.Time
	watchers map[chan *api.SeatUpdate]struct{}
}

type Store struct {
	theatre *theatre.Client

	mu    sync.Mutex
	shows map[string]*show
	holds map[string]*hold
}

func New(theatreClient *theatre.Client) *Store {
	return &Store{
		theatre: theatreClient,
		shows:   map[string]*show{},
		holds:   map[string]*hold{},
	}
}

func buildSeats(basePriceCents int) map[string]*seat {
	seats := map[string]*seat{}
	for _, sec := range layout {
		price := int32(basePriceCents * sec.priceNum / sec.priceDen)
		for _, letter := range strings.Split(sec.rows, "") {
			for n := 1; n <= sec.seats; n++ {
				id := fmt.Sprintf("%s%d", letter, n)
				seats[id] = &seat{
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
	return seats
}

// state returns the live house for a show, building it the first time the
// show is touched and reopening it empty once last week's performance is
// over. The catalog lookup is cached and happens outside the lock.
func (s *Store) state(showID string) (*show, error) {
	info, err := s.theatre.Show(showID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sh, ok := s.shows[showID]
	if !ok {
		sh = &show{id: showID, watchers: map[chan *api.SeatUpdate]struct{}{}}
		s.shows[showID] = sh
	}
	if !sh.startsAt.Equal(info.StartsAt) {
		sh.startsAt = info.StartsAt
		sh.seats = buildSeats(info.BasePriceCents)
	}
	return sh, nil
}

func (sh *show) snapshotLocked() *api.SeatMap {
	m := &api.SeatMap{
		ShowId:             sh.id,
		AvailableBySection: map[string]int32{},
	}
	for _, sec := range layout {
		sectionMap := &api.SectionMap{Section: sec.section}
		sectionName := strings.TrimPrefix(sec.section.String(), "SECTION_")
		m.AvailableBySection[sectionName] = 0
		for _, letter := range strings.Split(sec.rows, "") {
			row := &api.Row{Letter: letter}
			for n := 1; n <= sec.seats; n++ {
				st := sh.seats[fmt.Sprintf("%s%d", letter, n)]
				row.Seats = append(row.Seats, seatProto(st))
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
	return m
}

// emitLocked fans a seat change out to every watcher. Watchers that cannot
// keep up are dropped so a stuck stream never blocks the house.
func (sh *show) emitLocked(seatID string, st api.SeatStatus, reason api.ChangeReason) {
	update := &api.SeatUpdate{Update: &api.SeatUpdate_Change{Change: &api.SeatChange{
		ShowId:    sh.id,
		SeatId:    seatID,
		Status:    st,
		Reason:    reason,
		ChangedAt: timestamppb.Now(),
	}}}
	for ch := range sh.watchers {
		select {
		case ch <- update:
		default:
			delete(sh.watchers, ch)
			close(ch)
		}
	}
}

// SeatMap returns a snapshot of a show's seats.
func (s *Store) SeatMap(showID string) (*api.SeatMap, error) {
	sh, err := s.state(showID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return sh.snapshotLocked(), nil
}

// Subscribe returns a consistent snapshot plus a channel of every change
// after it. Call the returned cancel function when done.
func (s *Store) Subscribe(showID string) (*api.SeatMap, <-chan *api.SeatUpdate, func(), error) {
	sh, err := s.state(showID)
	if err != nil {
		return nil, nil, nil, err
	}
	ch := make(chan *api.SeatUpdate, watchBuffer)

	s.mu.Lock()
	snapshot := sh.snapshotLocked()
	sh.watchers[ch] = struct{}{}
	s.mu.Unlock()

	cancel := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := sh.watchers[ch]; ok {
			delete(sh.watchers, ch)
			close(ch)
		}
	}
	return snapshot, ch, cancel, nil
}

// Hold places a hold on the given seats. All seats must be available;
// otherwise the caller is told exactly which ones were lost.
func (s *Store) Hold(showID string, seatIDs []string) (*api.Hold, error) {
	if len(seatIDs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "pick at least one seat")
	}
	if len(seatIDs) > MaxSeatsPerHold {
		return nil, status.Errorf(codes.InvalidArgument, "at most %d seats per hold", MaxSeatsPerHold)
	}

	sh, err := s.state(showID)
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
		if _, ok := sh.seats[id]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "there is no seat %q in the house", raw)
		}
		ids = append(ids, id)
	}

	var taken []string
	var total int32
	for _, id := range ids {
		if sh.seats[id].status != api.SeatStatus_SEAT_STATUS_AVAILABLE {
			taken = append(taken, id)
		}
		total += sh.seats[id].price
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
		showID:  showID,
		seatIDs: ids,
		total:   total,
		expires: time.Now().Add(HoldTTL),
	}
	s.holds[h.id] = h
	for _, id := range ids {
		sh.seats[id].status = api.SeatStatus_SEAT_STATUS_HELD
		sh.seats[id].holdID = h.id
		sh.emitLocked(id, api.SeatStatus_SEAT_STATUS_HELD, api.ChangeReason_CHANGE_REASON_HOLD)
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
	sh := s.shows[h.showID]
	delete(s.holds, holdID)

	var seats []*api.Seat
	for _, id := range h.seatIDs {
		st := sh.seats[id]
		st.status = api.SeatStatus_SEAT_STATUS_SOLD
		st.holdID = ""
		sh.emitLocked(id, api.SeatStatus_SEAT_STATUS_SOLD, api.ChangeReason_CHANGE_REASON_PURCHASE)
		seats = append(seats, seatProto(st))
	}
	return seats, nil
}

// Release frees a hold's seats early. Not exposed over gRPC — holds expire
// on their own — but the crowd uses it to put seats back.
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
	sh := s.shows[h.showID]
	delete(s.holds, holdID)

	for _, id := range h.seatIDs {
		sh.seats[id].status = api.SeatStatus_SEAT_STATUS_AVAILABLE
		sh.seats[id].holdID = ""
		sh.emitLocked(id, api.SeatStatus_SEAT_STATUS_AVAILABLE, reason)
	}
	return nil
}

func seatProto(st *seat) *api.Seat {
	return &api.Seat{
		Id:         st.id,
		Section:    st.section,
		Row:        st.row,
		Number:     st.number,
		Status:     st.status,
		PriceCents: st.price,
	}
}

func holdProto(h *hold) *api.Hold {
	return &api.Hold{
		Id:              h.id,
		ShowId:          h.showID,
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
