#!/usr/bin/env bash
# Daily backup of the production database, run by the systemd timer in deploy/systemd.
#
# The app image is distroless (no shell, no rm), so the copy is written by the app itself into a
# host folder mounted at /backups, and old copies are pruned from the host side.
#
# Exit codes: 0 backup written and verified, pruning done; 1 the backup failed.
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
DOCKER="${DOCKER:-docker}"
CONTAINER="${CONTAINER:-shadowcommit-app}"
BACKUP_DIR="${BACKUP_DIR:-$DEPLOY_DIR/backups}"
KEEP_DAYS="${KEEP_DAYS:-14}"

log() { echo "$(date -u +%FT%TZ) $*"; }

mkdir -p "$BACKUP_DIR"
stamp=$(date -u +%Y%m%dT%H%M%SZ)
name="shadowcommit-$stamp.db"

# The copy is made inside the app container, where the database and its permissions live.
if ! "$DOCKER" exec "$CONTAINER" /app/shadowcommit -backup "/backups/$name"; then
  log "BACKUP FAILED: $name was not written; the previous copies are untouched"
  exit 1
fi
log "backup written and verified: $BACKUP_DIR/$name"

# Prune only files this script names, and only once a new copy has been verified.
pruned=$(find "$BACKUP_DIR" -maxdepth 1 -name 'shadowcommit-*.db' -mtime +"$KEEP_DAYS" -print -delete | wc -l)
log "pruned $pruned copies older than $KEEP_DAYS days"
