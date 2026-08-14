package store

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaja-tools/website/v2/internal/theatre"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newStore builds a store over a one-screening programme, so a test never
// depends on the real catalog. Curtain-up is a full week out, which is the
// moment a house goes on sale: nothing has been sold in advance yet, so the
// test has every seat to itself.
func newStore(t *testing.T) *Store {
	t.Helper()
	startsAt := time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)
	programme := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"a-film","title":"A Film","startsAt":"` + startsAt + `","basePriceCents":1000}]`))
	}))
	t.Cleanup(programme.Close)
	return New(theatre.NewClient(programme.URL))
}

func TestBookSellsTheSeats(t *testing.T) {
	s := newStore(t)

	booking, err := s.Book("a-film", []string{"E7", "E8"})
	if err != nil {
		t.Fatalf("Book: %v", err)
	}
	if len(booking.Seats) != 2 {
		t.Fatalf("bought %d seats, want 2", len(booking.Seats))
	}
	if booking.TotalPriceCents != 2000 {
		t.Errorf("total %d, want 2000", booking.TotalPriceCents)
	}
	if booking.BookingId == "" {
		t.Error("a booking with no id")
	}

	seatMap, err := s.SeatMap("a-film")
	if err != nil {
		t.Fatalf("SeatMap: %v", err)
	}
	if seatMap.Sold != 2 {
		t.Errorf("%d seats sold, want 2", seatMap.Sold)
	}
}

// Buying is the one call that spends money, so it is all or nothing: a block
// with one seat gone sells none of it.
func TestBookIsAllOrNothing(t *testing.T) {
	s := newStore(t)

	if _, err := s.Book("a-film", []string{"E8"}); err != nil {
		t.Fatalf("first Book: %v", err)
	}

	_, err := s.Book("a-film", []string{"E7", "E8", "E9"})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("second Book: %v, want AlreadyExists", err)
	}

	seatMap, err := s.SeatMap("a-film")
	if err != nil {
		t.Fatalf("SeatMap: %v", err)
	}
	if seatMap.Sold != 1 {
		t.Errorf("%d seats sold, want only the first one", seatMap.Sold)
	}
}

func TestBookRefusesWhatTheHouseDoesNotSell(t *testing.T) {
	s := newStore(t)

	for _, test := range []struct {
		name  string
		seats []string
		want  codes.Code
	}{
		{"no seats", nil, codes.InvalidArgument},
		{"too many", []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7"}, codes.InvalidArgument},
		{"no such seat", []string{"Z99"}, codes.InvalidArgument},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := s.Book("a-film", test.seats); status.Code(err) != test.want {
				t.Errorf("Book(%v) = %v, want %s", test.seats, err, test.want)
			}
		})
	}
}

// A hold is not part of the API any more, but it is still what the crowd does
// and what keeps a seat out of everybody else's reach while it lasts.
func TestHeldSeatsCannotBeBought(t *testing.T) {
	s := newStore(t)

	holdID, err := s.Hold("a-film", []string{"E7"})
	if err != nil {
		t.Fatalf("Hold: %v", err)
	}

	if _, err := s.Book("a-film", []string{"E7"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("Book over a hold: %v, want AlreadyExists", err)
	}

	if err := s.Release(holdID); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, err := s.Book("a-film", []string{"E7"}); err != nil {
		t.Errorf("Book after the hold went: %v", err)
	}
}
