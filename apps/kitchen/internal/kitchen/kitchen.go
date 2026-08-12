// Package kitchen is what the bakery knows that the book does not print: what
// is actually in a dough, what is on the shelf this morning, what to reach for
// when it isn't, and what to drink with the result.
//
// It is deliberately opinionated. A tool that answers "it depends" is a tool
// nobody calls twice.
package kitchen

import (
	"sort"
	"strings"
)

// Ingredient is a share of the dough by weight. A recipe's shares add up to
// 100, so scaling a batch is one multiplication and no unit conversions.
type Ingredient struct {
	Name    string
	Percent float64
}

var recipes = map[string][]Ingredient{
	"brown-butter": {
		{"plain flour", 38},
		{"brown butter", 22},
		{"chopped dark chocolate", 16},
		{"light brown sugar", 14},
		{"egg", 8},
		{"vanilla", 1},
		{"flaky salt", 1},
	},
	"rye-choc": {
		{"dark rye flour", 26},
		{"dark chocolate", 24},
		{"butter", 18},
		{"caster sugar", 12},
		{"plain flour", 10},
		{"egg", 8},
		{"cocoa", 2},
	},
	"snickerdoodle": {
		{"plain flour", 42},
		{"caster sugar", 24},
		{"butter", 22},
		{"egg", 8},
		{"cream of tartar", 2},
		{"cinnamon", 2},
	},
	"ginger-snap": {
		{"plain flour", 40},
		{"butter", 20},
		{"caster sugar", 20},
		{"molasses", 8},
		{"egg", 6},
		{"crystallised ginger", 3},
		{"ground ginger", 2},
		{"black pepper", 1},
	},
	"oat-raisin": {
		{"rolled oats", 26},
		{"plain flour", 20},
		{"butter", 18},
		{"light brown sugar", 14},
		{"raisins", 14},
		{"egg", 7},
		{"cinnamon", 1},
	},
	"shortbread": {
		{"plain flour", 50},
		{"butter", 33},
		{"caster sugar", 15},
		{"flaky salt", 2},
	},
	"peanut-blossom": {
		{"plain flour", 30},
		{"peanut butter", 22},
		{"caster sugar", 16},
		{"butter", 14},
		{"chocolate kisses", 10},
		{"egg", 7},
		{"flaky salt", 1},
	},
}

// What the shelf is short of this morning. Two gaps, both in recipes somebody
// will actually pick, so `scale` has a reason to send you to `substitute`.
var outOfStock = map[string]bool{
	"cream of tartar":     true,
	"crystallised ginger": true,
}

// Swap is what to use instead of something, and what it costs you.
type Swap struct {
	Use   string
	Ratio string
	Note  string
}

var swaps = map[string]Swap{
	"cream of tartar": {
		Use:   "lemon juice",
		Ratio: "two parts lemon juice to one part cream of tartar",
		Note:  "It is in there for the tang and for keeping the crumb soft. You will get the tang and lose a little of the softness, which is a fair trade at eight in the morning.",
	},
	"crystallised ginger": {
		Use:   "stem ginger in syrup, drained and chopped small",
		Ratio: "1:1 by weight",
		Note:  "Pat it dry or the extra syrup will slacken the dough and the snap will not snap.",
	},
	"butter": {
		Use:   "a good block margarine — not the soft tub kind",
		Ratio: "1:1 by weight",
		Note:  "It has more water in it, so the dough spreads further. Chill it for an hour and bake it a minute shorter.",
	},
	"brown butter": {
		Use:   "melted butter with a spoonful of molasses stirred in",
		Ratio: "1:1 by weight, plus 5g molasses per 100g",
		Note:  "You are faking the toffee note rather than making it. Close enough that nobody has ever sent one back.",
	},
	"egg": {
		Use:   "apple sauce",
		Ratio: "60g per egg",
		Note:  "It binds but it does not lift, so expect a denser, chewier cookie. In shortbread it does nothing at all, which is fine — there is no egg in shortbread.",
	},
	"plain flour": {
		Use:   "a 1:1 gluten-free blend with xanthan already in it",
		Ratio: "1:1 by weight",
		Note:  "Rest the dough for half an hour before it goes near a tray, or it bakes gritty.",
	},
	"dark rye flour": {
		Use:   "wholemeal flour",
		Ratio: "1:1 by weight",
		Note:  "You lose the slightly savoury edge that is the whole reason for the rye. It is still a good cookie, it is just a different one.",
	},
	"caster sugar": {
		Use:   "granulated sugar, buzzed for ten seconds",
		Ratio: "1:1 by weight",
		Note:  "Straight granulated works too; the cookie comes out a shade craggier, which some people prefer.",
	},
	"light brown sugar": {
		Use:   "caster sugar with molasses",
		Ratio: "1:1 by weight, plus 5g molasses per 100g",
		Note:  "This is what light brown sugar is anyway. Nobody has to know.",
	},
	"molasses": {
		Use:   "dark treacle, or honey at a push",
		Ratio: "1:1 by weight",
		Note:  "Honey browns faster, so drop the oven 10 degrees.",
	},
	"dark chocolate": {
		Use:   "anything over 60%",
		Ratio: "1:1 by weight",
		Note:  "Chips hold their shape and a chopped bar does not, which is the entire argument between chips and chunks.",
	},
	"raisins": {
		Use:   "sultanas, or chopped dates",
		Ratio: "1:1 by weight",
		Note:  "Soak them in strong tea either way. That is the part people are actually complaining about.",
	},
	"rolled oats": {
		Use:   "jumbo oats, chopped once with a knife",
		Ratio: "1:1 by weight",
		Note:  "Do not use instant. It dissolves and you have made a flat sugar cookie with regrets in it.",
	},
	"peanut butter": {
		Use:   "any smooth nut butter",
		Ratio: "1:1 by weight",
		Note:  "Almond behaves almost identically. Tahini is better than it sounds and worse than it should be.",
	},
	"vanilla": {
		Use:   "leave it out",
		Ratio: "—",
		Note:  "One teaspoon in a batch this size is doing less than the label implies. Nobody will find it missing.",
	},
	"flaky salt": {
		Use:   "fine salt",
		Ratio: "half the weight",
		Note:  "Fine salt dissolves, so it seasons instead of crunching. Use less of it and accept that the top will be quieter.",
	},
	"cinnamon": {
		Use:   "mixed spice",
		Ratio: "1:1 by weight",
		Note:  "It brings cloves along with it. In a snickerdoodle that is a bigger change than it sounds.",
	},
}

// What to reach for when the reason rules out dairy or eggs. Only the two
// ingredients where the ordinary answer would be no help at all.
var veganSwaps = map[string]Swap{
	"butter": {
		Use:   "a vegan block, the firm kind sold for pastry",
		Ratio: "1:1 by weight",
		Note:  "The soft spreads are half water and will give you a puddle. Chill the dough properly and it behaves.",
	},
	"brown butter": {
		Use:   "a vegan block browned the same way, with a pinch of nutritional yeast",
		Ratio: "1:1 by weight",
		Note:  "It browns paler and faster. Take it off the heat when it smells like biscuits, not when it looks like tea.",
	},
	"egg": {
		Use:   "ground flaxseed in water",
		Ratio: "1 tablespoon flax to 3 of water, per egg, left ten minutes",
		Note:  "It binds well and lifts not at all, which suits every cookie in this book except the peanut blossom.",
	},
}

// Pairing is the answer to the only question anybody asks about a cookie once
// it is out of the oven.
type Pairing struct {
	Drink   string
	Why     string
	AlsoTry string
}

var pairings = map[string]Pairing{
	"brown-butter": {
		Drink:   "a flat white",
		Why:     "The toffee in the browned butter needs something with body behind it. Filter coffee gets flattened; the milk in a flat white meets it halfway.",
		AlsoTry: "cold milk, if it is still warm from the rack",
	},
	"rye-choc": {
		Drink:   "an espresso",
		Why:     "Rye is faintly savoury and the chocolate is not sweet enough to argue with it. Anything milky just gets in between them.",
		AlsoTry: "a stout, after six",
	},
	"snickerdoodle": {
		Drink:   "black tea with milk",
		Why:     "Cinnamon and sugar want something plain and hot. This is a cookie that belongs beside a mug, not a cup.",
		AlsoTry: "chai, if you want the cinnamon twice",
	},
	"ginger-snap": {
		Drink:   "strong black coffee",
		Why:     "There is pepper in it. Coffee is the only thing that takes the heat seriously.",
		AlsoTry: "ginger beer, which is either brilliant or too much",
	},
	"oat-raisin": {
		Drink:   "builder's tea",
		Why:     "The raisins were soaked in tea. Finishing the pot with the cookie is only fair.",
		AlsoTry: "a glass of cold milk",
	},
	"shortbread": {
		Drink:   "Earl Grey, no milk",
		Why:     "Four ingredients and three of them are butter, sugar and salt. The bergamot cuts it; milk would just add more fat to fat.",
		AlsoTry: "a small whisky, which is traditional and not our fault",
	},
	"peanut-blossom": {
		Drink:   "cold milk",
		Why:     "There is no second answer. Peanut butter and a lump of chocolate is a childhood flavour and it wants a childhood drink.",
		AlsoTry: "cold milk",
	},
}

// Recipe returns what goes into a cookie, biggest share first.
func Recipe(cookieID string) ([]Ingredient, bool) {
	r, ok := recipes[cookieID]
	return r, ok
}

// InStock says whether the shelf has an ingredient this morning.
func InStock(ingredient string) bool {
	return !outOfStock[normalize(ingredient)]
}

// Substitute finds what to use instead. why is free text — a reason mentioning
// veganism, dairy or eggs gets a different answer where there is one to give.
func Substitute(ingredient, why string) (Swap, bool) {
	name := normalize(ingredient)
	if avoidsAnimals(why) {
		if swap, ok := veganSwaps[name]; ok {
			return swap, true
		}
	}
	if swap, ok := swaps[name]; ok {
		return swap, true
	}
	// "chocolate" should find the dark chocolate answer, and "chopped dark
	// chocolate" should find it too — so the near miss is allowed to go either
	// way round. Longest match wins, so "flour" cannot answer for the rye.
	best, found := "", false
	var match Swap
	for key, swap := range swaps {
		if !strings.Contains(key, name) && !strings.Contains(name, key) {
			continue
		}
		if len(key) > len(best) {
			best, match, found = key, swap, true
		}
	}
	return match, found
}

// PairingFor returns what to drink with a cookie.
func PairingFor(cookieID string) (Pairing, bool) {
	p, ok := pairings[cookieID]
	return p, ok
}

// Ingredients lists every ingredient the book uses, for an error message that
// can say what there is instead of only what there isn't.
func Ingredients() []string {
	seen := map[string]bool{}
	var out []string
	for _, recipe := range recipes {
		for _, i := range recipe {
			if !seen[i.Name] {
				seen[i.Name] = true
				out = append(out, i.Name)
			}
		}
	}
	// Recipes are a map, so without this the same question gets its answer in a
	// different order every time.
	sort.Strings(out)
	return out
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func avoidsAnimals(why string) bool {
	why = strings.ToLower(why)
	for _, word := range []string{"vegan", "dairy", "lactose", "egg", "plant"} {
		if strings.Contains(why, word) {
			return true
		}
	}
	return false
}
