# syntax=docker/dockerfile:1.7

FROM oven/bun:1.3.0-alpine AS build
WORKDIR /src
COPY web/package.json web/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile
COPY web ./
RUN bun run build

FROM caddy:2.10.2-alpine
RUN setcap -r /usr/bin/caddy
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /src/dist /srv
USER 1000:1000
