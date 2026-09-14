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
  src/pages/       one file per route (index, docs/[platform], privacy, 404)
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
- **The hero's drawing is the app's own map** (`ui/src/McpMap.tsx`), ported
  stroke for stroke: an agent, Kaja drawn as its own canvas, the four
  protocols. It is what says "you and your agents" before a word of the page is
  read. Three weights and no colour — `wire` for the wires and the canvas blocks,
  `muted-foreground` for the node frames and arrows, `foreground` for names and
  marks. **There is no screenshot in it**: the app itself is the next thing on
  the page, and a picture of the window beside the drawing says the same thing
  twice. The site's own layer is the entrance, and the one thing not ported
  verbatim is the width of the app cards, because the site's body face is wider
  than the app's. Below `md` the drawing is too small to read, so the same five
  things are a plain stack of cards.
- **The eight screenshots are `data/shots.ts`, and every picture of the whole
  window is a crop of one** — the home page's, and the docs figures taken
  before the docs took their own (`snaps`, below). The shots are the window at 2880x1800, zoomed to 125%, which is what
  `scripts/demo` photographs in [wham/kaja](https://github.com/wham/kaja); a
  fresh set replaces the files under the same names and every crop still
  lands. The zoom is what makes a crop legible at the size it is read, so a
  set taken at 100% is a set every crop is a quarter smaller in. A crop is four fractions drawn as a background by `cropStyle` rather
  than an image file of its own, so a screen contributes as many close-ups as
  it has things to say and nothing under `public/assets/` is a cut-out to
  re-cut.
- **The home page is a ladder, and every rung is a section** (`Section.astro`,
  one heading and one line under it). The order is what a visitor needs and
  not what the app is proud of: the hero, the flow through the window, what an
  agent does inside it, what a call leaves behind, using it by hand, the way it
  differs from a client built on saved collections, the protocols and the
  agent, what the window does beyond a request, the source, and the way in
  again. The hero asks for the demo and offers the download second, because
  nothing has to be installed to see Kaja work; the header's one button is the
  demo for the same reason. The page returns to that same ask at the foot,
  unchanged — the last section is the hero's two buttons and no new claim.
- **The ladder has three rungs, and `level` is which one a section is on.**
  `lead` is the product story — the hero, the flow, the agent, the record —
  and it is set at full size. `support` is what makes Kaja different from
  another client, a size down. `minor` is the capabilities around the request,
  a size down again. `Poster`'s `tone` is the same decision for the statements
  a section brings: `lead` for the record, `quiet` for everything under it,
  which is smaller type and less air between the items. The page gets quieter
  as it goes, so a visitor who reads to the end has met one story rather than
  a feature catalogue. Add a section at the level its argument belongs to
  rather than at the size that would get it noticed.
- **A heading counts nothing that can change.** "Your APIs. Your agent." rather
  than the number of protocols and agents in the list, so the page does not go
  stale the week an agent is added. The same goes for a statement in
  `poster.ts`: it is the thing and what it is good for, in one sentence,
  because the crop beside it already shows the control.
- **The flow is explained once, as three steps** (`Flow.astro`, from `steps` in
  `data/home.ts`): connect, run, inspect. Each step is the copy and the piece
  of the window it happens in, stacked down the page rather than set in a row,
  so the copy sits above its picture on a phone and the order a visitor reads
  is the order the steps happen in.
- **The agent section is one task end to end** (`AgentTask.astro`): a sentence
  somebody typed, the script the agent wrote, and the calls landing in the
  console. Three beats down one rail, because the claim is the order.
- **The statements set against a crop are `Poster.astro`, and they come in
  groups** — `manual`, `record`, `agents` and `features` in `data/poster.ts`,
  each brought by the section that makes its argument rather than run together
  at the foot of the page. `app-hero.png` is shown whole under the hero and
  carries several of the crops for nothing. Each statement has a red line to a
  box around the thing it names; the line and the box are geometry
  `Motion.astro` measures, because both halves drift as the page scrolls, so
  without script the page is the statements and the crops and no ink. A new
  statement is an entry in one of those groups, not markup.
- **A crop is read at about two thirds of the width it was taken at**, so a
  region wider than about half the shot is a picture of text nobody can read.
  Keep the flow's and the agent section's crops under that, and remember that
  **a crop wider than about 8:1 is a band on a phone**, 40px tall and legible
  only as texture — spend those on strips that read as one line, a call row or
  a tile strip, rather than on anything with rows to read.
- **The docs are one source file and two pages** (`src/pages/docs/[platform].astro`),
  and they are the core flow and nothing else: install, connect an app, run a
  script, keep a secret, run a script from outside, point an agent at it.
  **The two builds have two audiences**, and a section is written for each
  where they differ: the desktop does everything in the window and never
  shows you `kaja.json`, so its blocks name the buttons; the container is set
  up from the files you mount, so its blocks show the file. What is true of
  both is written once, in the one file both pages are built from, because a
  second file would fork every paragraph to keep two copies of four
  differences.
- **Which build you are reading for is the URL** — `/docs/desktop/` and
  `/docs/docker/`, with `/docs` redirecting to Desktop, which is the one a
  reader gets if they have not said. So a docs link says which build it was
  written for and lands a reader on it, and the two pages are indexed apart,
  each with its own title and description from `builds` in `data/docs.ts`.
  `<Platform only="desktop">` / `only="docker"` are the blocks, and a block
  that is not this page's build is never rendered rather than hidden — there
  is nothing clientside to pick, and no `noscript` rule to need. Switching
  build is a navigation, so `DocsMotion` puts the section being read on the
  other page's link: you come out where you went in.
- `data/docs.ts` is the section list and the two builds — the nav and the
  scroll spy read the sections, so
  a section is added there and its body written in the page under the same
  `id`. **A screenshot sits in the copy, under the thing it shows**
  (`Figure.astro`, from the `figures` table in `docs/[platform].astro`), and inside one
  platform's block where the two builds differ; a section with nothing worth
  showing has none rather than borrowing a picture of something else. A crop
  is read at the width of the column, which is about a quarter of the shot's,
  so a region wider than about two thirds of the shot is a picture nobody can
  read the words in; one taller than it is wide is capped by `--size-figure`
  and stands beside the copy instead of filling it.
- **The `kaja.*` verbs in the Scripts section are `data/runtime.ts`, and they
  are an index, not a reference.** One sentence each, a snippet only where the
  sentence cannot say the shape, and one link at the foot of the list to
  `ui/src/kajaModule.ts` in [wham/kaja](https://github.com/wham/kaja) — the
  declaration the editor completes against and an agent reads through
  `describe_type "kaja"`, which the compiler checks against the runtime. Never
  retell a verb's API here; that copy goes stale. A snippet is a real script
  out of kaja's `workspace/scripts/`, which is what lets the figure under it be
  that snippet's own output, and **a verb that draws on the canvas carries a
  figure** — a sentence says what a verb is for, and only the picture says what
  it leaves on screen. **The docs take their own**: a **shot** is the whole
  macOS window and belongs to the home page, while a **snap** (`snaps` in
  `data/shots.ts`) is one part of the canvas photographed from kaja's web build
  at the width this column reads it at, so it is drawn whole and its type is
  the size it was on screen. `.claude/skills/docs-figure` is the whole flow —
  running the build, driving the run, and the crop arithmetic the shots still
  need.
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
