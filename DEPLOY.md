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

On the host, as the deploy user (a member of the `docker` group):

```bash
git clone https://github.com/<owner>/shadowcommit.git ~/shadowcommit
cd ~/shadowcommit/deploy

cp .env.example .env
chmod 600 .env
# Fill in .env: SHADOWCOMMIT_DOMAIN, APP_BASE_URL, WEB_BASE_URL, CORS_ORIGINS (all https://<your-subdomain>),
# GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET, and a fresh secret:
openssl rand -hex 32        # paste as JWT_SECRET

docker compose up -d
docker compose ps           # app should become "healthy"
curl -s https://<your-subdomain>/healthz
```

Then enable automatic updates with systemd user units:

```bash
mkdir -p ~/.config/systemd/user
cp systemd/shadowcommit-update.* ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now shadowcommit-update.timer
loginctl enable-linger "$USER"      # keep the timer running when nobody is logged in
```

## How a release flows

1. A pull request is merged to `main`.
2. CI runs the tests and lint, builds and smoke-tests the image, then **publishes**
   `ghcr.io/<owner>/shadowcommit:main` (also `:latest` and the immutable `:sha-<short>`) for
   `linux/amd64` and `linux/arm64`.
3. Within about two minutes the host's timer runs `deploy/update.sh`:
   - same image as the running one → nothing happens;
   - new image → recreate, wait up to 90 s for the health check;
   - unhealthy → the previous image is restored automatically.
4. `GET /healthz` reports the commit that is running (`"version"`).

`update.sh` exit codes: `0` up to date or deployed, `1` deploy failed and was rolled back, `2` deploy
failed and could not be rolled back (needs a person).

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

Pin a known-good image (a `sha-<short>` tag from the registry) and stop the timer so it does not
move forward again:

```bash
systemctl --user disable --now shadowcommit-update.timer
cd ~/shadowcommit/deploy
SHADOWCOMMIT_IMAGE=ghcr.io/<owner>/shadowcommit:sha-<short> docker compose up -d app
```

Re-enable the timer when the fix is merged.

## Data and backups

SQLite lives in the `shadowcommit-data` volume (mounted at `/data`) and Caddy's certificates in
`caddy-data`. A bind mount for `/data` is not writable by the app's non-root user until you
`chown 65532:65532` it. Backups are tracked separately (issue #41): until they exist, treat the
volume as the only copy.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Browser cannot connect | Ports 80/443 not open in the cloud rules or the host firewall |
| Certificate error right after the first start | DNS not pointing at the host yet; Caddy retries on its own, see `docker logs shadowcommit-caddy` |
| App container restarts, log says it refuses to boot | `JWT_SECRET` missing or still the default, or `DEV_LOGIN` set |
| Login with GitHub fails | OAuth callback URL or `*_BASE_URL` does not match the public https origin |
| `update.sh` exits 2 | The new image is unhealthy and the rollback also failed: inspect `docker logs shadowcommit-app` |
| Update never happens | Package is still private (the host cannot pull) or the timer is not enabled |
