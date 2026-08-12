# Deployment (Fly.io)

kaja.tools runs on [Fly.io](https://fly.io). Every service is its own Fly app
with its own public hostname under `kaja.tools`. There is no shared gateway —
each app terminates TLS at the Fly edge for its own subdomain. This replaces the
single-domain nginx-ingress that ran on Azure Kubernetes.

## Apps

| Fly app         | Source          | Public hostname       | Notes                                       |
| --------------- | --------------- | --------------------- | ------------------------------------------- |
| `kaja-home`     | `home`          | `kaja.tools`, `www`   | the website — Astro, served by Caddy        |
| `kaja-bakebook` | `apps/bakebook` | `bakebook.kaja.tools` | OpenAPI (e.g. `/openapi.yaml`, `/cookies`)  |
| `kaja-oven`     | `apps/oven`     | `oven.kaja.tools`     | gRPC over TLS on :443                       |
| `kaja-kitchen`  | `apps/kitchen`  | `kitchen.kaja.tools`  | MCP Streamable HTTP at `/mcp`               |
| `kaja-quirks`   | `apps/quirks`   | `quirks.kaja.tools`   | Twirp under `/twirp`                        |

`bakebook`, `oven` and `kitchen` are the public demo — one catalog, one live
service, one toolbelt, three protocols. `quirks` is the Twirp protocol testbed.
The website sits at the repository root rather than under `apps/`, because
`apps/` is the demo services and the website is not one of them.

### The website's image

`home/Dockerfile` is two stages: Node runs `astro build`, then the output is
copied into a `caddy:alpine` image and Node is discarded. Nothing is rendered
per request, so the runtime is a file server and a directory — no JavaScript
runtime ships to production. `home/Caddyfile` holds the serving rules:
`auto_https off` (Fly terminates TLS at the edge and forwards plain HTTP to
`internal_port`), long immutable caching for the fingerprinted assets under
`/_astro`, revalidation for everything else, and 301s from the old `.html`
URLs to the directory form Astro emits.

**The IDE at `demo.kaja.tools` is not deployed from here.** The `kaja-demo` Fly
app lives in the same organization, but it is built and deployed by
[wham/kaja](https://github.com/wham/kaja) on every push to its `main`, from
that repository's own `workspace/` — the same image its pull-request previews
run. This repository owns the demo *services* the IDE calls; it no longer holds
a second copy of the IDE's configuration to keep in step.

Each service is served at the root of its own hostname (e.g. the bakebook spec
is at `https://bakebook.kaja.tools/openapi.yaml`, the kitchen's MCP endpoint at
`https://kitchen.kaja.tools/mcp`, and Twirp uses the standard `/twirp/...`
prefix). The bakebook OpenAPI `servers` URL — and the IDE's
`workspace/kaja.json` over in wham/kaja — point at these hostnames.

### East-west (service-to-service) traffic

`oven` and `kitchen` both call `bakebook` (HTTP) to look up which cookies exist
and how they bake. Those calls stay on Fly's private network via
`<app>.internal` DNS (`kaja-bakebook.internal:41530`) — they never leave the
org, so only public browser/IDE traffic goes through the edge.

### gRPC

gRPC needs HTTP/2 end to end. The gRPC app (`kaja-oven`) sets
`http_options.h2_backend = true` and `tls_options.alpn = ["h2"]` in its
`fly.toml`, so the Fly edge negotiates HTTP/2 with clients and forwards HTTP/2
cleartext to the app. Clients dial `oven.kaja.tools:443`.

The MCP app needs none of it: MCP Streamable HTTP is ordinary HTTP POST, so
`kaja-kitchen` is a plain `http_service`.

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

# 4. Create a token and add it to the repo as the FLY_API_TOKEN secret so
#    GitHub Actions can deploy on push to main. It has to be an org token
#    rather than the app-scoped deploy one: pull request previews create and
#    destroy an app of their own, which a deploy token may not do.
fly tokens create org personal
```

## Ongoing deploys

`.github/workflows/deploy.yml` holds the app matrix and does the deploying. It
runs two ways, both needing the `FLY_API_TOKEN` repo secret:

- **On push to `main`** — `.github/workflows/main.yml` builds and vets every
  app, then calls it to deploy all of them. Pull requests only build.
- **By hand** — Actions → **deploy** → **Run workflow**, then pick an app (or
  `all`). This skips the build gate, so it also serves as a redeploy button
  when nothing changed. Note the ref picker deploys whatever branch you choose;
  leave it on `main` unless you mean otherwise.

There was a third, a daily `schedule`, and it is gone with the demo IDE: it
existed only so a newly published `kajatools/kaja:latest` reached `kaja-demo`
without a commit here to deploy against. Every app left is built from this
repository's own source, so there is nothing to ship on a day nothing changed.

Push deploys are a reusable-workflow call, so they appear as
`deploy / Deploy <app>` jobs inside the **main** run, not as runs under
**deploy** in the Actions sidebar — that list shows only manual runs.

Apps deploy in parallel with `fail-fast: false`, so one bad app doesn't block
the others, and a per-app concurrency group keeps two deploys of the same app
from overlapping.

To deploy from your own machine instead:

```bash
scripts/deploy            # all apps
scripts/deploy oven       # a single app
```

Adding or renaming an app means updating the `APPS` list in `deploy.yml` (and
its `workflow_dispatch` options), `scripts/deploy`, and `scripts/fly-setup` —
then re-running `scripts/fly-setup`, which is idempotent. A Fly app that has
never been created fails the deploy with `app not found`; the workflow calls
out which app it was.

## Pull request previews

Every pull request opened from a branch in this repository gets its own copy of
the **website** running on Fly, at `https://kaja-home-pr-<number>.fly.dev`.
`.github/workflows/preview.yml` deploys `home` with
`home/fly.preview.toml` under an app named after the pull request, and a
sticky comment on the pull request carries the URL. Every push rebuilds it;
closing the pull request destroys the app and removes the comment.

Only the website is previewed. The demo services (`bakebook`, `oven`,
`kitchen`, `quirks`) are stateful, hostname-bound and called by the IDE, so a
change to one ships when it merges rather than to a throwaway app nothing
points at.

A preview app runs no machine until someone opens its URL
(`min_machines_running = 0`), so an idle preview costs nothing. A pull request
from a fork is skipped: it has no access to the token.

## Notes

- The oven runs the rest of the bakery in-process (`CROWD=off` to disable), so
  the racks keep moving without a second app driving it. It also runs on demo
  time: a recipe's minutes become seconds, so a bake finishes while you watch.
- `primary_region` is set to `sjc` in every `fly.toml`. Change it to match the
  region you host in; keeping all apps in one region keeps the private-network
  (east-west) hops fast.
