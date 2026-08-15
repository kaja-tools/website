package catalog

import (
	"log/slog"
	"time"
)

// Theater is one house. The chain runs a dozen of them across the country,
// which is what makes a screening worth locating: two people asking what is
// on tonight are not asking about the same building.
type Theater struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	City    string `json:"city"`
	State   string `json:"state"`
	Address string `json:"address"`
	// TimeZone is the IANA name for the city the house is in. Every showtime
	// this service publishes is that house's local evening, written with its
	// own offset, because half past seven means half past seven wherever you
	// bought the ticket.
	TimeZone string `json:"timeZone"`

	// What this house charges against the chain's own price, as a percentage.
	// A restored downtown palace is dearer than a suburban twin, and it is
	// the only thing a theater decides about a film's price.
	priceIndex int
	loc        *time.Location
}

// The chain. Twelve houses, five time zones, and no two in the same city —
// the demo is about joining a screening to a place, so a place has to be
// somewhere.
var theaters = []Theater{
	{ID: "the-marquee", Name: "The Marquee", City: "New York", State: "NY", Address: "1140 Amsterdam Avenue", TimeZone: "America/New_York", priceIndex: 130},
	{ID: "beacon-hill", Name: "Beacon Hill Picture House", City: "Boston", State: "MA", Address: "62 Charles Street", TimeZone: "America/New_York", priceIndex: 115},
	{ID: "the-foundry", Name: "The Foundry", City: "Pittsburgh", State: "PA", Address: "815 Liberty Avenue", TimeZone: "America/New_York", priceIndex: 95},
	{ID: "tidewater", Name: "Tidewater Cinema", City: "Miami", State: "FL", Address: "340 Alton Road", TimeZone: "America/New_York", priceIndex: 110},
	{ID: "the-lantern", Name: "The Lantern", City: "Chicago", State: "IL", Address: "2233 North Lincoln Avenue", TimeZone: "America/Chicago", priceIndex: 110},
	{ID: "riverbend", Name: "Riverbend Cinema", City: "Austin", State: "TX", Address: "1109 South Lamar Boulevard", TimeZone: "America/Chicago", priceIndex: 100},
	{ID: "cadence", Name: "Cadence Theatre", City: "Nashville", State: "TN", Address: "508 Woodland Street", TimeZone: "America/Chicago", priceIndex: 95},
	{ID: "bayou-palace", Name: "Bayou Palace", City: "New Orleans", State: "LA", Address: "723 Magazine Street", TimeZone: "America/Chicago", priceIndex: 100},
	{ID: "copper-room", Name: "Copper Room", City: "Denver", State: "CO", Address: "1470 Larimer Street", TimeZone: "America/Denver", priceIndex: 105},
	{ID: "mesa-picture-house", Name: "Mesa Picture House", City: "Phoenix", State: "AZ", Address: "210 East Roosevelt Street", TimeZone: "America/Phoenix", priceIndex: 90},
	{ID: "sunset-reel", Name: "Sunset Reel", City: "Los Angeles", State: "CA", Address: "4360 Sunset Boulevard", TimeZone: "America/Los_Angeles", priceIndex: 125},
	{ID: "harbor-light", Name: "Harbor Light Cinema", City: "Seattle", State: "WA", Address: "1918 Ballard Avenue", TimeZone: "America/Los_Angeles", priceIndex: 115},
}

var theatersByID map[string]Theater

func loadTheaters() {
	theatersByID = make(map[string]Theater, len(theaters))
	for i := range theaters {
		t := &theaters[i]
		loc, err := time.LoadLocation(t.TimeZone)
		if err != nil {
			// The binary embeds the zone database, so this cannot happen for
			// the names above — a house whose zone went missing keeps trading
			// in UTC rather than taking the service down.
			slog.Warn("theater has no time zone", "theater", t.ID, "timeZone", t.TimeZone, "error", err)
			loc = time.UTC
		}
		t.loc = loc
		theatersByID[t.ID] = *t
	}
}

func Theaters() []Theater { return theaters }

func TheaterByID(id string) (Theater, bool) {
	t, ok := theatersByID[id]
	return t, ok
}
