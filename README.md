# kaja.tools

This repo is the source for the [kaja.tools](https://kaja.tools) website.

## Development

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
