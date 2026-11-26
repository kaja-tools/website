package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kaja-tools/website/v2/internal/catalog"
)

func get(t *testing.T, path string, into any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	New().Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s = %d: %s", path, recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), into); err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
}

// A cursor is a position in a list rather than a page number, so walking one
// to the end sees every row exactly once — which is the property the whole
// demo's paging rests on.
func TestWalkingTheScheduleSeesEveryScreeningOnce(t *testing.T) {
	seen := map[string]bool{}
	total := 0
	for cursor, requests := "", 0; ; requests++ {
		if requests > 10 {
			t.Fatal("the schedule for one house did not end")
		}
		var page showPage
		url := "/shows?theaterId=the-lantern&limit=50"
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		get(t, url, &page)
		total = page.Total

		for _, show := range page.Shows {
			if seen[show.ID] {
				t.Errorf("%s came back twice", show.ID)
			}
			seen[show.ID] = true
			if show.TheaterID != "the-lantern" {
				t.Errorf("%s is at %s, not the house asked for", show.ID, show.TheaterID)
			}
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	if len(seen) != total {
		t.Errorf("walked %d screenings, total said %d", len(seen), total)
	}
	if len(seen) == 0 {
		t.Error("nothing is playing at a house in the chain")
	}
}

// The schedule carries two ids and nothing else, so a page of it is only
// readable joined against the other two lists. Every id it publishes has to
// resolve, or the join has a hole in it.
func TestEveryScreeningJoinsToAFilmAndAHouse(t *testing.T) {
	var page showPage
	get(t, "/shows?limit=100", &page)
	if len(page.Shows) != 100 {
		t.Fatalf("asked for 100 screenings, got %d", len(page.Shows))
	}

	var catalogPage moviePage
	get(t, "/movies?limit=500&ids="+ids(page), &catalogPage)
	films := map[string]bool{}
	for _, movie := range catalogPage.Movies {
		films[movie.ID] = true
	}

	var chain theaterList
	get(t, "/theaters", &chain)
	houses := map[string]bool{}
	for _, theater := range chain.Theaters {
		houses[theater.ID] = true
	}

	for _, show := range page.Shows {
		if !films[show.MovieID] {
			t.Errorf("%s names a film the catalog does not have", show.ID)
		}
		if !houses[show.TheaterID] {
			t.Errorf("%s names a house the chain does not have", show.ID)
		}
		if !show.StartsAt.After(time.Now()) {
			t.Errorf("%s starts at %s, which has been and gone", show.ID, show.StartsAt)
		}
	}
}

func ids(page showPage) string {
	out := ""
	for _, show := range page.Shows {
		if out != "" {
			out += ","
		}
		out += show.MovieID
	}
	return out
}

// The chain is small enough to answer in one response, which is what makes it
// a lookup table rather than a third thing to walk.
func TestTheChainIsOneResponse(t *testing.T) {
	var chain theaterList
	get(t, "/theaters", &chain)
	if len(chain.Theaters) != len(catalog.Theaters()) {
		t.Errorf("returned %d houses of %d", len(chain.Theaters), len(catalog.Theaters()))
	}

	var chicago theaterList
	get(t, "/theaters?city=chicago", &chicago)
	if len(chicago.Theaters) == 0 {
		t.Fatal("no house in a city the chain is in")
	}
	for _, theater := range chicago.Theaters {
		if theater.City != "Chicago" {
			t.Errorf("%s is in %s", theater.ID, theater.City)
		}
	}
}

func TestFiltersNarrowTheSchedule(t *testing.T) {
	var everywhere showPage
	get(t, "/shows?movieId=dune-part-two", &everywhere)
	if len(everywhere.Shows) == 0 {
		t.Fatal("a film in the catalog is playing nowhere")
	}
	if everywhere.Total != len(everywhere.Shows) {
		t.Errorf("total %d over %d rows — total counts what matched", everywhere.Total, len(everywhere.Shows))
	}
	for _, show := range everywhere.Shows {
		if show.MovieID != "dune-part-two" {
			t.Errorf("%s is not the film asked for", show.ID)
		}
	}

	var chicago showPage
	get(t, "/shows?city=Chicago&limit=5", &chicago)
	for _, show := range chicago.Shows {
		theater, ok := catalog.TheaterByID(show.TheaterID)
		if !ok || theater.City != "Chicago" {
			t.Errorf("%s is not in the city asked for", show.ID)
		}
	}
}

func TestBadRequestsSayWhy(t *testing.T) {
	for _, test := range []struct{ name, path string }{
		{"limit over the maximum", "/shows?limit=501"},
		{"limit that is not a number", "/movies?limit=lots"},
		{"a cursor from somewhere else", "/shows?cursor=nonsense"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			New().Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("GET %s = %d, want 400", test.path, recorder.Code)
			}
			var body struct {
				Message string `json:"message"`
			}
			json.Unmarshal(recorder.Body.Bytes(), &body)
			if body.Message == "" {
				t.Error("a refusal with nothing to act on")
			}
		})
	}
}
