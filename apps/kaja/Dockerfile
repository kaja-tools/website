# Thin wrapper around the published kaja IDE image that bakes in the workspace
# configuration (kaja.json) and the demo proto files. Previously these were
# copied into a shared Kubernetes volume at deploy time; on Fly they are baked
# into the image instead, so no persistent volume is required.
#
# Build context is the repository root (the proto files live in sibling apps):
#   fly deploy . --config apps/kaja/fly.toml
FROM kajatools/kaja:latest

COPY apps/kaja/kaja.json /workspace/kaja.json
COPY apps/seating/proto/seating.proto /workspace/seating/seating.proto
COPY apps/boxoffice/proto/boxoffice.proto /workspace/boxoffice/boxoffice.proto
