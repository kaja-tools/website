// Package crowd simulates the theatre's other customers. It books seats
// through the same store the gRPC handlers use, which keeps the seat map
// moving and means a visitor can genuinely lose a race for a seat.
//
// The crowd is a controller, not a firehose: each show fills toward a
// target occupancy that grows as curtain-up approaches (hot shows aim
// higher), and once a house is at target the crowd only window-shops —
// holding seats and letting them go — so the map stays lively without ever
// selling the place out.
package crowd

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/store"
	"github.com/kaja-tools/website/v2/internal/theatre"
)

// How much of the house a screening's crowd wants, at most. It is read off
// the ticket price rather than looked up by film, because the schedule is
// thousands of screenings long and a table of opinions about it would be
// stale by the weekend: a house charges more for the films it expects to
// fill, so the price already says what a table would have.
func demand(s theatre.Show) float64 {
	switch {
	case s.PriceCents >= 2000:
		return 0.85
	case s.PriceCents >= 1700:
		return 0.65
	case s.PriceCents >= 1400:
		return 0.5
	default:
		return 0.35
	}
}

type Crowd struct {
	seats   *store.Store
	theatre *theatre.Client
}

func New(seats *store.Store, theatreClient *theatre.Client) *Crowd {
	return &Crowd{seats: seats, theatre: theatreClient}
}

func (c *Crowd) Run(ctx context.Context) {
	slog.Info("crowd is queuing up")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(3+rand.Intn(5)) * time.Second):
		}
		c.tick(ctx)
	}
}

func (c *Crowd) tick(ctx context.Context) {
	shows, err := c.theatre.Shows()
	if err != nil || len(shows) == 0 {
		return
	}
	show := pickWeighted(shows)

	seatMap, err := c.seats.SeatMap(show.ID)
	if err != nil {
		return
	}
	capacity := seatMap.Available + seatMap.Held + seatMap.Sold
	if capacity == 0 {
		return
	}

	if float64(seatMap.Sold)/float64(capacity) < targetOccupancy(show) {
		c.buy(ctx, show.ID, seatMap)
	} else if rand.Float64() < 0.4 {
		c.windowShop(ctx, show.ID, seatMap)
	}
}

// targetOccupancy grows linearly from 0 (a week out) to the show's demand
// cap (at curtain-up).
func targetOccupancy(s theatre.Show) float64 {
	week := 7 * 24 * time.Hour
	untilShow := time.Until(s.StartsAt)
	if untilShow < 0 {
		untilShow = 0
	}
	closeness := 1 - float64(untilShow)/float64(week)
	if closeness < 0 {
		closeness = 0
	}
	return demand(s) * closeness
}

// pickWeighted prefers hot shows and near curtain-ups.
func pickWeighted(shows []theatre.Show) theatre.Show {
	weights := make([]float64, len(shows))
	total := 0.0
	for i, s := range shows {
		hoursUntil := time.Until(s.StartsAt).Hours()
		if hoursUntil < 0 {
			hoursUntil = 0
		}
		weights[i] = demand(s) / (1 + hoursUntil/24)
		total += weights[i]
	}
	roll := rand.Float64() * total
	for i, w := range weights {
		roll -= w
		if roll <= 0 {
			return shows[i]
		}
	}
	return shows[len(shows)-1]
}

// buy holds 1-4 adjacent seats, thinks about it like a real customer, then
// usually pays. Losing the seats to somebody faster is normal and fine.
func (c *Crowd) buy(ctx context.Context, showID string, seatMap *api.SeatMap) {
	seats := adjacentAvailable(seatMap, 1+rand.Intn(4))
	if len(seats) == 0 {
		return
	}
	holdID, err := c.seats.Hold(showID, seats)
	if err != nil {
		return // beaten to the seats; the crowd shrugs
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(4+rand.Intn(9)) * time.Second):
		}
		if rand.Float64() < 0.85 {
			c.seats.Confirm(holdID) //nolint:errcheck // the hold may have expired
		} else {
			c.seats.Release(holdID) //nolint:errcheck
		}
	}()
}

// windowShop holds seats and always lets them go, so a full house still has
// a moving seat map.
func (c *Crowd) windowShop(ctx context.Context, showID string, seatMap *api.SeatMap) {
	seats := adjacentAvailable(seatMap, 1+rand.Intn(2))
	if len(seats) == 0 {
		return
	}
	holdID, err := c.seats.Hold(showID, seats)
	if err != nil {
		return
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(5+rand.Intn(10)) * time.Second):
		}
		c.seats.Release(holdID) //nolint:errcheck
	}()
}

// adjacentAvailable finds a random run of n adjacent available seats,
// scanning rows in random order.
func adjacentAvailable(seatMap *api.SeatMap, n int) []string {
	var rows []*api.Row
	for _, section := range seatMap.Sections {
		rows = append(rows, section.Rows...)
	}
	rand.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })

	for _, row := range rows {
		run := []string{}
		for _, seat := range row.Seats {
			if seat.Status == api.SeatStatus_SEAT_STATUS_AVAILABLE {
				run = append(run, seat.Id)
				if len(run) == n {
					return run
				}
			} else {
				run = run[:0]
			}
		}
		if len(run) > 0 {
			return run // settle for a shorter run
		}
	}
	return nil
}
