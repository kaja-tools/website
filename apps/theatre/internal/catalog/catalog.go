// Package catalog holds The Kaja Theatre's repertoire. Every show runs once
// a week on the same night, so its next date is derived from the current
// time — no storage needed, and the demo can never run out of shows.
//
// A show is the unit the whole demo is built on: its id is what you pass to
// the seating service. There is no separate "event" and "performance".
package catalog

import "time"

type Show struct {
	ID             string
	Title          string
	Genre          string
	Tagline        string
	Description    string
	BasePriceCents int
	// The weekly slot. Times are UTC.
	Weekday time.Weekday
	Hour    int
	Minute  int
}

var shows = []Show{
	{
		ID:      "neon-meridian",
		Title:   "Neon Meridian: World Tour",
		Genre:   "concert",
		Tagline: "Synthwave for the end of the night.",
		Description: "After two sold-out continents, Neon Meridian brings the World Tour home. " +
			"Expect walls of analog synth, a light rig that needed its own truck, and the " +
			"encore everyone films instead of watching.",
		BasePriceCents: 8500,
		Weekday:        time.Friday,
		Hour:           19,
		Minute:         30,
	},
	{
		ID:      "vera-lune",
		Title:   "An Evening with Vera Lune",
		Genre:   "concert",
		Tagline: "One voice, one piano, no microphone.",
		Description: "Jazz's quietest superstar performs entirely unamplified — the room was " +
			"built for it, and Vera Lune knows exactly what to do with it.",
		BasePriceCents: 6500,
		Weekday:        time.Wednesday,
		Hour:           20,
		Minute:         0,
	},
	{
		ID:      "cartographers-daughter",
		Title:   "The Cartographer's Daughter",
		Genre:   "play",
		Tagline: "A map of everywhere she never went.",
		Description: "Elena Brook's award-winning play about a daughter who inherits ten " +
			"thousand hand-drawn maps and a father she never understood. Bring tissues.",
		BasePriceCents: 5500,
		Weekday:        time.Tuesday,
		Hour:           19,
		Minute:         0,
	},
	{
		ID:      "twelve-clocks",
		Title:   "Twelve Clocks",
		Genre:   "play",
		Tagline: "Every hour, somebody lies.",
		Description: "A locked-room mystery told backwards: twelve scenes, twelve clocks, one " +
			"of them telling the truth. Monday's audience gets home arguing about it.",
		BasePriceCents: 4800,
		Weekday:        time.Monday,
		Hour:           19,
		Minute:         30,
	},
	{
		ID:      "milo-frey",
		Title:   "Milo Frey: Grand Opening",
		Genre:   "comedy",
		Tagline: "New hour. Old grudges.",
		Description: "Milo Frey road-tested this hour in forty basements so he could ruin it " +
			"here, in front of people who paid. The late show gets the unlisted bits.",
		BasePriceCents: 4200,
		Weekday:        time.Friday,
		Hour:           22,
		Minute:         0,
	},
	{
		ID:      "kaja-players",
		Title:   "Improv Night with The Kaja Players",
		Genre:   "comedy",
		Tagline: "Nothing is planned. Everything is your fault.",
		Description: "The house troupe builds an entire show out of audience suggestions. " +
			"Yes, they will use yours. No, you cannot take it back.",
		BasePriceCents: 2500,
		Weekday:        time.Sunday,
		Hour:           19,
		Minute:         30,
	},
	{
		ID:      "glass-mountain",
		Title:   "The Glass Mountain",
		Genre:   "opera",
		Tagline: "Climb it, or become part of the view.",
		Description: "Ilona Vasseur's modern fable, staged with a mirrored set that puts the " +
			"audience inside the mountain. Sung in Czech with English surtitles.",
		BasePriceCents: 9500,
		Weekday:        time.Sunday,
		Hour:           15,
		Minute:         0,
	},
}

func Shows() []Show { return shows }

func ByID(id string) (Show, bool) {
	for _, s := range shows {
		if s.ID == id {
			return s, true
		}
	}
	return Show{}, false
}

// NextStart is the show's next curtain-up after now. A weekly show always
// has one, so every show in the catalog is always on sale.
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
