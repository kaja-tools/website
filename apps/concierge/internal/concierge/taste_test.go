package concierge

import (
	"strings"
	"testing"
	"time"

	"github.com/kaja-tools/website/v2/internal/api"
)

// A programme is already joined by the time taste sees it: every screening
// carries its film and its house.
func screening(id string, in Theater, hours int, movie Movie) Screening {
	return Screening{ID: id, StartsAt: time.Now().Add(time.Duration(hours) * time.Hour), Movie: movie, Theater: in}
}

var (
	uptown    = Theater{ID: "uptown", Name: "The Uptown", City: "Chicago", State: "IL"}
	seaside   = Theater{ID: "seaside", Name: "The Seaside", City: "Miami", State: "FL"}
	programme = []Screening{
		screening("loud-one", uptown, 72, Movie{ID: "loud", Title: "Loud One", Genre: "action", Year: 2015, RuntimeMinutes: 120, Language: "English", Synopsis: "Cars."}),
		screening("quiet-one", seaside, 96, Movie{ID: "quiet", Title: "Quiet One", Genre: "romance", Year: 2000, RuntimeMinutes: 98, Language: "Cantonese", Synopsis: "Neighbours."}),
		screening("funny-one", uptown, 24, Movie{ID: "funny", Title: "Funny One", Genre: "comedy", Year: 2014, RuntimeMinutes: 100, Language: "English", Synopsis: "A hotel."}),
		screening("long-one", seaside, 48, Movie{ID: "long", Title: "Long One", Genre: "drama", Year: 1954, RuntimeMinutes: 207, Language: "Japanese", Synopsis: "Swords."}),
	}
)

func TestSuggestReadsTheRequest(t *testing.T) {
	for _, test := range []struct {
		mood string
		want string
	}{
		{"something loud", "loud-one"},
		{"quiet and sad", "quiet-one"},
		{"make me laugh", "funny-one"},
		{"an old classic", "long-one"},
		{"subtitled, and long", "long-one"},
	} {
		t.Run(test.mood, func(t *testing.T) {
			matches := Suggest(programme, test.mood, 0)
			if len(matches) == 0 {
				t.Fatal("no suggestion at all")
			}
			if matches[0].Screening.ID != test.want {
				t.Errorf("suggested %s, want %s", matches[0].Screening.ID, test.want)
			}
			if matches[0].Why == "" {
				t.Error("a suggestion with no case for it")
			}
		})
	}
}

// A request the concierge understands nothing of is still answered — with
// whatever starts next, which is what somebody with no information would say.
func TestSuggestAlwaysAnswers(t *testing.T) {
	matches := Suggest(programme, "hmm", 0)
	if len(matches) != len(programme) {
		t.Fatalf("ranked %d of %d films", len(matches), len(programme))
	}
	if matches[0].Screening.ID != "funny-one" {
		t.Errorf("fell back to %s, want the soonest screening", matches[0].Screening.ID)
	}
}

// The same film is on in four towns this week, so which town somebody is in
// is a filter over the programme rather than a taste in it.
func TestInCityNarrowsToOneTown(t *testing.T) {
	inChicago := InCity(programme, "chicago")
	if len(inChicago) != 2 {
		t.Fatalf("%d screenings in Chicago, want 2", len(inChicago))
	}
	for _, s := range inChicago {
		if s.Theater.City != "Chicago" {
			t.Errorf("%s is at %s", s.ID, s.Theater.Where())
		}
	}
	if len(InCity(programme, "")) != len(programme) {
		t.Error("no city asked for should leave the programme alone")
	}
	if got := InCity(programme, "Boise"); len(got) != 0 {
		t.Errorf("found %d screenings in a town with no house", len(got))
	}
}

func TestSuggestHonoursTheClock(t *testing.T) {
	matches := Suggest(programme, "anything", 100)
	for _, match := range matches {
		if match.Screening.Movie.RuntimeMinutes > 100 {
			t.Errorf("%s runs %d minutes, over the 100 asked for", match.Screening.ID, match.Screening.Movie.RuntimeMinutes)
		}
	}
	if len(matches) != 2 {
		t.Errorf("offered %d films under 100 minutes, want 2", len(matches))
	}
}

// A house of two sections, eight rows of ten in the orchestra and two in the
// balcony, with everything free unless a test takes it away.
func house(taken ...string) *api.SeatMap {
	gone := map[string]bool{}
	for _, id := range taken {
		gone[id] = true
	}
	seatMap := &api.SeatMap{ShowId: "a-film"}
	for _, section := range []struct {
		section api.Section
		rows    string
	}{
		{api.Section_SECTION_ORCHESTRA, "ABCDEFGH"},
		{api.Section_SECTION_BALCONY, "JK"},
	} {
		sectionMap := &api.SectionMap{Section: section.section}
		for _, letter := range strings.Split(section.rows, "") {
			row := &api.Row{Letter: letter}
			for n := 1; n <= 10; n++ {
				id := letter + string(rune('0'+n%10))
				if n == 10 {
					id = letter + "10"
				}
				status := api.SeatStatus_SEAT_STATUS_AVAILABLE
				if gone[id] {
					status = api.SeatStatus_SEAT_STATUS_SOLD
				}
				row.Seats = append(row.Seats, &api.Seat{
					Id: id, Section: section.section, Row: letter,
					Number: int32(n), Status: status, PriceCents: 1000,
				})
			}
			sectionMap.Rows = append(sectionMap.Rows, row)
		}
		seatMap.Sections = append(seatMap.Sections, sectionMap)
	}
	return seatMap
}

func TestPickSeatsTakesTheMiddleOfTheHouse(t *testing.T) {
	seats, ok := PickSeats(house(), 2)
	if !ok {
		t.Fatal("found no seats in an empty house")
	}
	if seats.Section != "ORCHESTRA" {
		t.Errorf("picked the %s over the orchestra", seats.Section)
	}
	// Eight rows, so the row 60% of the way back is E.
	if seats.Row != "E" {
		t.Errorf("picked row %s, want E", seats.Row)
	}
	if strings.Join(seats.IDs, ",") != "E5,E6" {
		t.Errorf("picked %v, want the middle of the row", seats.IDs)
	}
	if seats.TotalCents != 2000 {
		t.Errorf("total %d, want 2000", seats.TotalCents)
	}
}

func TestPickSeatsKeepsThePartyTogether(t *testing.T) {
	// Every row gapped in the middle, so no row seats four side by side
	// except the ends.
	var taken []string
	for _, letter := range strings.Split("ABCDEFGHJK", "") {
		taken = append(taken, letter+"5", letter+"6")
	}

	seats, ok := PickSeats(house(taken...), 4)
	if !ok {
		t.Fatal("found no block of four")
	}
	if len(seats.IDs) != 4 {
		t.Fatalf("picked %d seats, want 4", len(seats.IDs))
	}
	for _, id := range seats.IDs {
		if strings.HasSuffix(id, "5") || strings.HasSuffix(id, "6") {
			t.Errorf("picked %s, which is sold", id)
		}
	}
}

func TestPickSeatsSaysWhenThereIsNoRoom(t *testing.T) {
	var taken []string
	for _, letter := range strings.Split("ABCDEFGHJK", "") {
		for n := 1; n <= 10; n += 2 {
			taken = append(taken, letter+string(rune('0'+n)))
		}
	}

	// Every other seat gone: 50 seats free, and nowhere for two to sit together.
	if _, ok := PickSeats(house(taken...), 2); ok {
		t.Error("claimed to seat two together in a house with no pair left")
	}
}
