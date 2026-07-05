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

kaja.tools is deployed to Fly.io. Every service is its own Fly app, and a
single public Caddy **gateway** (`apps/gateway`) owns `kaja.tools`, terminates
TLS, and path-routes to the private backend apps over Fly's private network
(`<app>.internal`). See [docs/deployment.md](docs/deployment.md) for the full
map of apps, ports, and routed paths.

### Twirp behind the gateway
When a Twirp service is routed by path (e.g., `/boxoffice`), configure the
server with a matching path prefix:
```go
twirp.WithServerPathPrefix("/boxoffice/twirp")
```
This makes the service respond at `/boxoffice/twirp/ServiceName/Method`, and the
gateway forwards `/boxoffice/*` to it.

### gRPC through the gateway
gRPC needs HTTP/2 end to end. The gateway advertises `h2` via ALPN and forwards
HTTP/2 cleartext (h2c) to the backends (`reverse_proxy h2c://...`). gRPC paths
follow the format `/package.Service/Method` (e.g., `/seating.Seating/ReserveSeat`).

**Important:** When adding a new gRPC service, add a route for each of its
`/package.Service/*` paths to `apps/gateway/Caddyfile`, pointing at the new
app's `<app>.internal:<port>` with the `h2c://` scheme. Use the full
`package.Service` format.

**gRPC Reflection limitation:** Reflection paths
(`/grpc.reflection.v1.ServerReflection/` and
`/grpc.reflection.v1alpha.ServerReflection/`) can only route to one backend.
They currently route to `kaja-seating`. For `grpcurl` on other services, use
the `-import-path` flag with proto files:
```bash
grpcurl -import-path apps/quirks/proto -proto v1/quirks.proto kaja.tools:443 quirks.v1.Quirks/Sum
```

### Testing services
- **Twirp**: `curl -X POST https://kaja.tools/boxoffice/twirp/boxoffice.BoxOffice/GetOrder -H "Content-Type: application/json" -d '{}'`
- **gRPC**: `grpcurl -import-path apps/seating/proto -proto seating.proto kaja.tools:443 seating.Seating/GetSeatMap`
