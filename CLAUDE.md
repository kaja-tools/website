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
- Installs the required Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)
- Regenerates all proto files for the quirks and seating services

Do not use system-installed protoc or manually run protoc commands.

## Demo Services

The public demo is two services and two protocols — gRPC and OpenAPI. Twirp
is deliberately not part of it.

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

Each service is served at the root of its own hostname (no per-service path
prefix), e.g. the theatre programme lives at `https://theatre.kaja.tools/shows`.

### gRPC

gRPC needs HTTP/2 end to end. A gRPC app sets `http_options.h2_backend = true`
and `tls_options.alpn = ["h2"]` in its `fly.toml`, so the Fly edge negotiates
HTTP/2 with clients and forwards HTTP/2 cleartext to the app. gRPC paths follow
the format `/package.Service/Method` (e.g. `/seating.Seating/GetSeatMap`).

**Important:** When adding a new gRPC service, create a Fly app for it with that
`h2_backend` + `alpn = ["h2"]` config and attach its hostname
(`fly certs add <sub>.kaja.tools`).

For `grpcurl`, `seating` registers gRPC reflection so you can list its services
directly; for services without reflection, pass the proto files with
`-import-path`:

```bash
grpcurl -import-path apps/quirks/proto -proto v1/quirks.proto grpc-quirks.kaja.tools:443 quirks.v1.Quirks/Sum
```

### Testing services

- **OpenAPI**: `curl https://theatre.kaja.tools/shows`
- **gRPC**: `grpcurl -d '{"showId":"neon-meridian"}' seating.kaja.tools:443 seating.Seating/GetSeatMap`
