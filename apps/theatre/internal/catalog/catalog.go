// Package catalog holds The Kaja Theatre's programme — a repertory cinema in
// a restored movie palace, so the films are real ones and the house has a
// balcony. Every film screens once a week in the same slot, so its next
// showtime is derived from the current time: no storage, and the demo can
// never run out of screenings.
//
// A screening is the unit the whole demo is built on: its id is what you pass
// to the seating service. There is no separate "film" and "screening".
//
// The repertory is films.json, embedded at build time. Every fact in it —
// director, year, running time, language, and the sentence of synopsis — is
// real, taken from Wikidata (CC0) and English Wikipedia (CC BY-SA 4.0) by
// scripts/catalog. Nothing about a film is invented, which leaves only the
// two things a cinema decides for itself: the slot it screens in and what it
// charges, and both are derived below from the film's own id rather than
// written down, so the programme has nothing to keep in step.
package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"
)

//go:embed films.json
var filmsJSON []byte

type Show struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Director       string `json:"director"`
	Year           int    `json:"year"`
	RuntimeMinutes int    `json:"runtimeMinutes"`
	Genre          string `json:"genre"`
	Language       string `json:"language"`
	Synopsis       string `json:"synopsis"`

	BasePriceCents int `json:"-"`
	// The weekly slot. Times are UTC.
	Weekday time.Weekday `json:"-"`
	Hour    int          `json:"-"`
	Minute  int          `json:"-"`
}

var (
	shows []Show
	byID  map[string]Show
)

func init() {
	if err := json.Unmarshal(filmsJSON, &shows); err != nil {
		panic(fmt.Sprintf("catalog: films.json: %v", err))
	}
	byID = make(map[string]Show, len(shows))
	for i := range shows {
		show := &shows[i]
		show.Weekday, show.Hour, show.Minute = slot(show.ID)
		show.BasePriceCents = price(*show)
		if _, clash := byID[show.ID]; clash {
			panic(fmt.Sprintf("catalog: two screenings called %q", show.ID))
		}
		byID[show.ID] = *show
	}
}

// The house runs from late morning to the last late show, on the quarter
// hour. A film's slot is its id hashed into that grid: stable week to week,
// spread across the week, and nowhere written down.
const (
	firstHour  = 10
	hoursOpen  = 14
	quarters   = 4
	daysOfWeek = 7
)

func slot(id string) (time.Weekday, int, int) {
	h := fnv.New64a()
	h.Write([]byte(id))
	n := h.Sum64()
	hour := firstHour + int(n%hoursOpen)
	minute := 15 * int((n/hoursOpen)%quarters)
	weekday := time.Weekday((n / (hoursOpen * quarters)) % daysOfWeek)
	return weekday, hour, minute
}

// What the house charges for an orchestra seat. A new print costs more than
// an old one and a long evening costs more than a short one, which is the
// whole of the pricing policy.
func price(s Show) int {
	cents := 1200
	switch {
	case s.Year >= 2015:
		cents = 1800
	case s.Year >= 2000:
		cents = 1600
	case s.Year >= 1980:
		cents = 1400
	}
	if s.RuntimeMinutes >= 150 {
		cents += 100
	}
	return cents
}

func Shows() []Show { return shows }

func ByID(id string) (Show, bool) {
	s, ok := byID[id]
	return s, ok
}

// NextStart is the screening's next start after now. A weekly slot always has
// one, so every film in the programme is always on sale.
func (s Show) NextStart(now time.Time) time.Time {
	now = now.UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	daysAhead := (int(s.Weekday) - int(midnight.Weekday()) + 7) % 7
	start := midnight.AddDate(0, 0, daysAhead).
		Add(time.Duration(s.Hour)*time.Hour + time.Duration(s.Minute)*time.Minute)
	if !start.After(now) {
		start = start.AddDate(0, 0, 7)
	}
	return start
}
