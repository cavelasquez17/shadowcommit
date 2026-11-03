#!/usr/bin/env bash
# Scenario tests for deploy/backup.sh with the fake docker CLI. Run: deploy/test/test-backup.sh
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP="$HERE/../backup.sh"
failures=0

check() {
  local description=$1
  shift
  if "$@"; then
    echo "  ok   $description"
  else
    echo "  FAIL $description"
    failures=$((failures + 1))
  fi
}

new_case() {
  echo "$1"
  FAKE_DIR=$(mktemp -d)
  export FAKE_DIR DOCKER="$HERE/fake-docker" BACKUP_DIR="$FAKE_DIR/backups" KEEP_DAYS=14
  unset FAKE_BACKUP_FAILS
  mkdir -p "$BACKUP_DIR"
}

old_file() { touch -d "$2 days ago" "$BACKUP_DIR/$1"; }

new_case "a successful backup is verified and old copies are pruned"
old_file shadowcommit-20250101T000000Z.db 40
old_file shadowcommit-20250110T000000Z.db 30
old_file shadowcommit-keep-recent.db 2
old_file unrelated-notes.txt 90
out=$(bash "$BACKUP" 2>&1); rc=$?
check "exits 0" test "$rc" -eq 0
check "asks the app to write and verify the copy" grep -q -- 'exec .* -backup /backups/shadowcommit-' "$FAKE_DIR/calls"
check "prunes the copies older than the retention" test ! -e "$BACKUP_DIR/shadowcommit-20250101T000000Z.db"
check "prunes every old copy, not only one" test ! -e "$BACKUP_DIR/shadowcommit-20250110T000000Z.db"
check "keeps recent copies" test -e "$BACKUP_DIR/shadowcommit-keep-recent.db"
check "never touches files it did not name" test -e "$BACKUP_DIR/unrelated-notes.txt"
check "reports how many it pruned" grep -q 'pruned 2 copies' <<<"$out"

new_case "a failed backup keeps every existing copy"
old_file shadowcommit-20250101T000000Z.db 40
FAKE_BACKUP_FAILS=1 bash "$BACKUP" >/dev/null 2>&1; rc=$?
check "exits 1" test "$rc" -eq 1
check "does not prune anything when the new copy failed" test -e "$BACKUP_DIR/shadowcommit-20250101T000000Z.db"

echo
[ "$failures" -eq 0 ] && echo "all backup checks passed" || echo "$failures failed checks"
[ "$failures" -eq 0 ]
