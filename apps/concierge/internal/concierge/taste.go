package concierge

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/kaja-tools/website/v2/internal/api"
)

// The concierge's taste, and the whole of it. Every signal is read off what
// the catalog already publishes — genre, running time, year, language — so a
// film added to it tomorrow is understood the same day. There is no table of
// opinions keyed by film id here: that would go stale the first time the
// catalog changed, and a concierge who has only memorised this week is not
// one.
type signal struct {
	// words in the request that fire this signal.
	words []string
	// matches reports whether a screening satisfies it.
	matches func(Screening) bool
	// because is the half-sentence this signal contributes to the answer.
	because string
}

var signals = []signal{
	{
		words:   []string{"loud", "big", "spectacle", "epic", "blockbuster", "action", "fast", "thrilling", "exciting"},
		matches: genreIn("action", "sci-fi"),
		because: "it is the loudest thing on this week",
	},
	{
		words:   []string{"funny", "laugh", "comedy", "light", "cheerful", "fun", "silly"},
		matches: genreIn("comedy"),
		because: "it is genuinely funny",
	},
	{
		words:   []string{"scary", "horror", "creepy", "frightening", "spooky"},
		matches: genreIn("horror"),
		because: "it will frighten you",
	},
	{
		words:   []string{"tense", "thriller", "clever", "twist", "smart", "gripping"},
		matches: genreIn("thriller", "horror"),
		because: "it does not let go",
	},
	{
		words:   []string{"sad", "cry", "melancholy", "moving", "heartbreaking"},
		matches: genreIn("romance", "drama"),
		because: "it will get to you",
	},
	{
		// Quiet is about scale as much as genre: a three-hour battle epic is
		// many things, and quiet is not one of them.
		words:   []string{"quiet", "slow", "gentle", "calm", "intimate", "small"},
		matches: func(s Screening) bool { return genreIn("romance", "drama")(s) && s.Movie.RuntimeMinutes <= 140 },
		because: "it is a small, quiet one",
	},
	{
		words:   []string{"romantic", "romance", "love", "date", "anniversary"},
		matches: genreIn("romance"),
		because: "it is the one to bring somebody to",
	},
	{
		words:   []string{"kids", "child", "children", "family", "animation", "animated", "cartoon"},
		matches: genreIn("animation"),
		because: "it works for every age in the room",
	},
	{
		words:   []string{"sci-fi", "scifi", "science fiction", "future", "space"},
		matches: genreIn("sci-fi"),
		because: "it is science fiction",
	},
	{
		words:   []string{"classic", "old", "vintage", "black and white", "restored"},
		matches: func(s Screening) bool { return s.Movie.Year < 1990 },
		because: "it is a proper old print",
	},
	{
		words:   []string{"new", "recent", "modern"},
		matches: func(s Screening) bool { return s.Movie.Year >= 2015 },
		because: "it is recent",
	},
	{
		words:   []string{"short", "quick", "early", "brief", "school night"},
		matches: func(s Screening) bool { return s.Movie.RuntimeMinutes <= 110 },
		because: "it is done inside two hours",
	},
	{
		words:   []string{"long", "all evening", "marathon"},
		matches: func(s Screening) bool { return s.Movie.RuntimeMinutes >= 150 },
		because: "it takes the whole evening, on purpose",
	},
	{
		words:   []string{"subtitles", "subtitled", "foreign", "world cinema"},
		matches: func(s Screening) bool { return s.Movie.Language != "English" },
		because: "it is subtitled",
	},
	{
		words:   []string{"english", "no subtitles", "no reading"},
		matches: func(s Screening) bool { return s.Movie.Language == "English" },
		because: "it is in English",
	},
}

// Match is one screening the concierge would put forward, and why.
type Match struct {
	Screening Screening
	Why       string
	score     int
}

// Suggest ranks the programme against a request in somebody's own words. It
// never returns nothing: a request that matches no signal at all is answered
// with what is on soonest, which is what a concierge with no information to go
// on would say.
func Suggest(screenings []Screening, mood string, maxMinutes int) []Match {
	want := strings.ToLower(mood)

	matches := make([]Match, 0, len(screenings))
	for _, screening := range screenings {
		if maxMinutes > 0 && screening.Movie.RuntimeMinutes > maxMinutes {
			continue
		}

		var reasons []string
		score := 0
		for _, sig := range signals {
			if !mentions(want, sig.words) {
				continue
			}
			if sig.matches(screening) {
				score++
				reasons = append(reasons, sig.because)
			}
		}
		matches = append(matches, Match{Screening: screening, Why: explain(screening, reasons), score: score})
	}

	// Soonest first among equals: two films that answer the request equally
	// well are told apart by which one you can actually go to tonight.
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].Screening.StartsAt.Before(matches[j].Screening.StartsAt)
	})
	return matches
}

// InCity narrows the programme to one town. The city is the schedule's to
// know, not the concierge's — it is on the house the screening is at — which
// is why this reads off the join rather than asking the Theatre service again.
func InCity(screenings []Screening, city string) []Screening {
	want := strings.ToLower(strings.TrimSpace(city))
	if want == "" {
		return screenings
	}
	var out []Screening
	for _, s := range screenings {
		if strings.ToLower(s.Theater.City) == want {
			out = append(out, s)
		}
	}
	return out
}

// explain writes the sentence the concierge says out loud. With nothing
// matched it falls back to describing the film, which is honest: it is a
// suggestion, not a deduction. Where it is on is always in it: the same film
// is playing in four towns this week, and only one of them is yours.
func explain(s Screening, reasons []string) string {
	subject := fmt.Sprintf("%s (%s, %d) at %s",
		s.Movie.Title, s.Movie.Director, s.Movie.Year, s.Theater.Where())
	if len(reasons) == 0 {
		return fmt.Sprintf("%s — %s, and it is the next thing starting.",
			subject, strings.TrimRight(s.Movie.Synopsis, "."))
	}
	if len(reasons) > 2 {
		reasons = reasons[:2]
	}
	return fmt.Sprintf("%s — %s.", subject, joinWithAnd(reasons))
}

func genreIn(genres ...string) func(Screening) bool {
	return func(s Screening) bool {
		for _, genre := range genres {
			if s.Movie.Genre == genre {
				return true
			}
		}
		return false
	}
}

func mentions(text string, words []string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func joinWithAnd(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

// Seats is a block of seats together, and the case for them.
type Seats struct {
	IDs        []string
	Section    string
	Row        string
	TotalCents int32
	Why        string
}

// PickSeats finds the best run of `party` seats side by side in the live
// house. Best means what an usher means by it: the orchestra before the
// balcony, the middle of the house before the front row, and the middle of the
// row before either end.
func PickSeats(seatMap *api.SeatMap, party int) (Seats, bool) {
	var best Seats
	bestScore := math.Inf(-1)
	found := false

	for _, section := range seatMap.Sections {
		name := strings.TrimPrefix(section.Section.String(), "SECTION_")
		for index, row := range section.Rows {
			// The sweet spot is a little past halfway back: close enough to
			// fill your vision, far enough not to have to look up.
			depth := rowDepth(index, len(section.Rows))

			for _, run := range runsOf(row, party) {
				score := -3 * math.Abs(depth-0.6)
				score -= 2 * offCentre(run, row)
				if section.Section != api.Section_SECTION_ORCHESTRA {
					score -= 1.5
				}
				if score <= bestScore {
					continue
				}

				var ids []string
				var total int32
				for _, seat := range run {
					ids = append(ids, seat.Id)
					total += seat.PriceCents
				}
				bestScore, found = score, true
				best = Seats{
					IDs:        ids,
					Section:    name,
					Row:        row.Letter,
					TotalCents: total,
					Why:        seatReason(name, depth, offCentre(run, row)),
				}
			}
		}
	}
	return best, found
}

// rowDepth is how far back a row sits in its section, 0 at the front and 1 at
// the back. A section of one row is all of both, so it counts as the middle.
func rowDepth(index, rows int) float64 {
	if rows <= 1 {
		return 0.5
	}
	return float64(index) / float64(rows-1)
}

// offCentre is how far a block of seats sits from the middle of its row, 0 in
// the middle and 1 against a wall.
func offCentre(run []*api.Seat, row *api.Row) float64 {
	if len(row.Seats) == 0 {
		return 1
	}
	middle := float64(len(row.Seats)+1) / 2
	blockMiddle := (float64(run[0].Number) + float64(run[len(run)-1].Number)) / 2
	return math.Abs(blockMiddle-middle) / middle
}

func seatReason(section string, depth, off float64) string {
	where := "halfway back"
	switch {
	case depth < 0.34:
		where = "well forward"
	case depth > 0.75:
		where = "at the back"
	}
	across := "dead centre"
	switch {
	case off > 0.55:
		across = "over to one side"
	case off > 0.2:
		across = "a little off centre"
	}
	return fmt.Sprintf("%s in the %s, %s — the best block still free together.",
		strings.ToUpper(where[:1])+where[1:], strings.ToLower(section), across)
}

// runsOf returns every run of n available seats side by side in a row.
func runsOf(row *api.Row, n int) [][]*api.Seat {
	var runs [][]*api.Seat
	var run []*api.Seat
	for _, seat := range row.Seats {
		if seat.Status != api.SeatStatus_SEAT_STATUS_AVAILABLE {
			run = nil
			continue
		}
		run = append(run, seat)
		if len(run) >= n {
			block := make([]*api.Seat, n)
			copy(block, run[len(run)-n:])
			runs = append(runs, block)
		}
	}
	return runs
}
