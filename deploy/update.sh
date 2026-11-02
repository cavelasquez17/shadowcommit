#!/usr/bin/env bash
# Pull-based updater, run by the systemd timer in deploy/systemd.
#
# The host cannot be reached from CI, so it asks the registry whether the image changed. If it
# did, the app is recreated and must turn healthy; otherwise the previous image is restored.
#
# Exit codes: 0 up to date or deployed, 1 deploy failed and was rolled back,
#             2 deploy failed and could not be rolled back (needs a human).
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
DOCKER="${DOCKER:-docker}"
CONTAINER="${CONTAINER:-shadowcommit-app}"
IMAGE="${SHADOWCOMMIT_IMAGE:-ghcr.io/cavelasquez17/shadowcommit:main}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-90}"   # seconds to wait for the new container to turn healthy
HEALTH_INTERVAL="${HEALTH_INTERVAL:-3}"
LOCK_FILE="${LOCK_FILE:-/tmp/shadowcommit-update.lock}"
NOTIFY_CMD="${NOTIFY_CMD:-}"             # optional: called with one message argument

log() { echo "$(date -u +%FT%TZ) $*"; }
notify() {
  log "$*"
  if [ -n "$NOTIFY_CMD" ]; then "$NOTIFY_CMD" "$*" || log "notify command failed"; fi
}
compose() { (cd "$DEPLOY_DIR" && "$DOCKER" compose "$@"); }

wait_healthy() {
  local waited=0 status
  while [ "$waited" -lt "$HEALTH_TIMEOUT" ]; do
    status=$("$DOCKER" inspect --format '{{.State.Health.Status}}' "$CONTAINER" 2>/dev/null || echo missing)
    [ "$status" = healthy ] && return 0
    sleep "$HEALTH_INTERVAL"
    waited=$((waited + HEALTH_INTERVAL))
  done
  return 1
}

# Never run two updates at once (the timer can fire while a slow one is still going).
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  log "another update is running; nothing to do"
  exit 0
fi

old_id=$("$DOCKER" inspect --format '{{.Image}}' "$CONTAINER" 2>/dev/null || true)
compose pull app
new_id=$("$DOCKER" image inspect --format '{{.Id}}' "$IMAGE")

if [ "$old_id" = "$new_id" ]; then
  log "up to date ($new_id)"
  exit 0
fi

revision=$("$DOCKER" image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$IMAGE" 2>/dev/null || true)
short="${revision:0:7}"
log "updating ${old_id:-<none>} -> $new_id (${short:-unknown revision})"
compose up -d

if wait_healthy; then
  notify "deployed ${short:-new image}"
  exit 0
fi

notify "new image ${short:-?} did not become healthy within ${HEALTH_TIMEOUT}s"
if [ -z "$old_id" ]; then
  notify "first deployment failed and there is no previous image to roll back to"
  exit 2
fi

log "rolling back to $old_id"
SHADOWCOMMIT_IMAGE="$old_id" compose up -d app
if wait_healthy; then
  notify "rolled back to the previous image"
  exit 1
fi
notify "ROLLBACK FAILED: the app is not healthy and needs attention"
exit 2
