# syntax=docker/dockerfile:1

# ---- web: build the single-page app --------------------------------------------------------
# Always runs on the build machine; the output is plain static files for any architecture.
FROM --platform=$BUILDPLATFORM node:22-bookworm-slim AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
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
    # The compiler prefix (triplet) and the Debian package name differ for amd64:
    # x86_64-linux-gnu-gcc is shipped by gcc-x86-64-linux-gnu (hyphens, not underscores).
    case "$TARGETARCH" in \
        arm64) triplet=aarch64-linux-gnu; gccpkg=gcc-aarch64-linux-gnu; libc=arm64 ;; \
        amd64) triplet=x86_64-linux-gnu; gccpkg=gcc-x86-64-linux-gnu; libc=amd64 ;; \
        *) echo "unsupported TARGETARCH=$TARGETARCH" >&2; exit 1 ;; \
    esac; \
    if [ "$TARGETARCH" = "$BUILDARCH" ]; then echo gcc > /cc; \
    else \
        apt-get update; \
        apt-get install -y --no-install-recommends "$gccpkg" "libc6-dev-$libc-cross"; \
        rm -rf /var/lib/apt/lists/*; \
        echo "$triplet-gcc" > /cc; \
    fi
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY api/ ./
# Cache mounts keep the module and compiler caches between builds, so a source change does
# not recompile go-sqlite3's C code from scratch.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CC="$(cat /cc)" CGO_ENABLED=1 GOOS=linux GOARCH="$TARGETARCH" \
    go build -trimpath -ldflags "-s -w -X github.com/cavelasquez17/shadowcommit/api/internal/config.Version=${VERSION}" -o /out/shadowcommit ./cmd/api \
 && mkdir -p /out/data

# ---- runtime: minimal, non-root, no shell ---------------------------------------------------
# glibc-based because go-sqlite3 links dynamically. No RUN steps here, so this stage builds
# for any platform without emulation.
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=api /out/shadowcommit /app/shadowcommit
COPY --from=web /web/dist /app/web
# distroless has no shell to mkdir with, so /data is created in the build stage; 65532 is the
# uid of the image's `nonroot` user and must own it for SQLite to write.
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
