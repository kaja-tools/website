// Package catalog holds everything the Theatre publishes, which is three
// lists: the movies, the theaters that screen them, and the week's schedule
// relating the two.
//
// Only the third is a relationship, and it is deliberately nothing but one. A
// Show is a movie id, a theater id, a time and a price — no title, no
// address — so anybody reading a programme has to join it against the other
// two lists. That join is what the demo is here to show.
//
// Nothing is stored. The movies are films.json, embedded at build time; the
// theaters are the dozen houses in theaters.go; and every screening is
// derived from a movie and a house — which of them play it, when, and for how
// much — so there is no schedule to keep in step and the week never runs out.
//
// The films are real. Every fact about one — director, year, running time,
// language, and the sentence of synopsis — is taken from Wikidata (CC0) and
// English Wikipedia (CC BY-SA 4.0) by scripts/catalog. The chain is invented,
// and so is everything it decides: which house plays what, at what time, and
// for what money.
package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"time"
)

//go:embed films.json
var filmsJSON []byte

// Movie is one film in the catalog. It says nothing about when or where it is
// playing: a movie is in the catalog whether or not it is on this week.
type Movie struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Director       string `json:"director"`
	Year           int    `json:"year"`
	RuntimeMinutes int    `json:"runtimeMinutes"`
	Genre          string `json:"genre"`
	Language       string `json:"language"`
	Synopsis       string `json:"synopsis"`
}

// Show is one weekly screening: a movie, at a theater, at a time, for a
// price. Its id is what the seating service takes, and it is the two ids it
// relates written out — `dune-part-two@the-lantern` — so a screening says
// what it is without anything having to be looked up to read it.
type Show struct {
	ID         string
	MovieID    string
	TheaterID  string
	PriceCents int

	// The weekly slot, in the theater's own local time.
	Weekday time.Weekday
	Hour    int
	Minute  int

	loc *time.Location
}

var (
	movies       []Movie
	moviesByID   map[string]Movie
	shows        []Show
	showsByID    map[string]Show
	showsByMovie map[string][]Show
)

func init() {
	loadTheaters()
	loadMovies()
	loadShows()
}

func loadMovies() {
	if err := json.Unmarshal(filmsJSON, &movies); err != nil {
		panic(fmt.Sprintf("catalog: films.json: %v", err))
	}
	sort.Slice(movies, func(i, j int) bool { return movies[i].ID < movies[j].ID })
	moviesByID = make(map[string]Movie, len(movies))
	for _, m := range movies {
		if _, clash := moviesByID[m.ID]; clash {
			panic(fmt.Sprintf("catalog: two films called %q", m.ID))
		}
		moviesByID[m.ID] = m
	}
}

func loadShows() {
	showsByID = make(map[string]Show)
	showsByMovie = make(map[string][]Show)
	for _, m := range movies {
		for _, t := range housesFor(m.ID) {
			s := Show{
				ID:         m.ID + "@" + t.ID,
				MovieID:    m.ID,
				TheaterID:  t.ID,
				PriceCents: price(m, t),
				loc:        t.loc,
			}
			s.Weekday, s.Hour, s.Minute = slot(s.ID)
			shows = append(shows, s)
			showsByID[s.ID] = s
			showsByMovie[m.ID] = append(showsByMovie[m.ID], s)
		}
	}
}

// Which houses play a film, and how many of them. A film runs at one, two or
// three of the twelve, read off its own id, so the schedule is the same every
// week and is written down nowhere.
func housesFor(movieID string) []Theater {
	n := hash(movieID)
	count := 1 + int(n%3)
	index := int(n/3) % len(theaters)
	stride := 1 + int(n/9)%(len(theaters)-1)

	out := make([]Theater, 0, count)
	taken := map[int]bool{}
	for len(out) < count {
		for taken[index] {
			index = (index + 1) % len(theaters)
		}
		taken[index] = true
		out = append(out, theaters[index])
		index = (index + stride) % len(theaters)
	}
	return out
}

// A house runs from late morning to the last late show, on the quarter hour.
// A screening's slot is its id hashed into that grid: stable week to week,
// spread across the week, and different for the same film two towns over.
const (
	firstHour  = 11
	hoursOpen  = 12
	quarters   = 4
	daysOfWeek = 7
)

func slot(showID string) (time.Weekday, int, int) {
	n := hash(showID)
	hour := firstHour + int(n%hoursOpen)
	minute := 15 * int((n/hoursOpen)%quarters)
	weekday := time.Weekday((n / (hoursOpen * quarters)) % daysOfWeek)
	return weekday, hour, minute
}

// What a seat costs. A new print costs more than an old one and a long
// evening costs more than a short one; then the house takes its own view of
// what it can charge, which is the only thing about a price that is the
// theater's rather than the film's. Rounded to the quarter, because nobody
// prints a ticket at $16.38.
func price(m Movie, t Theater) int {
	cents := 1200
	switch {
	case m.Year >= 2015:
		cents = 1800
	case m.Year >= 2000:
		cents = 1600
	case m.Year >= 1980:
		cents = 1400
	}
	if m.RuntimeMinutes >= 150 {
		cents += 100
	}
	cents = cents * t.priceIndex / 100
	return (cents + 12) / 25 * 25
}

func hash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func Movies() []Movie { return movies }

func MovieByID(id string) (Movie, bool) {
	m, ok := moviesByID[id]
	return m, ok
}

func Shows() []Show { return shows }

func ShowByID(id string) (Show, bool) {
	s, ok := showsByID[id]
	return s, ok
}

// ShowsOf is every screening of one film, across the chain.
func ShowsOf(movieID string) []Show { return showsByMovie[movieID] }

// NextStart is the screening's next start after now, in the theater's own
// local time. A weekly slot always has one, so every screening in the
// schedule is always on sale.
func (s Show) NextStart(now time.Time) time.Time {
	local := now.In(s.loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc)
	daysAhead := (int(s.Weekday) - int(midnight.Weekday()) + 7) % 7

	day := midnight.AddDate(0, 0, daysAhead)
	start := time.Date(day.Year(), day.Month(), day.Day(), s.Hour, s.Minute, 0, 0, s.loc)
	if !start.After(now) {
		day = day.AddDate(0, 0, 7)
		start = time.Date(day.Year(), day.Month(), day.Day(), s.Hour, s.Minute, 0, 0, s.loc)
	}
	return start
}
