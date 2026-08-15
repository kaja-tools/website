package concierge

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kaja-tools/website/v2/internal/mcp"
)

const instructions = `You are talking to the concierge of the Theatre, a chain of repertory cinemas.

Ask suggest_film what somebody feels like in their own words — and which city
they are in, if they said — and it answers with one screening and the case for
it. The concierge has already joined the catalog, the chain and the schedule,
so a screening it names knows its title, its house and its time. Pass that
screening's showId to best_seats with the size of the party and it reads the
live seat map and picks a block of seats together. Buying them is not the
concierge's to do: pass the showId and the seat ids to the seating service's
BookSeats.`

// Tools is the concierge's whole surface. Three verbs, in the order somebody
// actually uses them: what should we see, where should we sit, and what do I
// tell the others.
func Tools(house *House) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "suggest_film",
			Title:       "Suggest a film",
			Description: "Pick one screening from this week's programme for somebody who has described what they feel like in their own words, and optionally which city they are in. Answers with the case for it and two runners-up.",
			Annotations: mcp.ReadOnly(),
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"mood": {
						"type": "string",
						"description": "What they feel like, in their own words: \"something loud\", \"quiet and sad\", \"funny, and we have to be home by ten\"."
					},
					"party": {
						"type": "integer",
						"description": "How many are coming. Only ever mentioned back to you.",
						"minimum": 1,
						"maximum": 6
					},
					"maxMinutes": {
						"type": "integer",
						"description": "The longest running time they will sit through. Leave it out if they did not say."
					},
					"city": {
						"type": "string",
						"description": "Which town they are in, e.g. \"Chicago\". Leave it out to search the whole chain."
					}
				},
				"required": ["mood"]
			}`),
			OutputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"showId": { "type": "string", "description": "Pass this to best_seats and to the seating service." },
					"title": { "type": "string" },
					"director": { "type": "string" },
					"year": { "type": "integer" },
					"runtimeMinutes": { "type": "integer" },
					"genre": { "type": "string" },
					"theater": { "type": "string", "description": "The house it is playing at." },
					"city": { "type": "string" },
					"startsAt": { "type": "string", "description": "The next screening, in the house's own local time, RFC 3339." },
					"priceCents": { "type": "integer", "description": "What an orchestra seat costs at that house." },
					"why": { "type": "string", "description": "The case for it, in the concierge's own words." },
					"runnersUp": {
						"type": "array",
						"description": "The next two the concierge would have said, in order.",
						"items": {
							"type": "object",
							"properties": {
								"showId": { "type": "string" },
								"title": { "type": "string" },
								"theater": { "type": "string" },
								"why": { "type": "string" }
							}
						}
					}
				},
				"required": ["showId", "title", "why"]
			}`),
			Call: func(arguments json.RawMessage) (mcp.Result, error) {
				var input struct {
					Mood       string `json:"mood"`
					Party      int    `json:"party"`
					MaxMinutes int    `json:"maxMinutes"`
					City       string `json:"city"`
				}
				if err := json.Unmarshal(arguments, &input); err != nil {
					return mcp.Result{}, mcp.Failf("I could not read that request: %v", err)
				}

				programme, err := house.Programme()
				if err != nil {
					return mcp.Result{}, err
				}
				screenings := InCity(programme, input.City)
				if len(screenings) == 0 {
					return mcp.Result{}, mcp.Failf(
						"the chain has no house in %s — we are in %s.",
						input.City, joinWithAnd(cities(programme)))
				}

				matches := Suggest(screenings, input.Mood, input.MaxMinutes)
				if len(matches) == 0 {
					return mcp.Result{}, mcp.Failf(
						"nothing there runs in under %d minutes — the shortest is %d.",
						input.MaxMinutes, shortest(screenings))
				}

				pick := matches[0]
				runnersUp := []map[string]any{}
				for _, match := range matches[1:min(3, len(matches))] {
					runnersUp = append(runnersUp, map[string]any{
						"showId":  match.Screening.ID,
						"title":   match.Screening.Movie.Title,
						"theater": match.Screening.Theater.Where(),
						"why":     match.Why,
					})
				}

				return mcp.Result{
					Text: pick.Why,
					Structured: map[string]any{
						"showId":         pick.Screening.ID,
						"title":          pick.Screening.Movie.Title,
						"director":       pick.Screening.Movie.Director,
						"year":           pick.Screening.Movie.Year,
						"runtimeMinutes": pick.Screening.Movie.RuntimeMinutes,
						"genre":          pick.Screening.Movie.Genre,
						"theater":        pick.Screening.Theater.Where(),
						"city":           pick.Screening.Theater.City,
						"startsAt":       pick.Screening.StartsAt.Format(time.RFC3339),
						"priceCents":     pick.Screening.PriceCents,
						"why":            pick.Why,
						"runnersUp":      runnersUp,
					},
				}, nil
			},
		},
		{
			Name:        "best_seats",
			Title:       "Pick the best seats",
			Description: "Read the live seat map for a screening and pick the best block of seats still free side by side. Reserves nothing — the seat ids come back for you to buy with the seating service's BookSeats.",
			Annotations: mcp.ReadOnly(),
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"showId": {
						"type": "string",
						"description": "A screening id from the schedule, e.g. \"dune-part-two@the-lantern\".",
						"examples": ["dune-part-two@the-lantern"]
					},
					"party": {
						"type": "integer",
						"description": "How many seats together. At most 6, which is what the house sells at a time.",
						"minimum": 1,
						"maximum": 6,
						"default": 2
					}
				},
				"required": ["showId"]
			}`),
			OutputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"seatIds": {
						"type": "array",
						"description": "Pass these to the seating service's BookSeats, with the same showId.",
						"items": { "type": "string" }
					},
					"section": { "type": "string" },
					"row": { "type": "string" },
					"totalCents": { "type": "integer", "description": "What the whole block costs." },
					"why": { "type": "string", "description": "Why these ones, in the concierge's own words." },
					"seatsFree": { "type": "integer", "description": "How many seats were free in the house when this was read." }
				},
				"required": ["seatIds", "why"]
			}`),
			Call: func(arguments json.RawMessage) (mcp.Result, error) {
				var input struct {
					ShowID string `json:"showId"`
					Party  int    `json:"party"`
				}
				if err := json.Unmarshal(arguments, &input); err != nil {
					return mcp.Result{}, mcp.Failf("I could not read that request: %v", err)
				}
				if input.Party <= 0 {
					input.Party = 2
				}
				if input.Party > 6 {
					return mcp.Result{}, mcp.Failf("the house sells at most 6 seats at a time; you asked for %d.", input.Party)
				}

				screening, err := house.Screening(input.ShowID)
				if err != nil {
					return mcp.Result{}, mcp.Failf("%v", err)
				}
				seatMap, err := house.SeatMap(input.ShowID)
				if err != nil {
					return mcp.Result{}, err
				}

				seats, ok := PickSeats(seatMap, input.Party)
				if !ok {
					return mcp.Result{}, mcp.Failf(
						"there is no run of %d seats together left for %s at %s — %d seats are free, but scattered.",
						input.Party, screening.Movie.Title, screening.Theater.Where(), seatMap.Available)
				}

				return mcp.Result{
					Text: fmt.Sprintf("%s for %s at %s: %s. %s",
						strings.Join(seats.IDs, ", "), screening.Movie.Title,
						screening.Theater.Where(), money(seats.TotalCents), seats.Why),
					Structured: map[string]any{
						"seatIds":    seats.IDs,
						"section":    seats.Section,
						"row":        seats.Row,
						"totalCents": seats.TotalCents,
						"why":        seats.Why,
						"seatsFree":  seatMap.Available,
					},
				}, nil
			},
		},
		{
			Name:        "write_confirmation",
			Title:       "Write the confirmation",
			Description: "Write the note that goes out after a booking — the film, the time, the seats, and how long to leave for the trailers.",
			Annotations: mcp.ReadOnly(),
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"showId": { "type": "string", "description": "The screening that was booked, as the schedule published it." },
					"seatIds": {
						"type": "array",
						"description": "The seats that were bought, as the seating service reported them.",
						"items": { "type": "string" }
					},
					"name": { "type": "string", "description": "Who to address it to. Left out, it is addressed to nobody in particular." }
				},
				"required": ["showId", "seatIds"]
			}`),
			OutputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"note": { "type": "string", "description": "The confirmation, ready to send." },
					"arriveBy": { "type": "string", "description": "When to be in the building, in the house's own local time, RFC 3339." }
				},
				"required": ["note"]
			}`),
			Call: func(arguments json.RawMessage) (mcp.Result, error) {
				var input struct {
					ShowID  string   `json:"showId"`
					SeatIDs []string `json:"seatIds"`
					Name    string   `json:"name"`
				}
				if err := json.Unmarshal(arguments, &input); err != nil {
					return mcp.Result{}, mcp.Failf("I could not read that request: %v", err)
				}
				if len(input.SeatIDs) == 0 {
					return mcp.Result{}, mcp.Failf("a confirmation needs the seats that were bought.")
				}

				screening, err := house.Screening(input.ShowID)
				if err != nil {
					return mcp.Result{}, mcp.Failf("%v", err)
				}

				// Doors are fifteen minutes before, and the trailers run ten
				// past that. Arriving on the hour is arriving on time.
				arriveBy := screening.StartsAt.Add(-15 * time.Minute)
				greeting := "You are booked in"
				if input.Name != "" {
					greeting = input.Name + ", you are booked in"
				}

				note := fmt.Sprintf(
					"%s for %s (%s, %d), %s at %s — %s %s.\n\n"+
						"The house is %s, %s, %s. Doors are at %s and the trailers run about ten "+
						"minutes, so there is no need to sprint. The film runs %d minutes and is in %s."+
						"\n\n— the Theatre",
					greeting,
					screening.Movie.Title, screening.Movie.Director, screening.Movie.Year,
					screening.StartsAt.Format("Monday 2 January, 15:04"),
					screening.Theater.Name,
					plural(len(input.SeatIDs), "seat", "seats"), strings.Join(input.SeatIDs, ", "),
					screening.Theater.Address, screening.Theater.City, screening.Theater.State,
					arriveBy.Format("15:04"),
					screening.Movie.RuntimeMinutes, screening.Movie.Language)

				return mcp.Result{
					Text: note,
					Structured: map[string]any{
						"note":     note,
						"arriveBy": arriveBy.Format(time.RFC3339),
					},
				}, nil
			},
		},
	}
}

// Instructions is what the server tells a client it is for.
func Instructions() string { return instructions }

func shortest(screenings []Screening) int {
	shortest := 0
	for _, s := range screenings {
		if shortest == 0 || s.Movie.RuntimeMinutes < shortest {
			shortest = s.Movie.RuntimeMinutes
		}
	}
	return shortest
}

// cities is every town the chain is in, in alphabetical order, which is what
// a request naming somewhere it is not gets told instead.
func cities(screenings []Screening) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range screenings {
		if !seen[s.Theater.City] {
			seen[s.Theater.City] = true
			out = append(out, s.Theater.City)
		}
	}
	sort.Strings(out)
	return out
}

func money(cents int32) string {
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
