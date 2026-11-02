#!/usr/bin/env bash
# Scenario tests for deploy/update.sh using a fake docker. Run: deploy/test/test-update.sh
set -uo pipefail
# A bad arithmetic expression aborts the whole command line silently; make that a loud failure.
trap 'echo "FAIL: a check could not be evaluated (line $LINENO)"; exit 1' ERR

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UPDATE="$HERE/../update.sh"
failures=0
cases=0

# new_case <name>: fresh state dir; "old" is what runs now, "new" is what the registry serves.
new_case() {
  cases=$((cases + 1))
  CASE="$1"
  FAKE_DIR=$(mktemp -d)
  export FAKE_DIR DOCKER="$HERE/fake-docker" DEPLOY_DIR="$FAKE_DIR" LOCK_FILE="$FAKE_DIR/lock"
  export HEALTH_TIMEOUT=2 HEALTH_INTERVAL=1
  unset NOTIFY_CMD SHADOWCOMMIT_IMAGE
  echo "sha256:new" > "$FAKE_DIR/registry_image"
  echo "sha256:aaaaaaa0001" > "$FAKE_DIR/revision"
}
run() { OUT=$("$UPDATE" 2>&1); RC=$?; }
running() { cat "$FAKE_DIR/running_image" 2>/dev/null || echo none; }
# Number of recorded docker calls matching a pattern (0 when docker was never called).
called() { if [ -f "$FAKE_DIR/calls" ]; then grep -c -- "$1" "$FAKE_DIR/calls" || true; else echo 0; fi; }
check() { # check <description> <condition-exit-code>
  if [ "$2" -eq 0 ]; then echo "  ok   $1"; else echo "  FAIL $1"; echo "$OUT" | sed 's/^/       | /'; failures=$((failures + 1)); fi
}

new_case "already up to date"; echo "sha256:new" > "$FAKE_DIR/running_image"; run
echo "$CASE"
check "exits 0" $((RC != 0)); check "does not recreate the app" $(( $(called "compose up") != 0 )); check "says it is up to date" $(grep -q 'up to date' <<<"$OUT" && echo 0 || echo 1)

new_case "update that turns healthy"; echo "sha256:old" > "$FAKE_DIR/running_image"; echo "sha256:old" > "$FAKE_DIR/pulled_image"; run
echo "$CASE"
check "exits 0" $((RC != 0)); check "now runs the new image" $([ "$(running)" = "sha256:new" ] && echo 0 || echo 1)

new_case "first deployment that turns healthy"; run
echo "$CASE"
check "exits 0" $((RC != 0)); check "runs the new image" $([ "$(running)" = "sha256:new" ] && echo 0 || echo 1)

new_case "bad update is rolled back"; echo "sha256:old" > "$FAKE_DIR/running_image"; echo "unhealthy" > "$FAKE_DIR/health_sha256:new"; run
echo "$CASE"
check "exits 1" $((RC != 1)); check "restored the previous image" $([ "$(running)" = "sha256:old" ] && echo 0 || echo 1); check "reports the rollback" $(grep -q 'rolled back' <<<"$OUT" && echo 0 || echo 1)

new_case "bad update whose rollback also fails"; echo "sha256:old" > "$FAKE_DIR/running_image"; echo "unhealthy" > "$FAKE_DIR/health_sha256:new"; echo "unhealthy" > "$FAKE_DIR/health_sha256:old"; run
echo "$CASE"
check "exits 2" $((RC != 2)); check "asks for attention" $(grep -q 'ROLLBACK FAILED' <<<"$OUT" && echo 0 || echo 1)

new_case "first deployment that never turns healthy"; echo "unhealthy" > "$FAKE_DIR/health_sha256:new"; run
echo "$CASE"
check "exits 2 (nothing to roll back to)" $((RC != 2)); check "says there is no previous image" $(grep -q 'no previous image' <<<"$OUT" && echo 0 || echo 1)

new_case "a second update while one is running"; echo "sha256:old" > "$FAKE_DIR/running_image"
( exec 8>"$LOCK_FILE"; flock 8; sleep 3 ) & holder=$!; sleep 0.5; run; wait "$holder"
echo "$CASE"
check "exits 0 without doing anything" $((RC != 0)); check "never pulled" $(( $(called "compose pull") != 0 )); check "left the app alone" $([ "$(running)" = "sha256:old" ] && echo 0 || echo 1)

new_case "notifications reach the notify command"; echo "sha256:old" > "$FAKE_DIR/running_image"; echo "unhealthy" > "$FAKE_DIR/health_sha256:new"
printf '#!/usr/bin/env bash\necho "$1" >> "%s/notes"\n' "$FAKE_DIR" > "$FAKE_DIR/notify"; chmod +x "$FAKE_DIR/notify"; NOTIFY_CMD="$FAKE_DIR/notify" run
echo "$CASE"
check "reported the failure and the rollback" $([ "$(wc -l < "$FAKE_DIR/notes")" -ge 2 ] && echo 0 || echo 1)

new_case "a successful deploy is announced with its revision"; echo "sha256:old" > "$FAKE_DIR/running_image"; echo "abcdef1234567890" > "$FAKE_DIR/revision"
printf '#!/usr/bin/env bash\necho "$1" >> "%s/notes"\n' "$FAKE_DIR" > "$FAKE_DIR/notify"; chmod +x "$FAKE_DIR/notify"; NOTIFY_CMD="$FAKE_DIR/notify" run
echo "$CASE"
check "exits 0" $((RC != 0)); check "announced the short revision" $(grep -q 'deployed abcdef1' "$FAKE_DIR/notes" && echo 0 || echo 1)

new_case "a failing notify command does not break the deploy"; echo "sha256:old" > "$FAKE_DIR/running_image"
printf '#!/usr/bin/env bash\nexit 1\n' > "$FAKE_DIR/notify"; chmod +x "$FAKE_DIR/notify"; NOTIFY_CMD="$FAKE_DIR/notify" run
echo "$CASE"
check "still exits 0" $((RC != 0))

echo; echo "$cases scenarios, $failures failed checks"
[ "$failures" -eq 0 ]
