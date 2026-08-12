// Package crowd is the rest of the bakery. It claims racks and bakes through
// the same store the gRPC handlers use, which is what keeps the oven moving
// between two calls and means a visitor can genuinely lose a rack.
//
// It leaves room on purpose: the staff only take a rack while at least two are
// free, so there is normally one waiting for you. The oven still fills up when
// a bake is long, which is the point — "all four racks are busy" has to be
// something you can actually see.
package crowd

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/kaja-tools/website/v2/internal/bakebook"
	"github.com/kaja-tools/website/v2/internal/store"
)

// The morning shift. Their names are what the baker column shows, so a rack
// you did not claim says who did.
var bakers = []string{"Ines", "Marta", "Ollie", "Pim", "Rae"}

// Leave at least this many racks free after the staff take one.
const keepFree = 1

type Crowd struct {
	oven *store.Store
	book *bakebook.Client
}

func New(oven *store.Store, book *bakebook.Client) *Crowd {
	return &Crowd{oven: oven, book: book}
}

func (c *Crowd) Run(ctx context.Context) {
	slog.Info("the morning shift clocked in")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(4+rand.IntN(6)) * time.Second):
		}
		c.tick(ctx)
	}
}

func (c *Crowd) tick(ctx context.Context) {
	// Customers first: the counter has to come down as well as up.
	if n := int32(1 + rand.IntN(6)); n > 0 {
		c.oven.Sell(n)
	}

	oven := c.oven.Oven()
	if oven.FreeRacks <= keepFree {
		return
	}

	cookies, err := c.book.Cookies()
	if err != nil || len(cookies) == 0 {
		return
	}
	cookie := cookies[rand.IntN(len(cookies))]
	baker := bakers[rand.IntN(len(bakers))]

	claim, err := c.oven.Claim(cookie.ID, int32(1+rand.IntN(store.MaxTrays)), baker)
	if err != nil {
		return // beaten to the rack; the shift shrugs
	}

	// Bakers think for a moment, the way anybody does with a full tray in
	// their hands, and then mostly go through with it.
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(2+rand.IntN(6)) * time.Second):
		}
		if rand.Float64() < 0.85 {
			c.oven.Start(claim.Id) //nolint:errcheck // the claim may have expired
		} else {
			c.oven.Release(claim.Id) //nolint:errcheck
		}
	}()
}
