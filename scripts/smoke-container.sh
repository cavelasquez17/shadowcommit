#!/usr/bin/env bash
# Smoke-tests a built ShadowCommit image: it must refuse to boot without a secret, then boot
# with one, report healthy, serve the SPA and the API, and run as a non-root user.
#
# Usage: scripts/smoke-container.sh <image> [host-port]
set -euo pipefail

IMAGE="${1:?usage: smoke-container.sh <image> [host-port]}"
PORT="${2:-18080}"
NAME="sc-smoke-$$"
fail() { echo "FAIL: $*" >&2; exit 1; }
cleanup() { docker rm -f "$NAME" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "1/6 refuses to boot in production without JWT_SECRET"
if out=$(docker run --rm "$IMAGE" 2>&1); then fail "container started without JWT_SECRET"; fi
grep -q "default JWT_SECRET" <<<"$out" || fail "unexpected failure output: $out"

echo "2/6 boots with a secret"
docker run -d --name "$NAME" --health-interval=2s --health-start-period=1s \
  -e JWT_SECRET=smoke-test-secret \
  -e WEB_BASE_URL="http://localhost:$PORT" -e APP_BASE_URL="http://localhost:$PORT" \
  -p "127.0.0.1:$PORT:8080" "$IMAGE" >/dev/null

echo "3/6 healthz answers and reports a version"
for _ in $(seq 1 30); do
  health=$(curl -fsS "http://127.0.0.1:$PORT/healthz" 2>/dev/null || true)
  [ -n "$health" ] && break
  sleep 1
done
grep -q '"status":"alive"' <<<"${health:-}" || fail "healthz did not answer: ${health:-<nothing>}"
grep -q '"version":"' <<<"$health" || fail "healthz has no version: $health"

echo "4/6 serves the SPA, the API, and keeps the JSON 404"
expect() { # expect <path> <status+type prefix>
  local got
  got=$(curl -s -o /dev/null -w '%{http_code} %{content_type}' "http://127.0.0.1:$PORT$1")
  [[ "$got" == "$2"* ]] || fail "GET $1: got '$got', want '$2...'"
}
expect / "200 text/html"
expect /feed "200 text/html"       # client route falls back to the SPA shell
expect /api/meta "200 application/json"
expect /api/nope "404 application/json"

echo "5/6 docker HEALTHCHECK turns healthy"
status=starting
for _ in $(seq 1 60); do
  status=$(docker inspect --format '{{.State.Health.Status}}' "$NAME")
  [ "$status" = healthy ] && break
  sleep 1
done
[ "$status" = healthy ] || fail "container health is '$status'"

echo "6/6 runs as a non-root user"
user=$(docker inspect --format '{{.Config.User}}' "$IMAGE")
{ [ -n "$user" ] && [ "$user" != "0" ] && [ "$user" != "root" ]; } || fail "image runs as '${user:-root}'"

echo "OK: $IMAGE passed the container smoke test"
