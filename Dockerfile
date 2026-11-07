# syntax=docker/dockerfile:1

# Production image for Cinderfish: the Vue frontend is built once, the Go
# server is cross-compiled for the target platform, and the runtime layer
# carries only the binary, the web assets and CA roots.

FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src/web
# package.json pins pnpm through "packageManager"; corepack honours it.
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
# Nothing in the build needs lifecycle scripts; skipping them also keeps
# pnpm from failing on optional native packages it has not approved.
RUN pnpm install --frozen-lockfile --ignore-scripts
COPY web/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS server
# Overridable so builds behind a network that blocks proxy.golang.org can
# point at a mirror; CI uses the default.
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags='-s -w' -o /out/cinderfish ./cmd/cinderfish

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 1001 cinderfish \
    && adduser -S -u 1001 -G cinderfish -h /var/lib/cinderfish cinderfish \
    && mkdir -p /var/lib/cinderfish/media \
    && chown -R cinderfish:cinderfish /var/lib/cinderfish
WORKDIR /app
COPY --from=server /out/cinderfish /app/cinderfish
COPY --from=web /src/web/dist /app/web
COPY config.example.toml /app/config.example.toml

# Container defaults; every value still follows the CINDERFISH_* override
# rules, so a mounted config.toml or Compose environment replaces them.
ENV CINDERFISH_HOST=0.0.0.0 \
    CINDERFISH_PORT=3000 \
    CINDERFISH_WEB_DIR=/app/web \
    CINDERFISH_MEDIA_LOCAL_PATH=/var/lib/cinderfish/media

USER cinderfish:cinderfish
EXPOSE 3000
VOLUME ["/var/lib/cinderfish"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:3000/api/ping || exit 1

ENTRYPOINT ["/app/cinderfish"]
