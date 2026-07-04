// Package crowd simulates the theatre's other customers. It buys seats
// through the same seating gRPC API that kaja visitors use, which keeps the
// live seat map dramatic and means a visitor can genuinely lose a race for
// a seat.
//
// The crowd is a controller, not a firehose: each performance fills toward
// a target occupancy that grows as showtime approaches (hot shows aim
// higher), and once a house is at target the crowd only window-shops —
// holding seats and letting them go — so streams stay lively without ever
// selling the place out.
package crowd

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/theatre"
)

// How much of the house each event's crowd wants, at most.
var demand = map[string]float64{
	"neon-meridian":          0.85,
	"milo-frey":              0.70,
	"glass-mountain":         0.60,
	"cartographers-daughter": 0.55,
	"vera-lune":              0.50,
	"kaja-players":           0.45,
	"twelve-clocks":          0.35,
}

const defaultDemand = 0.5

type Crowd struct {
	seating api.SeatingClient
	theatre *theatre.Client
}

func New(seating api.SeatingClient, theatreClient *theatre.Client) *Crowd {
	return &Crowd{seating: seating, theatre: theatreClient}
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
	perfs, err := c.theatre.Upcoming()
	if err != nil || len(perfs) == 0 {
		return
	}
	perf := pickWeighted(perfs)

	mapResp, err := c.seating.GetSeatMap(ctx, &api.GetSeatMapRequest{PerformanceId: perf.ID})
	if err != nil {
		return
	}
	seatMap := mapResp.SeatMap
	capacity := seatMap.Available + seatMap.Held + seatMap.Sold
	if capacity == 0 {
		return
	}

	if float64(seatMap.Sold)/float64(capacity) < targetOccupancy(perf) {
		c.buy(ctx, perf.ID, seatMap)
	} else if rand.Float64() < 0.4 {
		c.windowShop(ctx, perf.ID, seatMap)
	}
}

// targetOccupancy grows linearly from 0 (a week out) to the event's demand
// cap (at showtime).
func targetOccupancy(p theatre.Performance) float64 {
	d, ok := demand[p.EventID]
	if !ok {
		d = defaultDemand
	}
	week := 7 * 24 * time.Hour
	untilShow := time.Until(p.StartsAt)
	if untilShow < 0 {
		untilShow = 0
	}
	closeness := 1 - float64(untilShow)/float64(week)
	if closeness < 0 {
		closeness = 0
	}
	return d * closeness
}

// pickWeighted prefers hot shows and near showtimes.
func pickWeighted(perfs []theatre.Performance) theatre.Performance {
	weights := make([]float64, len(perfs))
	total := 0.0
	for i, p := range perfs {
		d, ok := demand[p.EventID]
		if !ok {
			d = defaultDemand
		}
		hoursUntil := time.Until(p.StartsAt).Hours()
		if hoursUntil < 0 {
			hoursUntil = 0
		}
		weights[i] = d / (1 + hoursUntil/24)
		total += weights[i]
	}
	roll := rand.Float64() * total
	for i, w := range weights {
		roll -= w
		if roll <= 0 {
			return perfs[i]
		}
	}
	return perfs[len(perfs)-1]
}

// buy holds 1-4 adjacent seats, thinks about it like a real customer, then
// usually pays. Losing the seats to somebody faster is normal and fine.
func (c *Crowd) buy(ctx context.Context, perfID string, seatMap *api.SeatMap) {
	seats := adjacentAvailable(seatMap, 1+rand.Intn(4))
	if len(seats) == 0 {
		return
	}
	resp, err := c.seating.HoldSeats(ctx, &api.HoldSeatsRequest{PerformanceId: perfID, SeatIds: seats})
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
			c.seating.ConfirmSeats(ctx, &api.ConfirmSeatsRequest{HoldId: resp.Hold.Id}) //nolint:errcheck
		} else {
			c.seating.ReleaseSeats(ctx, &api.ReleaseSeatsRequest{HoldId: resp.Hold.Id}) //nolint:errcheck
		}
	}()
}

// windowShop holds seats and always lets them go, so a full house still has
// a moving seat map.
func (c *Crowd) windowShop(ctx context.Context, perfID string, seatMap *api.SeatMap) {
	seats := adjacentAvailable(seatMap, 1+rand.Intn(2))
	if len(seats) == 0 {
		return
	}
	resp, err := c.seating.HoldSeats(ctx, &api.HoldSeatsRequest{PerformanceId: perfID, SeatIds: seats})
	if err != nil {
		return
	}
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(5+rand.Intn(10)) * time.Second):
		}
		c.seating.ReleaseSeats(ctx, &api.ReleaseSeatsRequest{HoldId: resp.Hold.Id}) //nolint:errcheck
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
