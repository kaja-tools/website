// Package catalog holds The Kaja Theatre's programme — a repertory cinema in
// a restored movie palace, so the films are real ones and the house has a
// balcony. Every film screens once a week in the same slot, so its next
// showtime is derived from the current time: no storage, and the demo can
// never run out of screenings.
//
// A screening is the unit the whole demo is built on: its id is what you pass
// to the seating service. There is no separate "film" and "screening".
package catalog

import "time"

type Show struct {
	ID             string
	Title          string
	Director       string
	Year           int
	RuntimeMinutes int
	Genre          string
	Language       string
	Synopsis       string
	BasePriceCents int
	// moods the concierge matches a request against. Not published: the
	// programme is the film, and taste is the concierge's job.
	Moods []string
	// The weekly slot. Times are UTC.
	Weekday time.Weekday
	Hour    int
	Minute  int
}

var shows = []Show{
	{
		ID:             "dune-part-two",
		Title:          "Dune: Part Two",
		Director:       "Denis Villeneuve",
		Year:           2024,
		RuntimeMinutes: 165,
		Genre:          "sci-fi",
		Language:       "English",
		Synopsis:       "Paul Atreides joins the Fremen and rides the desert into a war he can see the end of.",
		BasePriceCents: 1800,
		Moods:          []string{"loud", "epic", "spectacle", "big screen", "sci-fi"},
		Weekday:        time.Friday,
		Hour:           19,
		Minute:         30,
	},
	{
		ID:             "parasite",
		Title:          "Parasite",
		Director:       "Bong Joon Ho",
		Year:           2019,
		RuntimeMinutes: 132,
		Genre:          "thriller",
		Language:       "Korean",
		Synopsis:       "One family talks its way into another family's house, one job at a time.",
		BasePriceCents: 1500,
		Moods:          []string{"tense", "clever", "dark", "modern classic", "thriller"},
		Weekday:        time.Thursday,
		Hour:           20,
		Minute:         0,
	},
	{
		ID:             "spirited-away",
		Title:          "Spirited Away",
		Director:       "Hayao Miyazaki",
		Year:           2001,
		RuntimeMinutes: 124,
		Genre:          "animation",
		Language:       "Japanese",
		Synopsis:       "A ten-year-old takes a job in a bathhouse for spirits to win her parents back.",
		BasePriceCents: 1400,
		Moods:          []string{"gentle", "family", "beautiful", "kids", "animation"},
		Weekday:        time.Sunday,
		Hour:           14,
		Minute:         0,
	},
	{
		ID:             "grand-budapest-hotel",
		Title:          "The Grand Budapest Hotel",
		Director:       "Wes Anderson",
		Year:           2014,
		RuntimeMinutes: 100,
		Genre:          "comedy",
		Language:       "English",
		Synopsis:       "A concierge and his lobby boy inherit a painting and the trouble attached to it.",
		BasePriceCents: 1400,
		Moods:          []string{"funny", "light", "charming", "short", "comedy"},
		Weekday:        time.Wednesday,
		Hour:           19,
		Minute:         0,
	},
	{
		ID:             "blade-runner",
		Title:          "Blade Runner",
		Director:       "Ridley Scott",
		Year:           1982,
		RuntimeMinutes: 117,
		Genre:          "sci-fi",
		Language:       "English",
		Synopsis:       "A detective hunts four fugitives who are harder to tell from people than the job allows.",
		BasePriceCents: 1300,
		Moods:          []string{"moody", "classic", "rainy", "sci-fi", "late"},
		Weekday:        time.Friday,
		Hour:           22,
		Minute:         30,
	},
	{
		ID:             "in-the-mood-for-love",
		Title:          "In the Mood for Love",
		Director:       "Wong Kar-wai",
		Year:           2000,
		RuntimeMinutes: 98,
		Genre:          "romance",
		Language:       "Cantonese",
		Synopsis:       "Two neighbours work out that their spouses are having an affair, and say almost nothing about it.",
		BasePriceCents: 1300,
		Moods:          []string{"romantic", "quiet", "sad", "date", "short"},
		Weekday:        time.Tuesday,
		Hour:           20,
		Minute:         30,
	},
	{
		ID:             "mad-max-fury-road",
		Title:          "Mad Max: Fury Road",
		Director:       "George Miller",
		Year:           2015,
		RuntimeMinutes: 120,
		Genre:          "action",
		Language:       "English",
		Synopsis:       "A war rig turns left across the wasteland and everything that can drive follows it.",
		BasePriceCents: 1600,
		Moods:          []string{"loud", "fast", "action", "spectacle", "big screen"},
		Weekday:        time.Saturday,
		Hour:           21,
		Minute:         0,
	},
	{
		ID:             "seven-samurai",
		Title:          "Seven Samurai",
		Director:       "Akira Kurosawa",
		Year:           1954,
		RuntimeMinutes: 207,
		Genre:          "drama",
		Language:       "Japanese",
		Synopsis:       "A village with nothing to pay with hires seven swordsmen anyway.",
		BasePriceCents: 1200,
		Moods:          []string{"classic", "long", "epic", "drama"},
		Weekday:        time.Sunday,
		Hour:           17,
		Minute:         0,
	},
	{
		ID:             "get-out",
		Title:          "Get Out",
		Director:       "Jordan Peele",
		Year:           2017,
		RuntimeMinutes: 104,
		Genre:          "horror",
		Language:       "English",
		Synopsis:       "A weekend meeting the parents goes wrong slowly, and then all at once.",
		BasePriceCents: 1400,
		Moods:          []string{"scary", "tense", "horror", "short", "late"},
		Weekday:        time.Thursday,
		Hour:           22,
		Minute:         30,
	},
	{
		ID:             "everything-everywhere",
		Title:          "Everything Everywhere All at Once",
		Director:       "Daniel Kwan and Daniel Scheinert",
		Year:           2022,
		RuntimeMinutes: 139,
		Genre:          "sci-fi",
		Language:       "English",
		Synopsis:       "A laundromat audit becomes a tour of every life its owner didn't live.",
		BasePriceCents: 1500,
		Moods:          []string{"funny", "weird", "loud", "sci-fi", "modern classic"},
		Weekday:        time.Saturday,
		Hour:           18,
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
