# Deploying ShadowCommit

ShadowCommit runs as two containers on a single small ARM64 host: the app (API plus the built web
app, one origin) and Caddy, which terminates HTTPS. Nothing is built on the host.

```
GitHub (main) ──CI──▶ ghcr.io/<owner>/shadowcommit:main ◀──pull── host (systemd timer → update.sh)
                                                                     │
                                          browser ──HTTPS──▶ Caddy ──▶ app ──▶ SQLite volume
```

The host is **not reachable from CI** (its SSH access is restricted to a fixed address), so releases
are **pull-based**: every two minutes the host asks the registry whether `:main` changed, recreates
the stack if so, waits for the app to turn healthy and restores the previous image if it does not.

Host-specific details (addresses, user names, firewall rules) are intentionally **not** in this
repository. This document uses placeholders: `<your-subdomain>`, `<deploy-user>`.

## Prerequisites (owner actions)

These touch accounts or security settings and are done by the repository owner, not by tooling.

1. **A host name.** Create a free subdomain (for example at DuckDNS) and point its `A` record at the
   host's public address.
2. **Open ports 80 and 443** to the internet in the host's cloud network rules (the OCI security
   list) and in any host firewall. Caddy needs 80 for the Let's Encrypt challenge and the redirect,
   and 443 for visitors.
3. **A GitHub OAuth App** (Settings → Developer settings → OAuth Apps). Callback URL:
   `https://<your-subdomain>/api/auth/github/callback`. Keep the client secret to yourself: it only
   ever goes into the `.env` file on the host.
4. **Make the container package public** once, after the first publish: GitHub → Packages →
   `shadowcommit` → Package settings → Change visibility. (The image contains no secrets, and public
   means the host needs no registry credentials.)

## One-time host setup

Provide a Linux host with Docker Engine and its Compose plugin, Git, and systemd user services. The
deploy user must belong to the `docker` group; install these using the host distribution's package
instructions before continuing. Chat notifications additionally need `curl` and `python3` on the
host. Then, as the deploy user:

```bash
git clone https://github.com/<owner>/shadowcommit.git ~/shadowcommit
cd ~/shadowcommit/deploy

cp .env.example .env
chmod 600 .env
# Fill in .env: SHADOWCOMMIT_DOMAIN (<your-subdomain>, no https://), GITHUB_CLIENT_ID /
# GITHUB_CLIENT_SECRET, and a fresh secret:
openssl rand -hex 32        # paste as JWT_SECRET

docker compose up -d
docker compose ps           # app should become "healthy"
curl -s https://<your-subdomain>/healthz
```

### First sign-in (the invite-only bootstrap)

`INVITE_ONLY=true` means signing up needs an invite code, but only an existing member can create one, so
the very first account cannot get in. For that first login only:

```bash
cd ~/shadowcommit/deploy
sed -i 's|^INVITE_ONLY=.*|INVITE_ONLY=false|' .env
docker compose up -d --force-recreate --no-deps app     # sign in with GitHub now
sed -i 's|^INVITE_ONLY=.*|INVITE_ONLY=true|' .env
docker compose up -d --force-recreate --no-deps app     # close registration again
```

While registration is open anyone with the link can create an account, so keep that window short. After
that, create invite codes from the app's **Invites** page. The same recreate command is needed whenever
`.env` changes (for example after adding the GitHub client id and secret): a running container keeps the
environment it was created with.

Then enable automatic updates with systemd user units:

```bash
mkdir -p ~/.config/systemd/user
cp systemd/shadowcommit-update.* ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now shadowcommit-update.timer
loginctl enable-linger "$USER"      # keep the timer running when nobody is logged in
```

### Chat deploy notifications (optional)

The updater can post success, failure and rollback messages to a Slack-compatible incoming webhook.
Create the webhook in your chat workspace, then store its URL only in the host's ignored
`deploy/update.env` file. The URL is a secret; do not paste it in `.env`, shell history, or this repo.

```bash
cd ~/shadowcommit/deploy
umask 077
cat > update.env <<'EOF'
NOTIFY_CMD=/home/<deploy-user>/shadowcommit/deploy/notify-chat.sh
CHAT_WEBHOOK_URL=https://<chat-provider>/<secret-webhook-path>
EOF
chmod 600 update.env
systemctl --user daemon-reload
systemctl --user start shadowcommit-update.service
```

The service loads `update.env`; `notify-chat.sh` sends one JSON-encoded message and does not print the
webhook URL. Notifications include the full commit SHA and link to the GitHub Actions run that built
the image. An image built before run-link metadata was added falls back to its commit page. A notifier
failure is logged without failing an otherwise healthy deployment. Leave `NOTIFY_CMD` unset to use the
journal only.

## How a release flows

1. A pull request is merged to `main`.
2. CI runs the tests and lint, builds and smoke-tests the image, then **publishes**
   `ghcr.io/<owner>/shadowcommit:main` (also `:latest` and the immutable `:sha-<short>`) for
   `linux/amd64` and `linux/arm64`.
3. Within about two minutes the host's timer runs `deploy/update.sh`:
   - same image as the running one → nothing happens;
   - new image → recreate, wait up to 90 s for the health check;
   - unhealthy, or the containers fail to start → the previous image is restored automatically.
     It is kept under the local tag `shadowcommit:previous`, so a `docker image prune` cannot delete
     the rollback target.
4. `GET /healthz` reports the commit that is running (`"version"`).

`update.sh` exit codes: `0` up to date, deployed, or skipped because that exact image already failed;
`1` deploy failed and was rolled back; `2` deploy failed and could not be rolled back (needs a
person); `3` the registry could not be reached (the current version keeps running; reported once per
outage).

A failed image is remembered in `~/.local/state/shadowcommit/failed-image` and is not retried until
the registry serves a different one. Delete that file to retry the same image on purpose.

## Day-to-day operations

```bash
# what is running, and which commit
docker compose -f ~/shadowcommit/deploy/docker-compose.yml ps
curl -s https://<your-subdomain>/healthz

# logs
docker logs --tail 100 shadowcommit-app
journalctl --user -u shadowcommit-update.service --since today

# run an update right now
systemctl --user start shadowcommit-update.service

# pause automatic updates
systemctl --user disable --now shadowcommit-update.timer
```

### Roll back by hand

Pin a known-good image (`shadowcommit:previous`, or a `sha-<short>` tag from the registry) and stop the timer so it does not
move forward again:

```bash
systemctl --user disable --now shadowcommit-update.timer
cd ~/shadowcommit/deploy
SHADOWCOMMIT_IMAGE=ghcr.io/<owner>/shadowcommit:sha-<short> docker compose up -d app
```

Re-enable the timer when the fix is merged.

### Notifications

When configured, success, failed health checks, and rollback outcomes are sent to the chat webhook
with the full image commit SHA and workflow run link. Registry outage notices have no candidate
release yet, so they identify the image that could not be pulled instead. All messages also appear in
the systemd journal (`journalctl --user -u shadowcommit-update.service`).

## Data and backups

SQLite lives in the `shadowcommit-data` volume (mounted at `/data`) and Caddy's certificates in
`caddy-data`. Losing the first one loses every account and post.

**Daily copies.** A systemd timer runs `deploy/backup.sh` at 03:30. The app writes a verified copy
into `deploy/backups/` (`VACUUM INTO`, so it is consistent while the site is in use), and copies
older than 14 days are removed. A failed copy changes nothing that already exists.

One-time setup on the host, so the app can write there and the deploy user can prune:

```bash
cd ~/shadowcommit/deploy
mkdir -p backups
sudo chown "$USER":65532 backups && chmod 770 backups
cp systemd/shadowcommit-backup.* ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now shadowcommit-backup.timer
systemctl --user start shadowcommit-backup.service   # first copy now, to check it works
ls -lh backups/
```

**Copies live on the same machine as the data.** They protect against a bad deploy or a corrupted
database, not against losing the host. Before there are real accounts to protect, copy the folder
off the host as well (for example with `rsync` to another machine or to storage you own).

**Restore** (stops the app, keeps the current database as a file in case the restore is wrong):

```bash
cd ~/shadowcommit/deploy
docker compose stop app
docker run --rm -v shadowcommit_shadowcommit-data:/data -v "$PWD/backups:/backups" \
  busybox sh -c 'mv /data/shadowcommit.db /data/shadowcommit.db.replaced && cp /backups/<chosen-copy>.db /data/shadowcommit.db && chown 65532:65532 /data /data/shadowcommit.db'
docker compose start app
curl -s https://<your-subdomain>/healthz
```

Rehearsed on production data (not the live volume): the copy restored into a scratch volume started healthy with the expected user. The rehearsal needs the data folder owned by the app user too: SQLite creates a journal file next to the database, and a root-owned folder makes every write fail with "attempt to write a readonly database". Check the restore before relying on it: the app should report healthy and the expected users should be
visible. The file `shadowcommit.db.replaced` is the state before the restore and can be deleted once
you are satisfied.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Browser cannot connect | Ports 80/443 not open in the cloud rules or the host firewall |
| Certificate error right after the first start | DNS not pointing at the host yet; Caddy retries on its own, see `docker logs shadowcommit-caddy` |
| App container restarts, log says it refuses to boot | `JWT_SECRET` missing or still the default, or `DEV_LOGIN` set |
| No backup copies appear | The timer is not enabled, or `deploy/backups` is not writable by the app: run the one-time setup |
| Login with GitHub fails | The OAuth app's callback URL does not match `https://<your-subdomain>/api/auth/github/callback` |
| `update.sh` exits 2 | The new image is unhealthy and the rollback also failed: inspect `docker logs shadowcommit-app` |
| Update never happens | Package is still private (the host cannot pull) or the timer is not enabled |
| Deploy message missing | Check `update.env` permissions and `journalctl --user -u shadowcommit-update.service`; notifier failures do not stop deployments |
