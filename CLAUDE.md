# Agent Guidelines

## Pull Requests

**Write the shortest description that still says what changed.** The diff is
the detail; the description is the headline. Almost every PR here should be one
sentence — a title and a single line. Reach for bullets only when the PR really
does several unrelated things, and then it is one short bullet each, no nesting.

Hard limits: no headings, no "Summary"/"Changes"/"Testing"/"Notes" sections, no
tables, no code blocks, no generated-by or model boilerplate, no emoji. Never
explain the background, the root cause, the alternatives you rejected, or how
you tested — link the issue or the related PR instead of retelling it.

If you are wondering whether a sentence earns its place, it does not. Cut it.

## Proto Files

To regenerate proto files after modifying `.proto` definitions, run:

```bash
./scripts/protoc
```

This script:

- Installs a consistent version of [protoc-go](https://github.com/wham/protoc-go)
  (a pure Go protoc with the well-known types embedded) into the `build/`
  directory via `go install`
- Installs the required Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-twirp`)
- Regenerates all proto files for the quirks (Twirp) and seating (gRPC) services,
  and the seating client the concierge is built on

Do not use system-installed protoc or manually run protoc commands.

## Demo Services

The demo is the Theatre, a chain of repertory cinemas: three services, three
protocols — OpenAPI, gRPC and MCP. Twirp is deliberately not part of it (it is
still supported by kaja itself, and `apps/quirks` still exercises it).

- `apps/theatre` (OpenAPI) is the data. **Three operations, and only the third
  relates anything:** `GET /movies` is the catalog, `GET /theaters` is the
  dozen houses, and `GET /shows` is the schedule — a movie id, a theater id, a
  time and a price, and nothing else. Both paged lists take no parameters for
  the first page, then follow `nextCursor` until it is `null`; the theaters
  come back in one response, so that one has no cursor at all.

  **The schedule is deliberately only a relationship**, because reading a
  programme is then a join, and a join is what the demo scripts are for. A
  page of screenings is filled in with one `GET /movies?ids=…` rather than a
  call per row, which is why that filter exists.
- `apps/seating` (gRPC) owns live seat state: `GetSeatMap`, `BookSeats`, and
  the streaming `WatchSeats`. **Buying is one call** — `BookSeats` is all or
  nothing, and it is the only write in the demo, which is what makes it the one
  thing worth putting behind `kaja.approve`.
- `apps/concierge` (MCP) is front of house: `suggest_film`, `best_seats`,
  `write_confirmation`. It owns no data — it reads the three lists over HTTP
  and the seat map over gRPC, and turns a sentence somebody typed (and the
  town they are in) into a `showId` and a list of seat ids the other two
  services understand. The one thing it owns is the join: it puts the three
  lists back together every few minutes, and everything downstream of that
  sees screenings that know their film and their house.

**The films are real; the cinemas are not.**
`apps/theatre/internal/catalog/films.json` is a bit over a thousand actual
films with their real director, year, running time, language and a sentence
of synopsis, because a demo you understand in seconds cannot also be teaching
you an invented repertoire. Nothing about a film is invented. The chain is:
the dozen houses are in `theaters.go`, and everything the chain decides —
which of them play a film, at what time, for what money — is derived from a
movie id and a theater id in `catalog.go` rather than stored, so there is no
schedule to keep in step with the films.

A show id is the two ids it relates, written out: `dune-part-two@the-lantern`.
Nothing parses it, and nothing should — it is readable so that a screening
says what it is without a lookup.

Regenerate the repertory with `scripts/catalog`, which takes the facts from
Wikidata (CC0) and the synopsis from the English Wikipedia article's lead
(CC BY-SA 4.0) and caches every response under `build/`. A film whose article
never says what happens in it is left out rather than described by guesswork.
The ten films the demo was built on are in the script by hand and keep their
own synopses and their ids, which the OpenAPI examples point at.

A screening's `id` is still the only identifier that crosses between services:
copy it out of `/shows` and pass it to seating as `showId`, or let the
concierge hand you one. Movie and theater ids stay inside the Theatre service,
which is what keeps "buy a ticket" two calls however many lists the schedule
is joined against.

**The concierge holds no opinions keyed by film id.** Its taste
(`internal/concierge/taste.go`) is signals read off what the catalog already
publishes — genre, running time, year, language — so a film added tomorrow is
understood the same day, and there is no second catalog here to keep in step.
Which town somebody is in is a filter over the joined programme (`InCity`),
not a taste: the same film is on in four cities this week.

Houses are not all the same size. `apps/seating/internal/store` picks one of
three layouts off the screening's `theaterId` — the smallest has no balcony —
so a seat map is a different shape two towns over, and nothing about the house
has to be published to say so.

`apps/seating/internal/crowd` simulates other customers in-process so the seat
map is always moving. It still holds and releases seats even though holding is
no longer in the API — a hold is what makes a seat map worth reading twice.
`CROWD=off` disables it.

`apps/seating/internal/ratelimit` caps a caller at 40 calls per 10 seconds, in
memory and per client, and says so on every response — the `RateLimit` and
`RateLimit-Policy` structured fields of
draft-ietf-httpapi-ratelimit-headers, the older
`RateLimit-Limit`/`-Remaining`/`-Reset` triple beside them, and `Retry-After` on
a refusal, which comes back as `RESOURCE_EXHAUSTED`. It is high enough that
reading a programme never nears it and low enough that a script written to hit
it does, and the window is short so that hitting it costs seconds rather than
half a minute; `RATE_LIMIT=off` disables it. A caller is its `Fly-Client-IP` behind
the edge and its peer address otherwise, so it names the machine, not the
person: a hosted Kaja calls from its own server and everyone using it shares a
budget. Reflection is exempt, so a spent budget never leaves a client unable to
learn what the service is.

`apps/quirks` is not part of the demo — it is the Twirp protocol testbed for
edge cases (odd names, deep nesting, panics, streaming RPCs that Twirp renders
as unary).

## The Website

The website is `home/` at the repository root — an [Astro](https://astro.build)
site with Tailwind CSS v4, built to static files and served by Caddy. `apps/`
is the demo services only; the website does not belong there and does not move
back.

```
home/
  src/pages/       one file per route (index, docs, privacy, 404)
  src/layouts/     the page shell — <head>, header, footer
  src/components/  everything reused across pages
  src/data/        the lists a page is built from, out of its markup
  src/styles/      global.css: the Tailwind import and the design tokens
  public/          served verbatim at the root: favicons, logo, /assets/
  Caddyfile        serving rules for the runtime image
```

- `public/assets/` is screenshots and demo videos only.
- **The palette is kaja-ui's.** The site is built from the same shadcn neutral
  tokens as the app — `background`, `foreground`, `card`, `muted`,
  `muted-foreground`, `border`, `primary`, `accent`, `ring` — dark values only,
  because the site has no light theme and no toggle. `Button` is the app's
  Button as a link, variant for variant; the 404 is the app's Blankslate. When
  a page needs a control the app already has, copy the app's class list.
- **Colours, fonts, sizes, radii and widths are tokens.** They live in the
  `@theme` block in `src/styles/global.css` and are used through the utilities
  Tailwind generates from them (`bg-card`, `text-hero`, `max-w-page`,
  `rounded-frame`). Don't write a raw hex or a one-off pixel value in a
  component; add or reuse a token.
- Tailwind's preset `text-*` sizes carry their own `line-height`, and the
  design's body copy runs looser than the presets do, so pin it with the slash
  modifier (`text-lg/[1.6]`) wherever the design gives a line height.
- The brand mark's gradient is `Mark.astro` and the two favicons; the docs'
  snippet tags borrow its stops. The only other non-neutral colour is the amber
  of the docs' watch note (`Note.astro`) and the red `ink` the home page marks
  the screenshot up in.
- **The four protocol marks are kaja's own** (`ui/src/protocolMarks.ts` in
  [wham/kaja](https://github.com/wham/kaja)), copied into `Icon.astro` the way
  the lucide glyphs are, so the site and the app draw the same thing. They are
  what tells gRPC, OpenAPI, MCP and Twirp apart wherever the four are named
  together — the hero diagram and the app types in the docs — which is what the
  hero's coloured dots used to do badly. Don't draw a
  fifth: a mark exists per protocol, not per idea.
- **The hero is the app's own map** (`ui/src/McpMap.tsx`), ported stroke for
  stroke: an agent, Kaja drawn as its own canvas, the four protocols. Three
  weights and no colour — `wire` for the wires and the canvas blocks,
  `muted-foreground` for the node frames and arrows, `foreground` for names and
  marks. **There is no screenshot in it**: the app itself is the next thing on
  the page, and a picture of the window beside the drawing says the same thing
  twice. The site's own layer is the entrance, and the one thing not ported
  verbatim is the width of the app cards, because the site's body face is wider
  than the app's. Below `md` the drawing is too small to read, so the same five
  things are a plain stack of cards.
- **The home page below the hero is one screenshot, seven times over.** The
  app is shown once and whole under the drawing, and every close-up after it
  (`Poster.astro`, from `data/poster.ts`) is a region of that same
  `app-hero.png` drawn as a background rather than a file of its own — a crop
  is four fractions, and a new screenshot at the same size needs no new files.
  Each one is a statement beside its crop, with a red line from the statement
  to a box around the thing it names; the line and the box are geometry
  `Motion.astro` measures, because both halves drift as the page scrolls, so
  without script the page is the statements and the crops and no ink. A new
  statement is an entry in `data/poster.ts`, not markup.
- **The docs are one page, not one per platform** (`src/pages/docs.astro`).
  Most of what there is to say is true of both builds, so the two differences
  per section are `<Platform only="desktop">` / `only="docker"` blocks in the
  one page and the toggle picks between them clientside — **Desktop first, and
  the default**, with Docker as the variant. A second page would fork every
  shared paragraph to keep four differences apart; if that balance ever tips,
  `Platform` is the seam to split on. A reader with no JavaScript gets both
  blocks, labelled, by the `noscript` rule at the foot of the page.
- `data/docs.ts` is the section list — the nav, the screenshot rail and the
  scroll spy all read it, so a section is added there and its body written in
  the page under the same `id`. A section with no `shot` leaves the rail empty
  rather than borrowing a picture of something else; **Variables and Agents
  have none yet**, because `public/assets/` holds the crops the docs already
  point at and none of them is of those screens.
- Snippets are plain strings in `data/snippets.ts`, highlighted at build time
  by Shiki in `styles/codeTheme.ts` — which maps scopes onto the `code-*`
  custom properties rather than repeating them, so the palette stays in
  `global.css`. Keep a snippet narrow enough to fit the column; one you have
  to scroll sideways is one nobody finishes.
- **Everything scroll-driven is a data attribute `Motion.astro` reads** —
  `data-parallax` and `data-ink-item`. There is one scroll listener on the site
  and it lives there; a section stays plain markup. The drift is decorative and
  drops out under `prefers-reduced-motion`; the ink runs either way, because
  the line and the box are how a statement points at the thing it is about, and
  the reduced-motion rule already collapsed the drawing to an instant. The docs
  page has its own one listener, `DocsMotion.astro`, on the same rule.

Commands, all from `home/`:

```bash
npm run dev           # dev server with HMR at localhost:4321
npm run build         # static build into home/dist
npm run preview       # serve the build
npm run check         # astro check (TypeScript + template diagnostics)
npm run format        # prettier, including .astro files
```

CI runs `format:check`, `check` and `build`, so run them before pushing.

## Deployment and Service Routing

kaja.tools is deployed to Fly.io. Every service is its own Fly app with its own
public hostname under `kaja.tools` (e.g. `theatre.kaja.tools`, `seating.kaja.tools`);
there is no shared gateway. Service-to-service calls stay on Fly's private
network via `<app>.internal` DNS. See [docs/deployment.md](docs/deployment.md)
for the full map of apps, hostnames, and ports.

**This repository is the website and the demo services, and nothing else.** The
IDE at `demo.kaja.tools` is deployed by
[wham/kaja](https://github.com/wham/kaja) from its own `workspace/`, on every
push to that repository's `main` — so there is no copy of the IDE's
configuration here to keep in step, and nothing here to change when the demo
workspace changes. Don't add one back.

Each service is served at the root of its own hostname (no per-service path
prefix), e.g. the theatre schedule lives at `https://theatre.kaja.tools/shows`.

**A pull request previews the website and nothing else.**
`.github/workflows/preview.yml` deploys `home` to its own Fly app,
`kaja-home-pr-<number>`, and comments the URL on the pull request; closing it
destroys the app. The demo services are stateful, hostname-bound and called by
the IDE, so they ship on merge — don't add previews for them.

### MCP

`concierge` speaks MCP over Streamable HTTP at `/mcp` — JSON-RPC over ordinary
HTTP/1.1 POSTs, so it needs none of the HTTP/2 configuration gRPC does. It
implements the handshake era (`initialize`, `tools/list`, `tools/call`) and
keeps no session, because it keeps nothing between calls.

### Twirp

A Twirp service uses the standard `/twirp` prefix (no custom path prefix), so it
responds at `/twirp/package.Service/Method`, i.e.
`https://quirks.kaja.tools/twirp/...`. Only `quirks` speaks Twirp; the demo
services do not.

### gRPC

gRPC needs HTTP/2 end to end. A gRPC app sets `http_options.h2_backend = true`
and `tls_options.alpn = ["h2"]` in its `fly.toml`, so the Fly edge negotiates
HTTP/2 with clients and forwards HTTP/2 cleartext to the app. gRPC paths follow
the format `/package.Service/Method` (e.g. `/seating.Seating/GetSeatMap`).

**Important:** When adding a new gRPC service, create a Fly app for it with that
`h2_backend` + `alpn = ["h2"]` config and attach its hostname
(`fly certs add <sub>.kaja.tools`).

For `grpcurl`, `seating` registers gRPC reflection so you can list its services
directly; a service without reflection needs its proto files passed with
`-import-path` and `-proto`.

### Testing services

- **OpenAPI**: `curl 'https://theatre.kaja.tools/shows?city=Chicago'`
- **gRPC**: `grpcurl -d '{"showId":"dune-part-two@the-lantern"}' seating.kaja.tools:443 seating.Seating/GetSeatMap`
- **MCP**: `curl -X POST https://concierge.kaja.tools/mcp -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'`
- **Twirp** (quirks only): `curl -X POST https://quirks.kaja.tools/twirp/quirks.v1.Quirks/Sum -H "Content-Type: application/json" -d '{"a":"1","b":"2"}'`
