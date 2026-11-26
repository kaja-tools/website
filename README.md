# kaja.tools

This repo is the source for the [kaja.tools](https://kaja.tools) website.

## Layout

- `home/` — the website. [Astro](https://astro.build) with Tailwind CSS v4,
  built to static files and served by Caddy.
- `apps/` — the demo services, in Go. `theatre` (OpenAPI) and `seating` (gRPC)
  are the public demo; `quirks` is the Twirp protocol testbed.

## Development

The website:

```
cd home
npm install
npm run dev       # localhost:4321
npm run build     # static build into home/dist
npm run check     # TypeScript and template diagnostics
npm run format    # prettier, including .astro files
```

The services:

```
# Re-generate gRPC and Twirp code for all apps. Commit when done.
scripts/protoc
```

## Production

kaja.tools is deployed to [Fly.io](https://fly.io). Pushing to `main` deploys
every app via GitHub Actions. To deploy manually:

```
# Deploy everything to production
scripts/deploy
```

See [docs/deployment.md](docs/deployment.md) for the full architecture and
first-time setup.
