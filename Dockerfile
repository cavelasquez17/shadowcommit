# syntax=docker/dockerfile:1

# ---- web: build the single-page app --------------------------------------------------------
# Always runs on the build machine; the output is plain static files for any architecture.
FROM --platform=$BUILDPLATFORM node:22-bookworm-slim AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- api: build the Go binary --------------------------------------------------------------
# SQLite needs CGO, so cross-compiling to another architecture needs a matching C compiler.
# Building on the build platform with a cross toolchain avoids slow QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS api
ARG BUILDARCH
ARG TARGETARCH
ARG VERSION=dev
RUN set -eux; \
    if [ "$TARGETARCH" = "$BUILDARCH" ]; then CC=gcc; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
        apt-get update && apt-get install -y --no-install-recommends gcc-aarch64-linux-gnu libc6-dev-arm64-cross \
        && CC=aarch64-linux-gnu-gcc; \
    elif [ "$TARGETARCH" = "amd64" ]; then \
        apt-get update && apt-get install -y --no-install-recommends gcc-x86-64-linux-gnu libc6-dev-amd64-cross \
        && CC=x86_64-linux-gnu-gcc; \
    else echo "unsupported TARGETARCH=$TARGETARCH" >&2; exit 1; fi; \
    echo "$CC" > /cc; rm -rf /var/lib/apt/lists/*
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CC="$(cat /cc)" CGO_ENABLED=1 GOOS=linux GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags "-s -w -X github.com/cavelasquez17/shadowcommit/api/internal/config.Version=${VERSION}" -o /out/shadowcommit ./cmd/api \
 && mkdir -p /out/data

# ---- runtime: minimal, non-root, no shell ---------------------------------------------------
# glibc-based because go-sqlite3 links dynamically. No RUN steps here, so this stage builds
# for any platform without emulation.
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=api /out/shadowcommit /app/shadowcommit
COPY --from=web /web/dist /app/web
COPY --from=api --chown=65532:65532 /out/data /data

# Secrets (JWT_SECRET, GitHub OAuth) are deliberately NOT set here: in production the app
# refuses to start without a real JWT_SECRET, which is what we want from a bare `docker run`.
ENV APP_ENV=production \
    PORT=8080 \
    DB_DRIVER=sqlite \
    DATABASE_URL=/data/shadowcommit.db \
    STATIC_DIR=/app/web

VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/app/shadowcommit", "-healthcheck"]
ENTRYPOINT ["/app/shadowcommit"]
