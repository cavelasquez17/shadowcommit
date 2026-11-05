#!/usr/bin/env bash
# Pull-based updater, run by the systemd timer in deploy/systemd.
#
# The host cannot be reached from CI, so it asks the registry whether the image changed. If it
# did, the app is recreated and must turn healthy; otherwise the previous image is restored.
#
# Exit codes: 0 up to date, deployed, or skipped because this image already failed;
#             1 deploy failed and was rolled back; 3 the registry could not be reached;
#             2 deploy failed and could not be rolled back (needs a human).
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
DOCKER="${DOCKER:-docker}"
CONTAINER="${CONTAINER:-shadowcommit-app}"
IMAGE="${SHADOWCOMMIT_IMAGE:-ghcr.io/cavelasquez17/shadowcommit:main}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-90}"   # seconds to wait for the new container to turn healthy
LOCK_FILE="${LOCK_FILE:-/tmp/shadowcommit-update.lock}"
# Remembers the image that last failed, so a broken image is not redeployed on every tick.
# After a rollback the running image differs from :main, so without this the timer would recreate
# the app on the bad image every couple of minutes, with ~90s of downtime each time.
# Delete this file to retry the same image on purpose.
STATE_DIR="${STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/shadowcommit}"
FAILED_FILE="$STATE_DIR/failed-image"
PULL_FAILED_FILE="$STATE_DIR/pull-failed"   # present while the registry is unreachable
# Local tag that keeps the previous image alive: once :main moves, the old image has no tag and
# `docker image prune` would delete it, taking the rollback target with it.
PREVIOUS_TAG="${PREVIOUS_TAG:-shadowcommit:previous}"
NOTIFY_CMD="${NOTIFY_CMD:-}"             # optional: called with one message argument
RUN_URL_LABEL='io.shadowcommit.github.actions.run-url'

log() { echo "$(date -u +%FT%TZ) $*"; }
notify() {
  log "$*"
  if [ -n "$NOTIFY_CMD" ]; then "$NOTIFY_CMD" "$*" || log "notify command failed"; fi
}
compose() { (cd "$DEPLOY_DIR" && "$DOCKER" compose "$@"); }

# Recreate only the app (Caddy is untouched, so there is no HTTPS blip) and block until its
# healthcheck passes. Fails if the container cannot start, turns unhealthy, or times out.
update_app() { compose up -d --no-deps --wait --wait-timeout "$HEALTH_TIMEOUT" app; }

# Never run two updates at once (the timer can fire while a slow one is still going).
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  log "another update is running; nothing to do"
  exit 0
fi

old_id=$("$DOCKER" inspect --format '{{.Image}}' "$CONTAINER" 2>/dev/null || true)
mkdir -p "$STATE_DIR"
# A registry or network failure must not look like a rollback (exit 1) nor stay silent, and it
# must not page anyone every two minutes for the length of an outage: say it once per outage.
if ! compose pull app || ! new_id=$("$DOCKER" image inspect --format '{{.Id}}' "$IMAGE"); then
  if [ -e "$PULL_FAILED_FILE" ]; then
    log "still cannot pull $IMAGE"
  else
    : > "$PULL_FAILED_FILE"
    notify "could not pull $IMAGE; the current version keeps running and the updater will keep trying"
  fi
  exit 3
fi
rm -f "$PULL_FAILED_FILE"

if [ "$old_id" = "$new_id" ]; then
  log "up to date ($new_id)"
  exit 0
fi

if [ -f "$FAILED_FILE" ] && [ "$(cat "$FAILED_FILE")" = "$new_id" ]; then
  log "image $new_id already failed; staying on the current one until a newer image is published"
  exit 0
fi

revision=$("$DOCKER" image inspect --format '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$IMAGE" 2>/dev/null || true)
short="${revision:0:7}"
run_url=$("$DOCKER" image inspect --format "{{index .Config.Labels \"$RUN_URL_LABEL\"}}" "$IMAGE" 2>/dev/null || true)
commit="${revision:-unknown revision}"
run_link="${run_url:-https://github.com/cavelasquez17/shadowcommit/commit/${revision:-unknown}}"
release="commit $commit; run $run_link"
log "updating ${old_id:-<none>} -> $new_id (${short:-unknown revision})"
if [ -n "$old_id" ]; then "$DOCKER" tag "$old_id" "$PREVIOUS_TAG"; fi
if update_app; then
  rm -f "$FAILED_FILE"
  notify "deployed $release"
  exit 0
fi

echo "$new_id" > "$FAILED_FILE"

notify "new image did not start healthy (waited up to ${HEALTH_TIMEOUT}s); $release"
if [ -z "$old_id" ]; then
  notify "first deployment failed and there is no previous image to roll back to; $release"
  exit 2
fi

log "rolling back to $old_id (tag $PREVIOUS_TAG)"
if SHADOWCOMMIT_IMAGE="$PREVIOUS_TAG" update_app; then
  notify "rolled back to the previous image after $release"
  exit 1
fi
notify "ROLLBACK FAILED: the app is not healthy and needs attention after $release"
exit 2
