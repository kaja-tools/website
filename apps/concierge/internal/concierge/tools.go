package concierge

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kaja-tools/website/v2/internal/mcp"
)

const instructions = `You are talking to the concierge of The Kaja Theatre, a repertory cinema.

Ask suggest_film what somebody feels like in their own words and it answers with
one screening and the case for it. Pass that screening's showId to best_seats
with the size of the party and it reads the live seat map and picks a block of
seats together. Buying them is not the concierge's to do: pass the showId and
the seat ids to the seating service's BookSeats.`

// Tools is the concierge's whole surface. Three verbs, in the order somebody
// actually uses them: what should we see, where should we sit, and what do I
// tell the others.
func Tools(house *House) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "suggest_film",
			Title:       "Suggest a film",
			Description: "Pick one screening from this week's programme for somebody who has described what they feel like in their own words. Answers with the case for it and two runners-up.",
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
					"startsAt": { "type": "string", "description": "The next screening, RFC 3339." },
					"why": { "type": "string", "description": "The case for it, in the concierge's own words." },
					"runnersUp": {
						"type": "array",
						"description": "The next two the concierge would have said, in order.",
						"items": {
							"type": "object",
							"properties": {
								"showId": { "type": "string" },
								"title": { "type": "string" },
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
				}
				if err := json.Unmarshal(arguments, &input); err != nil {
					return mcp.Result{}, mcp.Failf("I could not read that request: %v", err)
				}

				shows, err := house.Programme()
				if err != nil {
					return mcp.Result{}, err
				}
				matches := Suggest(shows, input.Mood, input.MaxMinutes)
				if len(matches) == 0 {
					return mcp.Result{}, mcp.Failf(
						"nothing this week runs in under %d minutes — the shortest is %d.",
						input.MaxMinutes, shortest(shows))
				}

				pick := matches[0]
				runnersUp := []map[string]any{}
				for _, match := range matches[1:min(3, len(matches))] {
					runnersUp = append(runnersUp, map[string]any{
						"showId": match.Show.ID,
						"title":  match.Show.Title,
						"why":    match.Why,
					})
				}

				return mcp.Result{
					Text: pick.Why,
					Structured: map[string]any{
						"showId":         pick.Show.ID,
						"title":          pick.Show.Title,
						"director":       pick.Show.Director,
						"year":           pick.Show.Year,
						"runtimeMinutes": pick.Show.RuntimeMinutes,
						"genre":          pick.Show.Genre,
						"startsAt":       pick.Show.StartsAt.Format(time.RFC3339),
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
						"description": "A screening id from the programme, e.g. \"dune-part-two\".",
						"examples": ["dune-part-two"]
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

				show, err := house.Show(input.ShowID)
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
						"there is no run of %d seats together left for %s — %d seats are free, but scattered.",
						input.Party, show.Title, seatMap.Available)
				}

				return mcp.Result{
					Text: fmt.Sprintf("%s for %s: %s. %s",
						strings.Join(seats.IDs, ", "), show.Title, money(seats.TotalCents), seats.Why),
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
					"showId": { "type": "string", "description": "The screening that was booked." },
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
					"arriveBy": { "type": "string", "description": "When to be in the building, RFC 3339." }
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

				show, err := house.Show(input.ShowID)
				if err != nil {
					return mcp.Result{}, mcp.Failf("%v", err)
				}

				// Doors are fifteen minutes before, and the trailers run ten
				// past that. Arriving on the hour is arriving on time.
				arriveBy := show.StartsAt.Add(-15 * time.Minute)
				greeting := "You are booked in"
				if input.Name != "" {
					greeting = input.Name + ", you are booked in"
				}

				note := fmt.Sprintf(
					"%s for %s (%s, %d) at %s — %s %s.\n\n"+
						"Doors are at %s and the trailers run about ten minutes, so there is no need to sprint. "+
						"The film runs %d minutes and is in %s.\n\n— The Kaja Theatre",
					greeting,
					show.Title, show.Director, show.Year,
					show.StartsAt.Format("Monday 2 January, 15:04"),
					plural(len(input.SeatIDs), "seat", "seats"), strings.Join(input.SeatIDs, ", "),
					arriveBy.Format("15:04"),
					show.RuntimeMinutes, show.Language)

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

func shortest(shows []Show) int {
	shortest := 0
	for _, show := range shows {
		if shortest == 0 || show.RuntimeMinutes < shortest {
			shortest = show.RuntimeMinutes
		}
	}
	return shortest
}

func money(cents int32) string {
	return fmt.Sprintf("€%d.%02d", cents/100, cents%100)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
