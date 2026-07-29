# Deployment (Fly.io)

kaja.tools runs on [Fly.io](https://fly.io). Every service is its own Fly app
with its own public hostname under `kaja.tools`. There is no shared gateway —
each app terminates TLS at the Fly edge for its own subdomain. This replaces the
single-domain nginx-ingress that ran on Azure Kubernetes.

## Apps

| Fly app             | Source           | Public hostname            | Notes                                   |
| ------------------- | ---------------- | -------------------------- | --------------------------------------- |
| `kaja-home`         | `apps/home`      | `kaja.tools`, `www`        | static marketing site                   |
| `kaja-demo`         | `apps/kaja`      | `demo.kaja.tools`          | the IDE                                 |
| `kaja-theatre`      | `apps/theatre`   | `theatre.kaja.tools`       | OpenAPI (e.g. `/openapi.yaml`, `/events`) |
| `kaja-boxoffice`    | `apps/boxoffice` | `boxoffice.kaja.tools`     | Twirp under `/twirp`                    |
| `kaja-seating`      | `apps/seating`   | `seating.kaja.tools`       | gRPC over TLS on :443                    |
| `kaja-quirks-grpc`  | `apps/quirks`    | `grpc-quirks.kaja.tools`   | gRPC over TLS on :443                    |
| `kaja-quirks-twirp` | `apps/quirks`    | `twirp-quirks.kaja.tools`  | Twirp under `/twirp`                    |

Each service is served at the root of its own hostname (e.g. the theatre spec
is at `https://theatre.kaja.tools/openapi.yaml`, Twirp uses the standard
`/twirp/...` prefix). `apps/kaja/kaja.json` and the theatre OpenAPI `servers`
URL point at these hostnames.

### East-west (service-to-service) traffic

`boxoffice` calls `theatre` (HTTP) and `seating` (gRPC), and `seating` calls
`theatre` (HTTP). Those calls stay on Fly's private network via `<app>.internal`
DNS (`kaja-theatre.internal:41530`, `kaja-seating.internal:50053`) — they never
leave the org, so only public browser/IDE traffic goes through the edge.

### gRPC

gRPC needs HTTP/2 end to end. The gRPC apps (`kaja-seating`, `kaja-quirks-grpc`)
set `http_options.h2_backend = true` and `tls_options.alpn = ["h2"]` in their
`fly.toml`, so the Fly edge negotiates HTTP/2 with clients and forwards HTTP/2
cleartext to the app. Clients dial `seating.kaja.tools:443` (see `kaja.json`).

## First-time setup

```bash
# 1. Create the apps, allocate IPs, and attach each hostname's certificate.
scripts/fly-setup

# 2. Deploy everything.
scripts/deploy

# 3. DNS: point each hostname at its app.
#      - Apex kaja.tools: A/AAAA records to kaja-home's IPs (fly ips list --app kaja-home)
#      - Subdomains: CNAME <sub>.kaja.tools -> <app>.fly.dev
#    Verify with: fly certs check <hostname> --app <app>

# 4. Create a deploy token and add it to the repo as the FLY_API_TOKEN secret
#    so GitHub Actions can deploy on push to main.
fly tokens create deploy
```

## Ongoing deploys

Pushing to `main` deploys every app via `.github/workflows/main.yml`
(`FLY_API_TOKEN` repo secret required).

To deploy manually:

```bash
scripts/deploy            # all apps
scripts/deploy theatre    # a single app
```

## Notes

- The kaja demo IDE no longer needs a persistent volume. Its workspace config
  (`kaja.json`) and demo proto files are baked into the image at build time
  (`apps/kaja/Dockerfile`), so kaja builds from the repo root:
  `fly deploy . --config apps/kaja/fly.toml`.
- `primary_region` is set to `sjc` in every `fly.toml`. Change it to match the
  region you host in; keeping all apps in one region keeps the private-network
  (east-west) hops fast.
