package store

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
	"github.com/kaja-tools/website/v2/internal/bakebook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const perTray = 10

// testStore stands a whole bakery up against a one-recipe book, running fast
// enough that a bake finishes between two lines of a test.
func testStore(t *testing.T) *Store {
	t.Helper()
	book := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]bakebook.Cookie{{
			ID:                  "test-cookie",
			Name:                "Test Cookie",
			BakeMinutes:         2,
			OvenCelsius:         190,
			PerTray:             perTray,
			DoughGramsPerCookie: 40,
		}})
	}))
	t.Cleanup(book.Close)

	return New(bakebook.NewClient(book.URL), Pace{
		BakeUnit: 5 * time.Millisecond,
		CoolFor:  5 * time.Millisecond,
		ClaimTTL: 50 * time.Millisecond,
	})
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func codeOf(err error) codes.Code {
	return status.Code(err)
}

// A batch goes claim -> bake -> cool -> counter, and every cookie in it is
// accounted for exactly once at the end, whether or not the tray scorched.
func TestClaimThenBake(t *testing.T) {
	s := testStore(t)

	claim, err := s.Claim("test-cookie", 2, "you")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claim.Cookies != 2*perTray {
		t.Errorf("cookies = %d, want %d", claim.Cookies, 2*perTray)
	}
	if claim.DoughGrams != 2*perTray*40 {
		t.Errorf("dough = %dg, want %dg", claim.DoughGrams, 2*perTray*40)
	}

	oven := s.Oven()
	if oven.FreeRacks != Racks-1 {
		t.Errorf("free racks = %d, want %d", oven.FreeRacks, Racks-1)
	}
	if got := oven.Racks[claim.RackNumber-1]; got.State != api.RackState_RACK_STATE_CLAIMED || got.Baker != "you" {
		t.Errorf("rack = %v/%q, want CLAIMED/you", got.State, got.Baker)
	}

	if _, err := s.Start(claim.Id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitFor(t, "the rack to empty", func() bool { return s.Oven().FreeRacks == Racks })

	oven = s.Oven()
	if total := oven.OnCounter + oven.ScorchedToday; total != 2*perTray {
		t.Errorf("counter+scorched = %d, want %d", total, 2*perTray)
	}
	if oven.Cooling != 0 {
		t.Errorf("cooling = %d, want 0", oven.Cooling)
	}
	// A claim is spent by baking it, so it cannot be baked twice.
	if _, err := s.Start(claim.Id); codeOf(err) != codes.NotFound {
		t.Errorf("second Start = %v, want NotFound", codeOf(err))
	}
}

// Four racks is the whole constraint: the fifth baker is told to wait.
func TestOvenFillsUp(t *testing.T) {
	s := testStore(t)
	for i := 0; i < Racks; i++ {
		if _, err := s.Claim("test-cookie", 1, "you"); err != nil {
			t.Fatalf("Claim %d: %v", i, err)
		}
	}
	if _, err := s.Claim("test-cookie", 1, "you"); codeOf(err) != codes.ResourceExhausted {
		t.Errorf("Claim on a full oven = %v, want ResourceExhausted", codeOf(err))
	}
}

func TestClaimRejectsWhatItCannotBake(t *testing.T) {
	s := testStore(t)

	if _, err := s.Claim("sourdough", 1, "you"); codeOf(err) != codes.NotFound {
		t.Errorf("unknown cookie = %v, want NotFound", codeOf(err))
	}
	if _, err := s.Claim("test-cookie", MaxTrays+1, "you"); codeOf(err) != codes.InvalidArgument {
		t.Errorf("too many trays = %v, want InvalidArgument", codeOf(err))
	}
	// No trays means one tray, which is what anybody means.
	claim, err := s.Claim("test-cookie", 0, "you")
	if err != nil {
		t.Fatalf("Claim with no trays: %v", err)
	}
	if claim.Trays != 1 {
		t.Errorf("trays = %d, want 1", claim.Trays)
	}
}

// A rack nobody bakes on comes back on its own.
func TestClaimExpires(t *testing.T) {
	s := testStore(t)

	claim, err := s.Claim("test-cookie", 1, "you")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	waitFor(t, "the claim to expire", func() bool { return s.Oven().FreeRacks == Racks })

	if _, err := s.Start(claim.Id); codeOf(err) != codes.NotFound {
		t.Errorf("Start on an expired claim = %v, want NotFound", codeOf(err))
	}
	if oven := s.Oven(); oven.BakedToday != 0 || oven.OnCounter != 0 {
		t.Errorf("an expired claim baked something: %d/%d", oven.BakedToday, oven.OnCounter)
	}
}

// A watcher gets the whole life of a batch, in order.
func TestWatchOven(t *testing.T) {
	s := testStore(t)

	_, updates, cancel := s.Subscribe()
	defer cancel()

	claim, err := s.Claim("test-cookie", 1, "you")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if _, err := s.Start(claim.Id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	want := []api.ChangeReason{api.ChangeReason_CHANGE_REASON_CLAIMED, api.ChangeReason_CHANGE_REASON_STARTED}
	for _, reason := range want {
		select {
		case update := <-updates:
			if got := update.GetChange().GetReason(); got != reason {
				t.Fatalf("reason = %v, want %v", got, reason)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for %v", reason)
		}
	}
}

// The counter only ever holds cookies somebody could actually buy.
func TestSellNeverGoesNegative(t *testing.T) {
	s := testStore(t)
	if sold := s.Sell(5); sold != 0 {
		t.Errorf("sold %d from an empty counter, want 0", sold)
	}
	if on := s.Oven().OnCounter; on != 0 {
		t.Errorf("counter = %d, want 0", on)
	}
}
