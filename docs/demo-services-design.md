# Demo Services Redesign: The Box Office Trio

A proposal for the three demo services behind the kaja.tools live demo — one Twirp,
one gRPC, one OpenAPI — designed from scratch so that each protocol is shown where it
shines, the domain is instantly understandable to *anyone* (not just developers), and
a single script in the kaja editor can chain all three into one story.

## Why redesign at all

The current demo pair (users = Twirp, teams = gRPC) works, but it has three
limitations as a demo:

1. **It's static.** Nothing happens unless the visitor types something. A fresh
   visitor's first `GetAllUsers` call returns whatever the last stranger left behind.
2. **It's all unary CRUD.** The two protocols look interchangeable — the demo never
   answers "why would I have both gRPC *and* Twirp *and* REST in one shop?", which is
   exactly the situation kaja exists for.
3. **The composition is thin.** The only cross-service story is "put a user id into a
   team" — a foreign-key join, not a workflow.

Adding OpenAPI is the moment to fix this: pick a domain where the three protocols
have naturally different jobs, where the output of one call is the natural input of
the next, and where no explanation is needed to understand what's going on.

## The proposal: a theatre box office

The fictional "internal microservices group" becomes the ticketing platform of a
small venue — say, **The Kaja Theatre**. Everyone on earth understands browsing
shows, watching seats sell out, and racing to buy a ticket.

| Service | Protocol | Role | Why this protocol |
|---|---|---|---|
| `events` | **OpenAPI** | The public catalog: shows, performances, venue info | Catalogs are REST-shaped in real life: resources, path params, filtering, pagination, images |
| `seating` | **gRPC** | The live seat map: watch seats get held and sold in real time | Streaming is gRPC's unique power, and a seat map filling up live is self-explanatory drama |
| `boxoffice` | **Twirp** | The counter: reserve seats, purchase, cancel | Twirp's sweet spot: simple, curl-able transactional commands |

The killer ingredient is **the crowd**: a background simulation of other customers
buying tickets through the same APIs. It's what makes the demo alive before the
visitor types anything, it's what makes the seat-map stream worth watching, and —
best of all — it means the visitor can *lose a race for a seat*, turning error
handling into a story instead of a footnote.

### The money-shot script

The demo the landing page screenshots should show — one script, three protocols,
one real workflow, with live streaming and a race in the middle:

```ts
// Find tonight's hot show (OpenAPI)
const { items } = await events.listEvents({ date: "today" });
const show = items[0];

// Watch the seat map live as the crowd buys (gRPC server streaming)
for await (const change of seating.watchSeats({ performanceId: show.performances[0].id })) {
  console.log(`${change.seatId}: ${change.status}`);   // F7: HELD, F7: SOLD, B2: HELD...
}

// Race the crowd for two good seats (Twirp)
const { reservation } = await boxoffice.reserve({
  performanceId: show.performances[0].id,
  seatIds: ["F7", "F8"],
  customerName: "Ada",
});
const { tickets } = await boxoffice.purchase({ reservationId: reservation.id });
console.log(tickets[0].confirmationCode);
```

Every step's output feeds the next step's input, the middle section visibly streams
seat updates in real time, and the ending is either a confirmation code or a
`seat_already_taken` error because a simulated buyer got there first — both are
great demo outcomes.

## Service designs

### `events` (OpenAPI 3.1)

Spec-first, mirroring the proto-first ethos: `apps/events/openapi/openapi.yaml` is
the source of truth, Go server generated with `oapi-codegen` via a `scripts/openapi`
sibling of `scripts/protoc`. The spec itself is served at
`https://kaja.tools/events/openapi.yaml` — that URL is what kaja loads.

```
GET    /venue                                → the theatre: name, address, seat-map layout
GET    /events                               → paginated (?page, ?per_page, ?genre, ?date)
GET    /events/{eventId}
GET    /events/{eventId}/poster              → binary image download
GET    /events/{eventId}/performances        → showtimes with id, starts_at, availability summary
GET    /performances/{performanceId}
```

The schema deliberately exercises things protobuf can't express, so the OpenAPI
support in kaja gets a real workout:

- string enums (`genre: concert | theatre | comedy | opera`), `format: date-time`,
  `format: uri`
- a markdown `description` field per event
- query-param filtering and pagination via `Link` header
- RFC 7807 `application/problem+json` for 404 (unknown event) and 410 (past
  performance) — showing how kaja renders REST errors next to Twirp errors and gRPC
  status codes
- one binary (non-JSON) response: the poster image

The link forward is `performances[].id` on every event — the exact value the gRPC
and Twirp services take as input.

### `seating` (gRPC)

```proto
syntax = "proto3";
package seating;

service Seating {
  // Unary: full snapshot of a performance's seat map.
  rpc GetSeatMap(GetSeatMapRequest) returns (GetSeatMapResponse);
  // Server streaming: the wow moment. Sends a snapshot, then pushes every
  // hold/sale/release live until the client disconnects.
  rpc WatchSeats(WatchSeatsRequest) returns (stream SeatChange);
  // Bidirectional streaming (stretch): an interactive picking session — the client
  // streams hold/release commands while the server streams everyone's changes back.
  rpc PickSeats(stream PickCommand) returns (stream SeatChange);
}

message Seat {
  string id = 1;            // "F7" — row letter + seat number
  string section = 2;       // ORCHESTRA, BALCONY
  SeatStatus status = 3;    // AVAILABLE, HELD, SOLD
  int32 price_cents = 4;
}

message SeatChange {
  string performance_id = 1;
  string seat_id = 2;
  SeatStatus status = 3;
  string changed_at = 4;
}
```

The seat map is small enough to read in a response and big enough to feel real:
orchestra rows A–M × seats 1–20 plus a balcony. `WatchSeats` on a hot performance
shows a steady drip of `HELD`/`SOLD` changes from the crowd; unary `GetSeatMap`
covers the plain-RPC case. `PickSeats` is the optional bidi showcase — worth having
because nothing else in the demo exercises bidirectional streaming (today only
quirks does).

`seating` **owns seat state**. Holds and sales only enter through its internal API,
which is what makes the race honest: the visitor's reserve and the crowd's reserves
contend on the same state.

### `boxoffice` (Twirp)

```proto
syntax = "proto3";
package boxoffice;

service BoxOffice {
  rpc Reserve(ReserveRequest) returns (ReserveResponse);          // hold seats, 2-minute TTL
  rpc Purchase(PurchaseRequest) returns (PurchaseResponse);       // reservation -> tickets
  rpc CancelReservation(CancelReservationRequest) returns (CancelReservationResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);       // look up by confirmation code
}

message Reservation {
  string id = 1;
  string performance_id = 2;
  repeated string seat_ids = 3;
  string customer_name = 4;
  string expires_at = 5;      // holds auto-release, keeping the demo self-healing
  int32 total_cents = 6;
}

message Ticket {
  string confirmation_code = 1;   // "KAJA-7F3B"
  string event_id = 2;            // links back to the REST catalog
  string performance_id = 3;
  string seat_id = 4;
}
```

`Reserve` returns a proper Twirp error (`already_exists`, message
`seat F7 was just taken`) when the crowd wins the race — error handling demoed on
the Twirp side, complementing the REST 404/410 on the OpenAPI side. Tickets carry
`event_id`, so composition also runs backwards: look up an order (Twirp) → fetch the
event and poster (REST) → watch the remaining seats (gRPC).

### The crowd (simulation engine)

The crowd is part of the `boxoffice` service and buys through the same internal path
as real visitors — dogfooding, and it keeps the microservices story honest:

- Each performance gets a demand profile: a Friday-night headliner sells fast, a
  Tuesday matinee dribbles. Simulated buyers reserve 1–4 adjacent seats, sometimes
  abandon their hold (which visibly frees seats again), mostly purchase.
- **Rolling schedule**: `events` always maintains ~7 days of upcoming performances,
  generating new ones as old ones pass. Hot shows selling out is realistic and fine —
  there's always another performance with seats.
- Holds expire after 2 minutes, so no one can brick a performance by hoarding holds.

Seeded content is fictional and a little charming — acts worth naming well, across
genres, at The Kaja Theatre. Posters generated once and committed as static assets.

## What happens to users, teams, and quirks

- **users, teams**: retire from the public demo (`apps/kaja/kaja.json`) once the trio
  is live; delete the apps in a follow-up once nothing references them. They taught
  us the patterns (pebble storage, Twirp path prefixes, gRPC ingress) that the trio
  reuses.
- **quirks**: keep, but as the edge-case test harness it already is rather than part
  of the headline demo. Optionally keep it in the demo config as a clearly-labeled
  "kitchen sink" project.

## Infrastructure plan

Everything follows existing patterns in this repo:

- `apps/events` — plain HTTP server (chi + oapi-codegen), `k8s/base/events.yaml`,
  path `/events` in `k8s/base/ingress.yaml`, serving both the API and
  `/events/openapi.yaml`.
- `apps/seating` — gRPC server, `k8s/base/seating.yaml`, path `/seating.Seating/` in
  `k8s/base/ingress-grpc.yaml`. If teams is retired, repoint the gRPC reflection
  paths (currently → `teams-service`) at `seating-service`.
- `apps/boxoffice` — Twirp server with
  `twirp.WithServerPathPrefix("/boxoffice/twirp")`, `k8s/base/boxoffice.yaml`, path
  `/boxoffice` in `k8s/base/ingress.yaml`.
- `apps/kaja/kaja.json` — three projects: `events` (the new OpenAPI protocol enum,
  pointed at the spec URL), `seating` (`RPC_PROTOCOL_GRPC`, `dns:kaja.tools:443`),
  `boxoffice` (`RPC_PROTOCOL_TWIRP`, `https://kaja.tools/boxoffice`).
  *Assumption: kaja's config gains an OpenAPI protocol value; the exact enum name
  follows whatever kaja ships.*
- Storage: pebble per service on the workspace volume, like users today.
- Inter-service calls (`boxoffice → seating` for holds/sales, both → `events` for
  the schedule) go over in-cluster service DNS.

### Public-demo hygiene

The demo is on the open internet, so the trio is self-maintaining:

- Rolling performance window + hold TTLs mean the demo can never be bricked; a
  nightly reset is acceptable and invisible because seeded data looks the same as
  organic data.
- Caps: max 6 seats per reservation, length limits on `customer_name` (the only
  user-supplied string), a modest rate limit on `Reserve`.
- Rolling GC: archive performances older than a day, drop their seat state and
  orders.

## Alternatives considered

- **CI/delivery platform** (builds = Twirp, log tailing = gRPC, release registry =
  OpenAPI). The strongest *developer*-facing option — it mirrors kaja's audience's
  actual stack — and an earlier draft of this proposal. Ruled out for being
  insider-facing: a demo anyone can understand at a glance beats one that requires
  living in CI. The box office keeps its best ideas (lifecycle simulation, background
  activity, cross-protocol chaining) in a universally legible costume.
- **Radio station / jukebox** (REST music catalog, Twirp song requests, gRPC
  now-playing broadcast you tail until your song plays). Charming and alive by
  definition; close second. The box office won on drama — a race with a possible
  409-style loss beats passive waiting.
- **Food delivery** (REST menu, Twirp orders, gRPC pizza-tracker stream). Universally
  understood but tutorial-flavored territory, adjacent to the pet-store cliché.
- **Mission control** (REST missions, Twirp launch scheduling, gRPC countdown
  telemetry). The most screenshotable, but a costume more than a domain — no natural
  REST catalog anyone would browse, and no user-vs-world contention.
- **Keep users/teams, add an OpenAPI booking service.** Cheapest option, but it stays
  all-unary CRUD — it never shows streaming and never answers why the protocols
  differ.

## Suggested build order

1. `events` (OpenAPI) alone with the seeded catalog and rolling schedule — it's the
   new protocol and can ship to the demo first as a standalone project.
2. `seating` with seat maps and `WatchSeats`, fed by a first cut of the crowd.
3. `boxoffice` + the full crowd simulation contending with real visitors.
4. Swap `apps/kaja/kaja.json` to the trio, retire users/teams from the demo, update
   ingress and reflection routing.
5. Refresh landing-page screenshots/demo video around the money-shot script.
