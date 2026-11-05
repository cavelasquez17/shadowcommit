#!/usr/bin/env bash
# Send one updater message to a Slack-compatible incoming webhook.
set -euo pipefail

if [ -z "${CHAT_WEBHOOK_URL:-}" ]; then
  echo "CHAT_WEBHOOK_URL is not configured" >&2
  exit 1
fi
if [ "$#" -ne 1 ]; then
  echo "expected exactly one notification message" >&2
  exit 1
fi

payload=$(python3 -c 'import json, sys; print(json.dumps({"text": sys.argv[1]}))' "$1")
if ! curl --fail --silent --max-time 10 \
  -H 'Content-Type: application/json' \
  --data "$payload" "$CHAT_WEBHOOK_URL" >/dev/null 2>&1; then
  echo "chat notification request failed" >&2
  exit 1
fi
