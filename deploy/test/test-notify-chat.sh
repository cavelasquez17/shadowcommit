#!/usr/bin/env bash
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NOTIFIER="$HERE/../notify-chat.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

export FAKE_CURL_ARGS="$TMP/curl-args"
ln -s "$HERE/fake-curl" "$TMP/curl"
export PATH="$TMP:$PATH"
export CHAT_WEBHOOK_URL='https://hooks.example.invalid/secret-token'

"$NOTIFIER" 'deployed commit abc"123; run https://github.com/example/actions/runs/1'
python3 - "$FAKE_CURL_ARGS" <<'PY'
import json
import sys

args = open(sys.argv[1], encoding="utf-8").read().splitlines()
payload = json.loads(args[args.index("--data") + 1])
assert payload["text"] == 'deployed commit abc"123; run https://github.com/example/actions/runs/1'
assert args[-1] == "https://hooks.example.invalid/secret-token"
PY

if env -u CHAT_WEBHOOK_URL "$NOTIFIER" 'message' >/dev/null 2>&1; then
  echo "notifier unexpectedly accepted a missing webhook URL" >&2
  exit 1
fi

echo "chat notifier tests passed"
