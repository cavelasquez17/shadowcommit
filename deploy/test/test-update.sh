#!/usr/bin/env bash
# Scenario tests for deploy/update.sh using a fake docker. Run: deploy/test/test-update.sh
#
# Each check is "check <description> <command...>" and passes when the command succeeds, so a
# condition that cannot even be evaluated fails loudly instead of being skipped.
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UPDATE="$HERE/../update.sh"
failures=0
cases=0

# new_case <name>: fresh state; the registry serves "new". Tests add "running"/health files.
new_case() {
  cases=$((cases + 1))
  CASE="$1"
  FAKE_DIR=$(mktemp -d)
  export FAKE_DIR DOCKER="$HERE/fake-docker" DEPLOY_DIR="$FAKE_DIR" LOCK_FILE="$FAKE_DIR/lock"
  export HEALTH_TIMEOUT=2 STATE_DIR="$FAKE_DIR/state"
  unset NOTIFY_CMD SHADOWCOMMIT_IMAGE FAKE_PRUNE_OLD FAKE_UP_FAILS FAKE_PULL_FAILS
  echo "sha256:new" > "$FAKE_DIR/registry_image"
  echo "sha256:aaaaaaa0001" > "$FAKE_DIR/revision"
  echo "$CASE"
}
set_running() { echo "$1" > "$FAKE_DIR/running_image"; }
set_health() { echo "$2" > "$FAKE_DIR/health_$1"; }
make_notifier() { # make_notifier <exit-code>: appends every message to $FAKE_DIR/notes, then exits
  # shellcheck disable=SC2016 # the $1 belongs to the generated script, not to this one
  printf '#!/usr/bin/env bash\necho "$1" >> "%s/notes"\nexit %s\n' "$FAKE_DIR" "${1:-0}" > "$FAKE_DIR/notify"
  chmod +x "$FAKE_DIR/notify"
  export NOTIFY_CMD="$FAKE_DIR/notify"
}

run() { OUT=$("$UPDATE" 2>&1); RC=$?; }
running() { cat "$FAKE_DIR/running_image" 2>/dev/null || echo none; }
# Number of recorded docker calls matching a pattern (0 when docker was never called).
called() { if [ -f "$FAKE_DIR/calls" ]; then grep -c -- "$1" "$FAKE_DIR/calls" || true; else echo 0; fi; }
out_has() { grep -q -- "$1" <<<"$OUT"; }
notes_have() { grep -q -- "$1" "$FAKE_DIR/notes" 2>/dev/null; }

check() {
  local description=$1
  shift
  if "$@"; then
    echo "  ok   $description"
  else
    echo "  FAIL $description"
    printf '%s\n' "$OUT" | while IFS= read -r line; do echo "       | $line"; done
    failures=$((failures + 1))
  fi
}

new_case "already up to date"
set_running sha256:new; run
check "exits 0" test "$RC" -eq 0
check "does not recreate the app" test "$(called 'compose up')" -eq 0
check "says it is up to date" out_has 'up to date'

new_case "update that turns healthy"
set_running sha256:old; run
check "exits 0" test "$RC" -eq 0
check "now runs the new image" test "$(running)" = sha256:new
check "recreates only the app, without touching Caddy" grep -q -- 'up -d --no-deps --wait --wait-timeout 2 app' "$FAKE_DIR/calls"

new_case "first deployment that turns healthy"
run
check "exits 0" test "$RC" -eq 0
check "runs the new image" test "$(running)" = sha256:new

new_case "bad update is rolled back"
set_running sha256:old; set_health sha256:new unhealthy; run
check "exits 1" test "$RC" -eq 1
check "restored the previous image" test "$(running)" = sha256:old
check "reports the rollback" out_has 'rolled back'

new_case "rollback still works when the previous image was pruned"
export FAKE_PRUNE_OLD=1
set_running sha256:old; set_health sha256:new unhealthy; run
check "exits 1 (rolled back)" test "$RC" -eq 1
check "restored the previous image" test "$(running)" = sha256:old
check "kept it alive under a tag before updating" grep -q -- 'tag sha256:old' "$FAKE_DIR/calls"

new_case "an update whose containers fail to start is rolled back too"
export FAKE_UP_FAILS=1
set_running sha256:old; run
check "exits 1 (rolled back), not an abort" test "$RC" -eq 1
check "still runs the previous image" test "$(running)" = sha256:old
check "reports the rollback" out_has 'rolled back'

new_case "a bad image is not retried on every tick"
set_running sha256:old; set_health sha256:new unhealthy; run
check "first run rolls back (exit 1)" test "$RC" -eq 1
ups_after_first=$(called 'compose up')
run
check "second run does nothing (exit 0)" test "$RC" -eq 0
check "says it already failed" out_has 'already failed'
check "did not recreate the app again" test "$(called 'compose up')" -eq "$ups_after_first"
check "still runs the previous image" test "$(running)" = sha256:old
echo "sha256:fixed" > "$FAKE_DIR/registry_image"
run
check "a newer image is deployed after the failure" test "$RC" -eq 0
check "and is running" test "$(running)" = sha256:fixed
check "the earlier failure is forgotten after a good deploy" test ! -e "$STATE_DIR/failed-image"

new_case "an unreachable registry is reported once, with its own exit code"
export FAKE_PULL_FAILS=1
set_running sha256:old; make_notifier 0; run
check "exits 3, not 1 (that means a rollback)" test "$RC" -eq 3
check "tells the operator" notes_have 'could not pull'
check "leaves the app alone" test "$(running)" = sha256:old
run
check "keeps exiting 3" test "$RC" -eq 3
check "but does not repeat the notification" test "$(grep -c 'could not pull' "$FAKE_DIR/notes")" -eq 1
unset FAKE_PULL_FAILS
run
check "recovers once the registry is back (exit 0)" test "$RC" -eq 0
check "and deploys the new image" test "$(running)" = sha256:new
export FAKE_PULL_FAILS=1
run
check "a new outage is reported again" test "$(grep -c 'could not pull' "$FAKE_DIR/notes")" -eq 2
unset FAKE_PULL_FAILS

new_case "bad update whose rollback also fails"
set_running sha256:old; set_health sha256:new unhealthy; set_health sha256:old unhealthy; run
check "exits 2" test "$RC" -eq 2
check "asks for attention" out_has 'ROLLBACK FAILED'

new_case "first deployment that never turns healthy"
set_health sha256:new unhealthy; run
check "exits 2 (nothing to roll back to)" test "$RC" -eq 2
check "says there is no previous image" out_has 'no previous image'

new_case "a second update while one is running"
set_running sha256:old
( exec 8>"$LOCK_FILE"; flock 8; sleep 1 ) & holder=$!
sleep 0.3; run; wait "$holder"
check "exits 0 without doing anything" test "$RC" -eq 0
check "never pulled" test "$(called 'compose pull')" -eq 0
check "left the app alone" test "$(running)" = sha256:old

new_case "a failed deploy is reported to the notify command"
set_running sha256:old; set_health sha256:new unhealthy; make_notifier 0; run
check "reported the failure" notes_have 'did not start healthy'
check "reported the rollback" notes_have 'rolled back'

new_case "a successful deploy is announced with its revision"
set_running sha256:old; echo "abcdef1234567890" > "$FAKE_DIR/revision"; make_notifier 0; run
check "exits 0" test "$RC" -eq 0
check "announced the short revision" notes_have 'deployed abcdef1'

new_case "a failing notify command does not break the deploy"
set_running sha256:old; make_notifier 1; run
check "still exits 0" test "$RC" -eq 0
check "the new image is running" test "$(running)" = sha256:new

new_case "the default image is the same in update.sh and docker-compose.yml"
default_in() { grep -o 'SHADOWCOMMIT_IMAGE:-[^}"]*' "$1" | head -n1 | cut -d- -f2-; }
script_default=$(default_in "$UPDATE")
compose_default=$(default_in "$HERE/../docker-compose.yml")
OUT="update.sh: $script_default | compose: $compose_default"
check "both define a default" test -n "$script_default" -a -n "$compose_default"
check "and they are identical" test "$script_default" = "$compose_default"

echo
echo "$cases scenarios, $failures failed checks"
[ "$failures" -eq 0 ]
