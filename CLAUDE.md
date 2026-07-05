# Agent Guidelines

## Proto Files

To regenerate proto files after modifying `.proto` definitions, run:
```bash
./scripts/protoc
```

This script:
- Installs a consistent version of `protoc` into the `build/` directory (supports Linux and macOS)
- Installs the required Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-twirp`)
- Regenerates all proto files for the quirks, seating, and boxoffice services

Do not use system-installed protoc or manually run protoc commands.

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

Each app keeps its original internal path prefix, so a service reachable at
`theatre.kaja.tools` still serves its routes under `/theatre`.

### Twirp path prefix
A Twirp service registers under a path prefix that its public URL includes:
```go
twirp.WithServerPathPrefix("/boxoffice/twirp")
```
This makes the service respond at `/boxoffice/twirp/ServiceName/Method`, i.e.
`https://boxoffice.kaja.tools/boxoffice/twirp/...`. `apps/kaja/kaja.json` points
kaja at the base URL (`https://boxoffice.kaja.tools/boxoffice`).

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
- **Twirp**: `curl -X POST https://boxoffice.kaja.tools/boxoffice/twirp/boxoffice.BoxOffice/GetOrder -H "Content-Type: application/json" -d '{}'`
- **gRPC**: `grpcurl -import-path apps/seating/proto -proto seating.proto seating.kaja.tools:443 seating.Seating/GetSeatMap`
