// Package catalog holds The Kaja Theatre's repertoire. The venue has a
// single stage, so the weekly grid below is conflict-free, and the rolling
// 7-day schedule is derived deterministically from the current date —
// no storage needed, and the demo can never run out of shows.
package catalog

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const WindowDays = 7

const (
	StatusOnSale = "onSale"
	StatusPast   = "past"
)

type CastMember struct {
	Actor string `json:"actor"`
	Role  string `json:"role"`
}

// Details is the genre-specific part of an event. Exactly one group of
// fields is populated, matching the oneOf in openapi.yaml.
type Details struct {
	Type string `json:"type"`
	// concert
	Headliner       string   `json:"headliner,omitempty"`
	SupportActs     []string `json:"supportActs,omitempty"`
	DurationMinutes int      `json:"durationMinutes,omitempty"`
	// play
	Playwright string       `json:"playwright,omitempty"`
	Director   string       `json:"director,omitempty"`
	Cast       []CastMember `json:"cast,omitempty"`
	// comedy
	Comedian string   `json:"comedian,omitempty"`
	Openers  []string `json:"openers,omitempty"`
	// opera
	Composer  string `json:"composer,omitempty"`
	Language  string `json:"language,omitempty"`
	Surtitles *bool  `json:"surtitles,omitempty"`
}

type Showtime struct {
	Weekday time.Weekday
	Hour    int
	Minute  int
}

type Event struct {
	ID             string
	Title          string
	Genre          string
	Tagline        string
	Description    string
	Details        Details
	BasePriceCents int
	Showtimes      []Showtime
}

type Performance struct {
	ID             string    `json:"id"`
	EventID        string    `json:"eventId"`
	EventTitle     string    `json:"eventTitle"`
	StartsAt       time.Time `json:"startsAt"`
	DoorsOpenAt    time.Time `json:"doorsOpenAt"`
	BasePriceCents int       `json:"basePriceCents"`
	Status         string    `json:"status"`
}

type VenueRow struct {
	Letter string `json:"letter"`
	Seats  int    `json:"seats"`
}

type VenueSection struct {
	Name            string     `json:"name"`
	PriceMultiplier float64    `json:"priceMultiplier"`
	Rows            []VenueRow `json:"rows"`
}

type Venue struct {
	Name     string         `json:"name"`
	Address  string         `json:"address"`
	Capacity int            `json:"capacity"`
	Sections []VenueSection `json:"sections"`
}

func TheVenue() Venue {
	orchestra := VenueSection{Name: "ORCHESTRA", PriceMultiplier: 1.0}
	for _, letter := range strings.Split("A B C D E F G H", " ") {
		orchestra.Rows = append(orchestra.Rows, VenueRow{Letter: letter, Seats: 14})
	}
	balcony := VenueSection{Name: "BALCONY", PriceMultiplier: 0.75}
	for _, letter := range strings.Split("J K L M", " ") {
		balcony.Rows = append(balcony.Rows, VenueRow{Letter: letter, Seats: 12})
	}
	return Venue{
		Name:     "The Kaja Theatre",
		Address:  "12 Meridian Lane",
		Capacity: 8*14 + 4*12,
		Sections: []VenueSection{orchestra, balcony},
	}
}

func boolPtr(b bool) *bool { return &b }

var events = []Event{
	{
		ID:      "neon-meridian",
		Title:   "Neon Meridian: World Tour",
		Genre:   "concert",
		Tagline: "Synthwave for the end of the night.",
		Description: "After two sold-out continents, **Neon Meridian** brings the World Tour home.\n\n" +
			"Expect walls of analog synth, a light rig that needed its own truck, and the " +
			"encore everyone films instead of watching.",
		Details: Details{
			Type:            "concert",
			Headliner:       "Neon Meridian",
			SupportActs:     []string{"Glass Harbor", "MOTH"},
			DurationMinutes: 110,
		},
		BasePriceCents: 8500,
		Showtimes: []Showtime{
			{time.Friday, 19, 30},
			{time.Saturday, 21, 0},
		},
	},
	{
		ID:      "vera-lune",
		Title:   "An Evening with Vera Lune",
		Genre:   "concert",
		Tagline: "One voice, one piano, no microphone.",
		Description: "Jazz's quietest superstar performs entirely unamplified — the room was " +
			"built for it, and **Vera Lune** knows exactly what to do with it.",
		Details: Details{
			Type:            "concert",
			Headliner:       "Vera Lune",
			DurationMinutes: 95,
		},
		BasePriceCents: 6500,
		Showtimes: []Showtime{
			{time.Wednesday, 20, 0},
		},
	},
	{
		ID:      "cartographers-daughter",
		Title:   "The Cartographer's Daughter",
		Genre:   "play",
		Tagline: "A map of everywhere she never went.",
		Description: "Elena Brook's award-winning play about a daughter who inherits ten " +
			"thousand hand-drawn maps and a father she never understood. Bring tissues; " +
			"the Saturday matinee crowd will tell you why.",
		Details: Details{
			Type:       "play",
			Playwright: "Elena Brook",
			Director:   "Sam Okafor",
			Cast: []CastMember{
				{Actor: "Mirren Vale", Role: "Ada"},
				{Actor: "Theo Brandt", Role: "The Cartographer"},
				{Actor: "June Park", Role: "Iris"},
			},
		},
		BasePriceCents: 5500,
		Showtimes: []Showtime{
			{time.Tuesday, 19, 0},
			{time.Thursday, 19, 0},
			{time.Saturday, 14, 0},
		},
	},
	{
		ID:      "twelve-clocks",
		Title:   "Twelve Clocks",
		Genre:   "play",
		Tagline: "Every hour, somebody lies.",
		Description: "A locked-room mystery told backwards: twelve scenes, twelve clocks, one " +
			"of them telling the truth. Monday's audience gets home arguing about it.",
		Details: Details{
			Type:       "play",
			Playwright: "Horace Fenn",
			Director:   "Lena Adeyemi",
			Cast: []CastMember{
				{Actor: "Casper Roan", Role: "The Horologist"},
				{Actor: "Bibi Santos", Role: "Inspector Vane"},
			},
		},
		BasePriceCents: 4800,
		Showtimes: []Showtime{
			{time.Monday, 19, 30},
		},
	},
	{
		ID:      "milo-frey",
		Title:   "Milo Frey: Grand Opening",
		Genre:   "comedy",
		Tagline: "New hour. Old grudges.",
		Description: "Milo Frey road-tested this hour in forty basements so he could ruin it " +
			"here, in front of people who paid. The Friday late show gets the unlisted bits.",
		Details: Details{
			Type:     "comedy",
			Comedian: "Milo Frey",
			Openers:  []string{"Rosa Quinn"},
		},
		BasePriceCents: 4200,
		Showtimes: []Showtime{
			{time.Friday, 22, 0},
			{time.Saturday, 18, 0},
		},
	},
	{
		ID:      "kaja-players",
		Title:   "Improv Night with The Kaja Players",
		Genre:   "comedy",
		Tagline: "Nothing is planned. Everything is your fault.",
		Description: "The house troupe builds an entire show out of audience suggestions. " +
			"Yes, they will use yours. No, you cannot take it back.",
		Details: Details{
			Type:     "comedy",
			Comedian: "The Kaja Players",
		},
		BasePriceCents: 2500,
		Showtimes: []Showtime{
			{time.Sunday, 19, 30},
		},
	},
	{
		ID:      "glass-mountain",
		Title:   "The Glass Mountain",
		Genre:   "opera",
		Tagline: "Climb it, or become part of the view.",
		Description: "Ilona Vasseur's modern fable, staged with a mirrored set that puts the " +
			"audience inside the mountain. Sung in Czech with English surtitles.",
		Details: Details{
			Type:      "opera",
			Composer:  "Ilona Vasseur",
			Language:  "Czech",
			Surtitles: boolPtr(true),
		},
		BasePriceCents: 9500,
		Showtimes: []Showtime{
			{time.Sunday, 15, 0},
		},
	},
}

func Events() []Event { return events }

func EventByID(id string) (Event, bool) {
	for _, e := range events {
		if e.ID == id {
			return e, true
		}
	}
	return Event{}, false
}

func performanceID(eventID string, t time.Time) string {
	return fmt.Sprintf("%s-%s-%02d%02d", eventID, t.Format("20060102"), t.Hour(), t.Minute())
}

func makePerformance(e Event, startsAt, now time.Time) Performance {
	status := StatusOnSale
	if startsAt.Before(now) {
		status = StatusPast
	}
	return Performance{
		ID:             performanceID(e.ID, startsAt),
		EventID:        e.ID,
		EventTitle:     e.Title,
		StartsAt:       startsAt,
		DoorsOpenAt:    startsAt.Add(-45 * time.Minute),
		BasePriceCents: e.BasePriceCents,
		Status:         status,
	}
}

// PerformancesFor lists an event's performances in the rolling window
// [today, today+6], including today's already-played shows (status "past").
func PerformancesFor(e Event, now time.Time) []Performance {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var out []Performance
	for day := 0; day < WindowDays; day++ {
		date := today.AddDate(0, 0, day)
		for _, s := range e.Showtimes {
			if date.Weekday() != s.Weekday {
				continue
			}
			startsAt := date.Add(time.Duration(s.Hour)*time.Hour + time.Duration(s.Minute)*time.Minute)
			out = append(out, makePerformance(e, startsAt, now))
		}
	}
	return out
}

// NextPerformanceID returns the soonest upcoming performance's id, or nil.
func NextPerformanceID(e Event, now time.Time) *string {
	for _, p := range PerformancesFor(e, now) {
		if p.Status == StatusOnSale {
			id := p.ID
			return &id
		}
	}
	return nil
}

// Sentinel results for PerformanceByID.
type LookupResult int

const (
	LookupFound LookupResult = iota
	LookupNotFound
	// LookupGone means the id is a real showtime that has already happened.
	LookupGone
)

// PerformanceByID reconstructs a performance from its id, which encodes the
// event and the start time ("neon-meridian-20260710-1930"). The id must
// match the event's weekly grid; future dates beyond the window are not on
// sale yet and report as not found.
func PerformanceByID(id string, now time.Time) (Performance, LookupResult) {
	now = now.UTC()
	parts := strings.Split(id, "-")
	if len(parts) < 3 {
		return Performance{}, LookupNotFound
	}
	datePart := parts[len(parts)-2]
	timePart := parts[len(parts)-1]
	eventID := strings.Join(parts[:len(parts)-2], "-")

	e, ok := EventByID(eventID)
	if !ok {
		return Performance{}, LookupNotFound
	}
	date, err := time.ParseInLocation("20060102", datePart, time.UTC)
	if err != nil || len(timePart) != 4 {
		return Performance{}, LookupNotFound
	}
	hour, err1 := strconv.Atoi(timePart[:2])
	minute, err2 := strconv.Atoi(timePart[2:])
	if err1 != nil || err2 != nil {
		return Performance{}, LookupNotFound
	}

	matches := false
	for _, s := range e.Showtimes {
		if date.Weekday() == s.Weekday && hour == s.Hour && minute == s.Minute {
			matches = true
			break
		}
	}
	if !matches {
		return Performance{}, LookupNotFound
	}

	startsAt := date.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if startsAt.Before(now) {
		return Performance{}, LookupGone
	}
	if !startsAt.Before(today.AddDate(0, 0, WindowDays)) {
		return Performance{}, LookupNotFound
	}
	return makePerformance(e, startsAt, now), LookupFound
}
