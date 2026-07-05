# Deployment (Fly.io)

kaja.tools runs on [Fly.io](https://fly.io). Every service is its own Fly app.
A single public **gateway** app (Caddy) owns the `kaja.tools` domain and TLS
certificate and reverse-proxies each path to the matching backend app over
Fly's private network. This replaces the old nginx-ingress on Azure
Kubernetes without changing any public URL or application code.

## Apps

| Fly app             | Source              | Visibility | Internal address                 | Routed paths                              |
| ------------------- | ------------------- | ---------- | -------------------------------- | ----------------------------------------- |
| `kaja-gateway`      | `apps/gateway`      | public     | —                                | owns `kaja.tools`, terminates TLS         |
| `kaja-home`         | `apps/home`         | private    | `kaja-home.internal:8080`        | `/` (everything not matched below)        |
| `kaja-demo`         | `apps/kaja`         | private    | `kaja-demo.internal:41520`       | `/demo`                                   |
| `kaja-theatre`      | `apps/theatre`      | private    | `kaja-theatre.internal:41530`    | `/theatre`                                |
| `kaja-boxoffice`    | `apps/boxoffice`    | private    | `kaja-boxoffice.internal:41531`  | `/boxoffice` (Twirp)                      |
| `kaja-seating`      | `apps/seating`      | private    | `kaja-seating.internal:50053`    | `/seating.Seating/*` (gRPC)               |
| `kaja-quirks-grpc`  | `apps/quirks`       | private    | `kaja-quirks-grpc.internal:50053`| `/quirks.v1.*`, `/quirks.v2.*` (gRPC)     |
| `kaja-quirks-twirp` | `apps/quirks`       | private    | `kaja-quirks-twirp.internal:41523`| `/twirp-quirks` (Twirp)                  |

Only `kaja-gateway` has public IPs. Backends are reachable only over Fly's 6PN
private network via their `<app>.internal` DNS names, so the gateway is the
single ingress point — mirroring the old `ClusterIP` + ingress model.

### gRPC through the gateway

gRPC needs HTTP/2 end to end. The gateway's `[http_service]` sets
`h2_backend = true` and advertises `alpn = ["h2", "http/1.1"]`, so the Fly edge
forwards HTTP/2 cleartext (h2c) to Caddy, which proxies to the gRPC backends
over h2c (`reverse_proxy h2c://...`). gRPC clients dial `kaja.tools:443`
(see `apps/kaja/kaja.json`).

## First-time setup

```bash
# 1. Create the apps and allocate the gateway's public IPs.
scripts/fly-setup

# 2. Deploy everything.
scripts/deploy

# 3. Provide the kaja IDE's AI key (kept as a Fly secret, never committed).
fly secrets set AI_API_KEY=... --app kaja-demo

# 4. Attach the domain and point DNS at the gateway.
fly certs add kaja.tools --app kaja-gateway
fly ips list --app kaja-gateway   # create A/AAAA records for kaja.tools

# 5. Create a deploy token and add it to the repo as the FLY_API_TOKEN secret
#    so GitHub Actions can deploy on push to main.
fly tokens create deploy
```

## Ongoing deploys

Pushing to `main` deploys every app via `.github/workflows/main.yml`
(`FLY_API_TOKEN` repo secret required).

To deploy manually:

```bash
scripts/deploy            # all apps
scripts/deploy gateway    # a single app
```

## Notes

- The kaja demo IDE no longer needs a persistent volume. Its workspace config
  (`kaja.json`) and demo proto files are baked into the image at build time
  (`apps/kaja/Dockerfile`), so kaja builds from the repo root:
  `fly deploy . --config apps/kaja/fly.toml`.
- `primary_region` is set to `iad` in every `fly.toml`. Change it to match the
  region you host in; keeping all apps in one region keeps the gateway ⇄ backend
  private-network hops fast.
