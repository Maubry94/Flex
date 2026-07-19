# syntax=docker/dockerfile:1.7

FROM node:24-alpine AS web-dependencies
WORKDIR /src
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/package.json
RUN --mount=type=cache,target=/root/.npm npm ci

FROM web-dependencies AS web-build
COPY apps/web apps/web
RUN npm run build

FROM golang:1.25-alpine AS server-build
WORKDIR /src/apps/server
COPY apps/server/go.mod apps/server/go.sum ./
COPY apps/server/cmd ./cmd
COPY apps/server/internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/flex ./cmd/flex

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates ffmpeg tzdata \
    && addgroup -S -g 1000 flex \
    && adduser -S -D -H -u 1000 -G flex flex \
    && mkdir -p /app/web /config /cache /media \
    && chown -R flex:flex /app /config /cache

COPY --from=server-build /out/flex /usr/local/bin/flex
COPY --from=web-build /src/apps/web/dist /app/web

USER flex:flex
EXPOSE 8080
VOLUME ["/config", "/cache"]

ENV FLEX_HOST=0.0.0.0 \
    FLEX_PORT=8080 \
    FLEX_CONFIG_DIR=/config \
    FLEX_CACHE_DIR=/cache \
    FLEX_MEDIA_DIR=/media \
    FLEX_WEB_DIR=/app/web

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/flex"]
