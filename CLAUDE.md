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
- Regenerates all proto files for the quirks, seating, and boxoffice services

Do not use system-installed protoc or manually run protoc commands.

## Home Page Static Files

Structure for `apps/home/static/`:

- Root level: `index.html`, `styles.css`, `script.js`, `favicon.ico`, `favicon.svg`, `logo.svg`
- `/assets/`: Screenshots and demo videos only

## Ingress and Service Routing

### Twirp behind ingress
When a Twirp service runs behind an ingress with path-based routing (e.g., `/boxoffice`), configure the server with a matching path prefix:
```go
twirp.WithServerPathPrefix("/boxoffice/twirp")
```
This makes the service respond at `/boxoffice/twirp/ServiceName/Method`.

### gRPC with nginx-ingress
gRPC requires a **separate ingress resource** with the annotation:
```yaml
nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
```
This annotation applies to all backends in an ingress, so HTTP and gRPC services cannot share the same ingress.

gRPC paths follow the format `/package.Service/Method` (e.g., `/seating.Seating/GetSeatMap`).

**Important:** When adding a new gRPC service, you must add paths to `k8s/base/ingress-grpc.yaml` for each service in the proto files (e.g., `/mypackage.MyService/`). Remember to use the full `package.Service` format.

**gRPC Reflection limitation:** Reflection paths (`/grpc.reflection.v1.ServerReflection/` and `/grpc.reflection.v1alpha.ServerReflection/`) can only route to one backend per ingress. Currently they route to `seating-service`. For `grpcurl` on other services, use the `-import-path` flag with proto files:
```bash
grpcurl -plaintext -import-path apps/quirks/proto -proto v1/quirks.proto localhost:80 quirks.v1.Quirks/Sum
```

### Testing services locally
- **Twirp**: `curl -X POST http://localhost/boxoffice/twirp/BoxOffice/GetOrder -H "Content-Type: application/json" -d '{}'`
- **gRPC**: `grpcurl -plaintext -import-path apps/seating/proto -proto seating.proto localhost:80 seating.Seating/GetSeatMap`

- Run against the k8s cluster on localhost
