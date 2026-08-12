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

- Installs a consistent version of `protoc` into the `build/` directory (supports Linux and macOS)
- Installs the required Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-twirp`)
- Regenerates all proto files for the quirks (Twirp) and oven (gRPC) services

Do not use system-installed protoc or manually run protoc commands.

## Demo Services

The public demo is The Kaja Bakery: three services and three protocols —
OpenAPI, gRPC and MCP. Twirp is deliberately not part of it (it is still
supported by kaja itself, and `apps/quirks` still exercises it).

- `apps/bakebook` (OpenAPI) is the book: `GET /cookies` takes no parameters and
  returns everything we bake.
- `apps/oven` (gRPC) owns everything that is actually hot — four racks, live:
  `GetOven`, `ClaimRack`, `StartBake`, and the streaming `WatchOven`.
- `apps/kitchen` (MCP) is the assistant: `scale`, `substitute`, `pair`.

**A cookie's `id` is the only identifier in the demo**: copy it out of
`/cookies` and pass it to the oven as `cookieId` and to the kitchen's tools the
same way. Keep it that way — the point of this shape is that somebody can call
the demo without reading anything first. Adding methods, identifiers, or
required lookups walks it back.

**The three apps are each other's reason to exist.** The book is a catalog and
nothing in it moves; the oven is nothing but movement; the kitchen answers the
questions neither of them can, which is what makes it a set of tools rather
than a fourth REST endpoint. A change that blurs those roles — live state in
the book, a lookup dressed up as a tool — is the thing to say no to.

**The oven runs on demo time**: a recipe's minutes become seconds, so a bake
finishes while you are still looking at it. That is `store.Pace`, a parameter
rather than a constant, because the tests need it faster still.

`apps/oven/internal/crowd` is the rest of the bakery, in-process, claiming racks
through the same store the handlers use — so the oven keeps moving between two
calls and a rack can genuinely be taken from under you. It leaves at least one
rack free on purpose. `CROWD=off` disables it.

`apps/kitchen/internal/mcp` is a hand-rolled MCP server: JSON-RPC 2.0 over one
HTTP endpoint, the tools half of the protocol and nothing else. It is
dependency-free on purpose — a demo service that pulls in an SDK teaches you
about the SDK. Two things it must keep doing: answer an unknown method
(`server/discover` included) with a JSON-RPC error, because that is how a
dual-era client works out which revision it is talking to; and emit schemas
through `mcp.Object`, which preserves key order where a Go map would sort it
and scramble a tool's arguments.

A tool reporting `isError: true` is an answer — "we don't bake that" — and a Go
error from `Call` is the server failing. They are different things and the code
keeps them apart.

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
  src/pages/       one file per route (index, privacy, 404)
  src/layouts/     the page shell — <head>, header, footer
  src/components/  everything reused across pages
  src/styles/      global.css: the Tailwind import and the design tokens
  public/          served verbatim at the root: favicons, logo, /assets/
  Caddyfile        serving rules for the runtime image
```

- `public/assets/` is screenshots and demo videos only.
- **Colours, fonts, radii, shadows and animations are tokens.** They live in the
  `@theme` block in `src/styles/global.css` and are used through the utilities
  Tailwind generates from them (`bg-ink`, `max-w-page`, `shadow-frame`,
  `animate-fade-in`). Don't write a raw hex or a one-off pixel value in a
  component; add or reuse a token.
- The three custom utilities in `global.css` (`brand-gradient`, `text-gradient`,
  `gradient-ring`) exist because they need `background-clip`/`mask-composite`
  tricks that utilities can't express. That is the bar for adding another one.
- Tailwind's preset `text-*` sizes carry their own `line-height`. The design
  inherits `1.6` from the body, so pin it with the slash modifier
  (`text-xl/[1.6]`) whenever a preset size is used.

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
public hostname under `kaja.tools` (e.g. `bakebook.kaja.tools`, `oven.kaja.tools`);
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
prefix), e.g. the book lives at `https://bakebook.kaja.tools/cookies` and the
kitchen's MCP endpoint at `https://kitchen.kaja.tools/mcp`.

**A pull request previews the website and nothing else.**
`.github/workflows/preview.yml` deploys `home` to its own Fly app,
`kaja-home-pr-<number>`, and comments the URL on the pull request; closing it
destroys the app. The demo services are stateful, hostname-bound and called by
the IDE, so they ship on merge — don't add previews for them.

### Twirp

A Twirp service uses the standard `/twirp` prefix (no custom path prefix), so it
responds at `/twirp/package.Service/Method`, i.e.
`https://quirks.kaja.tools/twirp/...`. Only `quirks` speaks Twirp; the demo
services do not.

### gRPC

gRPC needs HTTP/2 end to end. A gRPC app sets `http_options.h2_backend = true`
and `tls_options.alpn = ["h2"]` in its `fly.toml`, so the Fly edge negotiates
HTTP/2 with clients and forwards HTTP/2 cleartext to the app. gRPC paths follow
the format `/package.Service/Method` (e.g. `/oven.Oven/GetOven`).

**Important:** When adding a new gRPC service, create a Fly app for it with that
`h2_backend` + `alpn = ["h2"]` config and attach its hostname
(`fly certs add <sub>.kaja.tools`).

For `grpcurl`, `oven` registers gRPC reflection so you can list its services
directly; a service without reflection needs its proto files passed with
`-import-path` and `-proto`.

### Testing services

- **OpenAPI**: `curl https://bakebook.kaja.tools/cookies`
- **gRPC**: `grpcurl -d '{}' oven.kaja.tools:443 oven.Oven/GetOven`
- **MCP**: `curl -X POST https://kitchen.kaja.tools/mcp -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'`
- **Twirp** (quirks only): `curl -X POST https://quirks.kaja.tools/twirp/quirks.v1.Quirks/Sum -H "Content-Type: application/json" -d '{"a":"1","b":"2"}'`
