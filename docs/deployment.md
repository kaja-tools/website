# Deployment (Fly.io)

kaja.tools runs on [Fly.io](https://fly.io). Every service is its own Fly app
with its own public hostname under `kaja.tools`. There is no shared gateway —
each app terminates TLS at the Fly edge for its own subdomain. This replaces the
single-domain nginx-ingress that ran on Azure Kubernetes.

## Apps

| Fly app        | Source         | Public hostname      | Notes                                    |
| -------------- | -------------- | -------------------- | ---------------------------------------- |
| `kaja-home`    | `apps/home`    | `kaja.tools`, `www`  | static marketing site                    |
| `kaja-demo`    | `apps/kaja`    | `demo.kaja.tools`    | the IDE                                  |
| `kaja-theatre` | `apps/theatre` | `theatre.kaja.tools` | OpenAPI (e.g. `/openapi.yaml`, `/shows`) |
| `kaja-seating` | `apps/seating` | `seating.kaja.tools` | gRPC over TLS on :443                    |
| `kaja-quirks`  | `apps/quirks`  | `quirks.kaja.tools`  | Twirp under `/twirp`                     |

`theatre` and `seating` are the public demo; `quirks` is the Twirp protocol
testbed.

Each service is served at the root of its own hostname (e.g. the theatre spec
is at `https://theatre.kaja.tools/openapi.yaml`, Twirp uses the standard
`/twirp/...` prefix). `apps/kaja/kaja.json` and the theatre OpenAPI `servers`
URL point at these hostnames.

### East-west (service-to-service) traffic

`seating` calls `theatre` (HTTP) to look up which shows exist and what they
cost. Those calls stay on Fly's private network via `<app>.internal` DNS
(`kaja-theatre.internal:41530`) — they never leave the org, so only public
browser/IDE traffic goes through the edge.

### gRPC

gRPC needs HTTP/2 end to end. The gRPC app (`kaja-seating`) sets
`http_options.h2_backend = true` and `tls_options.alpn = ["h2"]` in its
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

`.github/workflows/deploy.yml` holds the app matrix and does the deploying. It
runs three ways, all needing the `FLY_API_TOKEN` repo secret:

- **On push to `main`** — `.github/workflows/main.yml` builds and vets every
  app, then calls it to deploy all of them. Pull requests only build.
- **Daily at 15:00 UTC** — a `schedule` trigger deploys every app, so a new
  `kajatools/kaja:latest` reaches `apps/kaja` without a commit here to deploy
  against.
- **By hand** — Actions → **deploy** → **Run workflow**, then pick an app (or
  `all`). This skips the build gate, so it also serves as a redeploy button
  when nothing changed. Note the ref picker deploys whatever branch you choose;
  leave it on `main` unless you mean otherwise.

Push deploys are a reusable-workflow call, so they appear as
`deploy / Deploy <app>` jobs inside the **main** run, not as runs under
**deploy** in the Actions sidebar — that list shows only scheduled and manual
runs. GitHub also disables scheduled workflows in repositories with no activity
for 60 days.

Apps deploy in parallel with `fail-fast: false`, so one bad app doesn't block
the others, and a per-app concurrency group keeps two deploys of the same app
from overlapping.

To deploy from your own machine instead:

```bash
scripts/deploy            # all apps
scripts/deploy theatre    # a single app
```

Adding or renaming an app means updating the `APPS` list in `deploy.yml` (and
its `workflow_dispatch` options), `scripts/deploy`, and `scripts/fly-setup` —
then re-running `scripts/fly-setup`, which is idempotent. A Fly app that has
never been created fails the deploy with `app not found`; the workflow calls
out which app it was.

## Notes

- The seating service runs the simulated crowd in-process (`CROWD=off` to
  disable), so the seat map keeps moving without a second app driving it.
- The kaja demo IDE no longer needs a persistent volume. Its workspace config
  (`kaja.json`) and demo proto files are baked into the image at build time
  (`apps/kaja/Dockerfile`), so kaja builds from the repo root:
  `fly deploy . --config apps/kaja/fly.toml`.
- `primary_region` is set to `sjc` in every `fly.toml`. Change it to match the
  region you host in; keeping all apps in one region keeps the private-network
  (east-west) hops fast.
