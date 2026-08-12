// Package tools is the kitchen's side of the MCP protocol: three tools, each
// answering one question a person actually asks while holding a bowl.
//
// They are judgments rather than lookups, which is why this is an MCP server
// and not a fourth REST endpoint — "what do I use instead" has an opinion in it.
package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/kaja-tools/website/v2/internal/bakebook"
	"github.com/kaja-tools/website/v2/internal/kitchen"
	"github.com/kaja-tools/website/v2/internal/mcp"
)

// All returns the kitchen's tools in the order they should be read: plan a
// batch, fix what you are missing, then decide what to drink with it.
func All(book *bakebook.Client) []mcp.Tool {
	return []mcp.Tool{scale(book), substitute(), pair(book)}
}

func scale(book *bakebook.Client) mcp.Tool {
	return mcp.Tool{
		Name:  "scale",
		Title: "Scale a recipe",
		Description: "Work out what a batch actually needs: every ingredient by weight for the " +
			"number of cookies you want, how many trays that is, and which of it is not on the " +
			"shelf this morning. Cookie ids come from https://bakebook.kaja.tools/cookies.",
		InputSchema: mcp.Object{
			{"type", "object"},
			{"properties", mcp.Object{
				{"cookieId", mcp.Object{
					{"type", "string"},
					{"description", "An id from the bakebook, e.g. \"brown-butter\"."},
				}},
				{"cookies", mcp.Object{
					{"type", "integer"},
					{"description", "How many cookies you want to end up with."},
					{"minimum", 1},
				}},
			}},
			{"required", []string{"cookieId", "cookies"}},
		},
		OutputSchema: mcp.Object{
			{"type", "object"},
			{"properties", mcp.Object{
				{"cookieId", mcp.Object{{"type", "string"}}},
				{"cookie", mcp.Object{{"type", "string"}}},
				{"cookies", mcp.Object{{"type", "integer"}}},
				{"trays", mcp.Object{{"type", "integer"}}},
				{"doughGrams", mcp.Object{{"type", "integer"}}},
				{"lines", mcp.Object{
					{"type", "array"},
					{"description", "The ingredients, heaviest first."},
					{"items", mcp.Object{
						{"type", "object"},
						{"properties", mcp.Object{
							{"ingredient", mcp.Object{{"type", "string"}}},
							{"grams", mcp.Object{{"type", "integer"}}},
							{"inStock", mcp.Object{{"type", "boolean"}}},
						}},
						{"required", []string{"ingredient", "grams", "inStock"}},
					}},
				}},
				{"missing", mcp.Object{
					{"type", "array"},
					{"description", "Anything on the list the shelf is out of."},
					{"items", mcp.Object{{"type", "string"}}},
				}},
				{"note", mcp.Object{{"type", "string"}}},
			}},
			{"required", []string{"cookieId", "cookie", "cookies", "trays", "doughGrams", "lines", "missing", "note"}},
		},
		Call: func(arguments json.RawMessage) (*mcp.Result, error) {
			var args struct {
				CookieID string `json:"cookieId"`
				Cookies  int    `json:"cookies"`
			}
			if err := json.Unmarshal(arguments, &args); err != nil {
				return refuse("I could not read that: %v", err), nil
			}
			if args.Cookies < 1 {
				return refuse("Ask for at least one cookie."), nil
			}

			cookie, err := book.Cookie(args.CookieID)
			if err != nil {
				return unknownCookie(book, args.CookieID, err)
			}
			recipe, ok := kitchen.Recipe(cookie.ID)
			if !ok {
				return refuse("We bake %s but the recipe is in somebody's head rather than the book.", cookie.Name), nil
			}

			dough := args.Cookies * cookie.DoughGramsPerCookie
			lines := make([]mcp.Object, 0, len(recipe))
			var missing []string
			for _, ingredient := range recipe {
				inStock := kitchen.InStock(ingredient.Name)
				if !inStock {
					missing = append(missing, ingredient.Name)
				}
				lines = append(lines, mcp.Object{
					{"ingredient", ingredient.Name},
					{"grams", int(math.Round(float64(dough) * ingredient.Percent / 100))},
					{"inStock", inStock},
				})
			}
			if missing == nil {
				missing = []string{}
			}

			trays := int(math.Ceil(float64(args.Cookies) / float64(cookie.PerTray)))
			note := "Everything on this list is on the shelf."
			if len(missing) > 0 {
				note = fmt.Sprintf("The shelf is out of %s — ask substitute what to use instead.",
					strings.Join(missing, " and "))
			}

			return &mcp.Result{
				Text: fmt.Sprintf("%d %s is %dg of dough over %d tray(s). %s",
					args.Cookies, cookie.Name, dough, trays, note),
				Structured: mcp.Object{
					{"cookieId", cookie.ID},
					{"cookie", cookie.Name},
					{"cookies", args.Cookies},
					{"trays", trays},
					{"doughGrams", dough},
					{"lines", lines},
					{"missing", missing},
					{"note", note},
				},
			}, nil
		},
	}
}

func substitute() mcp.Tool {
	return mcp.Tool{
		Name:  "substitute",
		Title: "Substitute an ingredient",
		Description: "You have run out of something, or somebody at the table cannot eat it. " +
			"Says what to use instead, in what ratio, and what it costs you — there is always " +
			"a cost, and pretending otherwise is how you end up with a flat cookie.",
		InputSchema: mcp.Object{
			{"type", "object"},
			{"properties", mcp.Object{
				{"ingredient", mcp.Object{
					{"type", "string"},
					{"description", "What you are missing, e.g. \"cream of tartar\"."},
				}},
				{"why", mcp.Object{
					{"type", "string"},
					{"description", "Optional. A reason mentioning vegan, dairy or eggs gets a different answer where there is one."},
				}},
			}},
			{"required", []string{"ingredient"}},
		},
		OutputSchema: mcp.Object{
			{"type", "object"},
			{"properties", mcp.Object{
				{"ingredient", mcp.Object{{"type", "string"}}},
				{"use", mcp.Object{{"type", "string"}}},
				{"ratio", mcp.Object{{"type", "string"}}},
				{"note", mcp.Object{{"type", "string"}, {"description", "What the swap changes about the cookie."}}},
				{"inStock", mcp.Object{{"type", "boolean"}, {"description", "Whether we had it after all."}}},
			}},
			{"required", []string{"ingredient", "use", "ratio", "note", "inStock"}},
		},
		Call: func(arguments json.RawMessage) (*mcp.Result, error) {
			var args struct {
				Ingredient string `json:"ingredient"`
				Why        string `json:"why"`
			}
			if err := json.Unmarshal(arguments, &args); err != nil {
				return refuse("I could not read that: %v", err), nil
			}
			if strings.TrimSpace(args.Ingredient) == "" {
				return refuse("Name something and I will tell you what to use instead."), nil
			}

			swap, ok := kitchen.Substitute(args.Ingredient, args.Why)
			if !ok {
				return refuse("We don't stock %s, so there is nothing to swap it for. The book uses: %s.",
					args.Ingredient, strings.Join(kitchen.Ingredients(), ", ")), nil
			}

			return &mcp.Result{
				Text: fmt.Sprintf("Instead of %s: %s (%s). %s", args.Ingredient, swap.Use, swap.Ratio, swap.Note),
				Structured: mcp.Object{
					{"ingredient", args.Ingredient},
					{"use", swap.Use},
					{"ratio", swap.Ratio},
					{"note", swap.Note},
					{"inStock", kitchen.InStock(args.Ingredient)},
				},
			}, nil
		},
	}
}

func pair(book *bakebook.Client) mcp.Tool {
	return mcp.Tool{
		Name:        "pair",
		Title:       "Pair a cookie with a drink",
		Description: "What to drink with it, and why that and not something else. One answer, not a list.",
		InputSchema: mcp.Object{
			{"type", "object"},
			{"properties", mcp.Object{
				{"cookieId", mcp.Object{
					{"type", "string"},
					{"description", "An id from the bakebook, e.g. \"shortbread\"."},
				}},
			}},
			{"required", []string{"cookieId"}},
		},
		OutputSchema: mcp.Object{
			{"type", "object"},
			{"properties", mcp.Object{
				{"cookieId", mcp.Object{{"type", "string"}}},
				{"cookie", mcp.Object{{"type", "string"}}},
				{"drink", mcp.Object{{"type", "string"}}},
				{"why", mcp.Object{{"type", "string"}}},
				{"alsoTry", mcp.Object{{"type", "string"}}},
			}},
			{"required", []string{"cookieId", "cookie", "drink", "why", "alsoTry"}},
		},
		Call: func(arguments json.RawMessage) (*mcp.Result, error) {
			var args struct {
				CookieID string `json:"cookieId"`
			}
			if err := json.Unmarshal(arguments, &args); err != nil {
				return refuse("I could not read that: %v", err), nil
			}

			cookie, err := book.Cookie(args.CookieID)
			if err != nil {
				return unknownCookie(book, args.CookieID, err)
			}
			pairing, ok := kitchen.PairingFor(cookie.ID)
			if !ok {
				return refuse("Nobody has decided what goes with %s yet.", cookie.Name), nil
			}

			return &mcp.Result{
				Text: fmt.Sprintf("%s with %s. %s", cookie.Name, pairing.Drink, pairing.Why),
				Structured: mcp.Object{
					{"cookieId", cookie.ID},
					{"cookie", cookie.Name},
					{"drink", pairing.Drink},
					{"why", pairing.Why},
					{"alsoTry", pairing.AlsoTry},
				},
			}, nil
		},
	}
}

// unknownCookie separates the two failures that look alike from the outside: a
// question about something we don't bake is an answer, and a book we cannot
// reach is a broken call.
func unknownCookie(book *bakebook.Client, id string, err error) (*mcp.Result, error) {
	if !errors.Is(err, bakebook.ErrUnknownCookie) {
		return nil, err
	}
	ids := book.IDs()
	if len(ids) == 0 {
		return refuse("We don't bake %q.", id), nil
	}
	return refuse("We don't bake %q. The book has: %s.", id, strings.Join(ids, ", ")), nil
}

// refuse is the tool answering "no" — a result, not an error. The call worked;
// the kitchen just has nothing useful to say.
func refuse(format string, args ...any) *mcp.Result {
	return &mcp.Result{Text: fmt.Sprintf(format, args...), IsError: true}
}
