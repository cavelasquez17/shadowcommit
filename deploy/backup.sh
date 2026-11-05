#!/usr/bin/env bash
# Daily backup of the production database and mood photos, run by systemd timer in deploy/systemd.
#
# The app image is distroless (no shell, no rm), so the copy is written by the app itself into a
# host folder mounted at /backups, and old pairs are pruned from the host side.
#
# Exit codes: 0 backup written and verified, pruning done; 1 the backup failed.
set -euo pipefail

DOCKER="${DOCKER:-docker}"
CONTAINER="${CONTAINER:-shadowcommit-app}"
BACKUP_DIR="$HOME/.local/state/shadowcommit/backups"
KEEP_DAYS="${KEEP_DAYS:-14}"

log() { echo "$(date -u +%FT%TZ) $*"; }

mkdir -p "$BACKUP_DIR"
stamp=$(date -u +%Y%m%dT%H%M%SZ)
name="shadowcommit-$stamp.db"
photos="shadowcommit-$stamp.photos.tar.gz"

# The database and photo archive are made under one shared photo-store lock so the pair agrees.
if ! "$DOCKER" exec "$CONTAINER" /app/shadowcommit -backup "/backups/$name" -backup-photos "/backups/$photos"; then
  log "BACKUP FAILED: $name and $photos were not written as a verified pair; previous copies are untouched"
  exit 1
fi
log "database and photo backups written and verified: $BACKUP_DIR/$name, $BACKUP_DIR/$photos"

# Prune only files this script names, and only once a new matching pair has been verified.
pruned_db=$(find "$BACKUP_DIR" -maxdepth 1 -name 'shadowcommit-*.db' -mtime +"$KEEP_DAYS" -print -delete | wc -l)
pruned_photos=$(find "$BACKUP_DIR" -maxdepth 1 -name 'shadowcommit-*.photos.tar.gz' -mtime +"$KEEP_DAYS" -print -delete | wc -l)
pruned=$((pruned_db + pruned_photos))
log "pruned $pruned copies older than $KEEP_DAYS days"
