// Package recipes holds The Kaja Bakery's book. It is a fixed list: nothing
// here changes, nothing sells out, and every id in it is always bakeable —
// everything that moves lives in the oven service instead.
//
// A cookie is the unit the whole demo is built on. Its id is what you pass to
// the oven to bake it and to the kitchen to ask about it, so there is exactly
// one identifier to carry around.
package recipes

type Texture string

const (
	Chewy   Texture = "chewy"
	Soft    Texture = "soft"
	Crisp   Texture = "crisp"
	Fudgy   Texture = "fudgy"
	Crumbly Texture = "crumbly"
)

type Cookie struct {
	ID          string
	Name        string
	Texture     Texture
	Tagline     string
	Description string
	// Minutes in a real oven. The oven service runs on demo time — a minute
	// here is a second there — so a bake finishes while you watch it.
	BakeMinutes         int
	OvenCelsius         int
	PerTray             int
	DoughGramsPerCookie int
	PriceCents          int
	Allergens           []string
}

// Quickest bake first, which is also roughly the order the counter fills up.
var cookies = []Cookie{
	{
		ID:      "rye-choc",
		Name:    "Rye Double Chocolate",
		Texture: Fudgy,
		Tagline: "Dark, and a little savoury about it.",
		Description: "Dark rye flour against two kinds of chocolate, which is what keeps it " +
			"from tasting like a brownie that lost its nerve. It comes out looking underbaked. " +
			"It is meant to.",
		BakeMinutes:         9,
		OvenCelsius:         200,
		PerTray:             12,
		DoughGramsPerCookie: 42,
		PriceCents:          340,
		Allergens:           []string{"gluten", "dairy", "egg"},
	},
	{
		ID:      "peanut-blossom",
		Name:    "Peanut Blossom",
		Texture: Soft,
		Tagline: "The one with the chocolate pressed into the middle.",
		Description: "Rolled in sugar, baked until it cracks, then a chocolate kiss pushed in " +
			"the second it comes out. Press too early and it slides off; press too late and it " +
			"sits on top like a hat.",
		BakeMinutes:         9,
		OvenCelsius:         190,
		PerTray:             15,
		DoughGramsPerCookie: 40,
		PriceCents:          300,
		Allergens:           []string{"gluten", "dairy", "egg", "nuts"},
	},
	{
		ID:      "snickerdoodle",
		Name:    "Snickerdoodle",
		Texture: Soft,
		Tagline: "Cinnamon, sugar, no argument.",
		Description: "Cream of tartar is what makes it tangy and what makes it stay soft. " +
			"Leave it out and you have baked a sugar cookie and disappointed somebody.",
		BakeMinutes:         10,
		OvenCelsius:         180,
		PerTray:             15,
		DoughGramsPerCookie: 38,
		PriceCents:          280,
		Allergens:           []string{"gluten", "dairy", "egg"},
	},
	{
		ID:      "brown-butter",
		Name:    "Brown Butter Chocolate Chunk",
		Texture: Chewy,
		Tagline: "The one we are known for.",
		Description: "Butter cooked until it smells like toffee, and chunks rather than chips " +
			"so you get seams of chocolate instead of dots. The dough rests overnight, which is " +
			"the entire trick and the reason we cannot do it to order.",
		BakeMinutes:         11,
		OvenCelsius:         190,
		PerTray:             12,
		DoughGramsPerCookie: 45,
		PriceCents:          320,
		Allergens:           []string{"gluten", "dairy", "egg"},
	},
	{
		ID:      "ginger-snap",
		Name:    "Ginger Snap",
		Texture: Crisp,
		Tagline: "Snaps. Loudly.",
		Description: "Three gingers — ground, fresh and crystallised — and enough black pepper " +
			"that people ask what is in it and then ask for another one.",
		BakeMinutes:         12,
		OvenCelsius:         175,
		PerTray:             15,
		DoughGramsPerCookie: 32,
		PriceCents:          260,
		Allergens:           []string{"gluten", "dairy", "egg"},
	},
	{
		ID:      "oat-raisin",
		Name:    "Oat and Raisin",
		Texture: Chewy,
		Tagline: "Not a chocolate chip cookie. Never said it was.",
		Description: "The most unfairly treated cookie in the case. The raisins are soaked in " +
			"strong tea first, so they stay plump instead of turning into the little dry pellets " +
			"everybody is actually complaining about.",
		BakeMinutes:         12,
		OvenCelsius:         180,
		PerTray:             12,
		DoughGramsPerCookie: 46,
		PriceCents:          290,
		Allergens:           []string{"gluten", "dairy", "egg"},
	},
	{
		ID:      "shortbread",
		Name:    "Salted Shortbread",
		Texture: Crumbly,
		Tagline: "Four ingredients and nowhere to hide.",
		Description: "Flour, butter, sugar, salt. Baked low and long until it is barely blond, " +
			"because the moment it takes colour it stops tasting of butter and starts tasting " +
			"of biscuit.",
		BakeMinutes:         18,
		OvenCelsius:         160,
		PerTray:             20,
		DoughGramsPerCookie: 30,
		PriceCents:          240,
		Allergens:           []string{"gluten", "dairy"},
	},
}

// Cookies returns the whole book, quickest bake first.
func Cookies() []Cookie {
	out := make([]Cookie, len(cookies))
	copy(out, cookies)
	return out
}

// ByID looks one recipe up.
func ByID(id string) (Cookie, bool) {
	for _, c := range cookies {
		if c.ID == id {
			return c, true
		}
	}
	return Cookie{}, false
}
