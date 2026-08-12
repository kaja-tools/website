# Agent Guidelines

## Pull Requests

Keep PR descriptions short. Aim for a one-line summary plus a brief bullet list
of the concrete changes. Skip lengthy background, root-cause narratives, and
generated-by boilerplate — link to related PRs/issues instead of restating them.

## Proto Files

To regenerate proto files after modifying `.proto` definitions, run:

```bash
./scripts/protoc
```

This script:

- Installs a consistent version of `protoc` into the `build/` directory (supports Linux and macOS)
- Installs the required Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-twirp`)
- Regenerates all proto files for the quirks (Twirp) and seating (gRPC) services

Do not use system-installed protoc or manually run protoc commands.

## Demo Services

The public demo is two services and two protocols — gRPC and OpenAPI. Twirp
is deliberately not part of the demo (it is still supported by kaja itself,
and `apps/quirks` still exercises it).

- `apps/theatre` (OpenAPI) is the programme: `GET /shows` takes no parameters
  and returns the whole repertoire.
- `apps/seating` (gRPC) owns live seat state: `GetSeatMap`, `HoldSeats`,
  `ConfirmSeats`, and the streaming `WatchSeats`.

A show's `id` is the only identifier in the demo: copy it out of `/shows` and
pass it to seating as `showId`. Keep it that way — the point of this shape is
that somebody can call the demo without reading anything first. Adding
methods, identifiers, or required lookups walks it back.

`apps/seating/internal/crowd` simulates other customers in-process so the seat
map is always moving. `CROWD=off` disables it.

`apps/quirks` is not part of the demo — it is the Twirp protocol testbed for
edge cases (odd names, deep nesting, panics, streaming RPCs that Twirp renders
as unary).

## Home Page Static Files

Structure for `apps/home/static/`:

- Root level: `index.html`, `styles.css`, `script.js`, `favicon.ico`, `favicon.svg`, `logo.svg`
- `/assets/`: Screenshots and demo videos only

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
prefix), e.g. the theatre programme lives at `https://theatre.kaja.tools/shows`.

**A pull request previews the website and nothing else.**
`.github/workflows/preview.yml` deploys `apps/home` to its own Fly app,
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
the format `/package.Service/Method` (e.g. `/seating.Seating/GetSeatMap`).

**Important:** When adding a new gRPC service, create a Fly app for it with that
`h2_backend` + `alpn = ["h2"]` config and attach its hostname
(`fly certs add <sub>.kaja.tools`).

For `grpcurl`, `seating` registers gRPC reflection so you can list its services
directly; a service without reflection needs its proto files passed with
`-import-path` and `-proto`.

### Testing services

- **OpenAPI**: `curl https://theatre.kaja.tools/shows`
- **gRPC**: `grpcurl -d '{"showId":"neon-meridian"}' seating.kaja.tools:443 seating.Seating/GetSeatMap`
- **Twirp** (quirks only): `curl -X POST https://quirks.kaja.tools/twirp/quirks.v1.Quirks/Sum -H "Content-Type: application/json" -d '{"a":"1","b":"2"}'`
