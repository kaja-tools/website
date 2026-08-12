# Build the site with Node, then throw Node away. The runtime image is Caddy
# and a directory of files — the site is fully static, so there is nothing to
# render at request time and no reason to ship a JavaScript runtime.
FROM node:24-alpine AS builder

WORKDIR /app

# Dependencies first, so a content-only change reuses the install layer.
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM caddy:2.11-alpine

COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=builder /app/dist /srv

EXPOSE 8080
