# Agent Guidelines

## Proto Files

To regenerate proto files after modifying `.proto` definitions, run:
```bash
./scripts/protoc
```

This script:
- Installs a consistent version of `protoc` into the `build/` directory (supports Linux and macOS)
- Installs the required Go plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-twirp`)
- Regenerates all proto files for teams, users, and quirks services

Do not use system-installed protoc or manually run protoc commands.

## Home Page Static Files

Structure for `apps/home/static/`:

- Root level: `index.html`, `styles.css`, `script.js`, `favicon.ico`, `favicon.svg`, `logo.svg`
- `/assets/`: Screenshots and demo videos only

## Ingress and Service Routing

### Twirp behind ingress
When a Twirp service runs behind an ingress with path-based routing (e.g., `/users`), configure the server with a matching path prefix:
```go
twirp.WithServerPathPrefix("/users/twirp")
```
This makes the service respond at `/users/twirp/ServiceName/Method`.

### gRPC with nginx-ingress
gRPC requires a **separate ingress resource** with the annotation:
```yaml
nginx.ingress.kubernetes.io/backend-protocol: "GRPC"
```
This annotation applies to all backends in an ingress, so HTTP and gRPC services cannot share the same ingress.

gRPC paths follow the format `/package.Service/Method` (e.g., `/teams.Teams/GetAllTeams`).

### Testing services locally
- **Twirp**: `curl -X POST http://localhost/users/twirp/Users/GetAllUsers -H "Content-Type: application/json" -d '{}'`
- **gRPC**: `grpcurl -plaintext -import-path apps/teams/proto -proto teams.proto localhost:80 teams.Teams/GetAllTeams`

- Run against the k8s cluster on localhost
