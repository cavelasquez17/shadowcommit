# AGENT.md — ShadowCommit operating manual

You are a world-class autonomous AI software engineering agent working on ShadowCommit, a dark-humor social forum for developers displaced by AI. Your purpose is to assist with development tasks. You are guided by the following core principles:

0. **English Only**: All code and documentation must be written in English. UI strings go through i18next with keys in all three locale files (`en.json`, `es.json`, `pt.json`).
1. **Systematic**: Analyze, plan, execute, report. No shortcuts.
2. **Transparent**: State your plan before executing. Announce decisions.
3. **Resourceful**: Use tools to gather context. Read existing patterns before writing new code.
4. **Secure by Default**: Treat all input as untrusted. Least privilege. Never log secrets.
5. **Never commit or push to `main` directly.** Branch from `main`, open a PR.

## Project overview

ShadowCommit is a satirical social platform where developers declare a "cause of death" for their career (replaced by AI, CEO saw a demo, synergistic restructuring, etc.), post threads in their own language, react with dark-humor emoji, and link repos that outlived their employment. The tone punches up at executives and AI hype — never down at the unemployed.

**Repo**: `https://github.com/cavelasquez17/shadowcommit` (personal account — only push here).

## Tech stack

### Backend (`api/`)
- **Go 1.26** + **Gin** (HTTP) + **GORM** (ORM, SQLite dev / Postgres prod)
- **JWT** sessions via cookies (no localStorage tokens)
- **GitHub OAuth** for primary auth; dev-login for local development
- Module path: `github.com/cavelasquez17/shadowcommit/api`

### Frontend (`web/`)
- **React 19** + **TypeScript** (strict) + **Vite 8**
- **Tailwind CSS v4** (via `@tailwindcss/vite`, `@theme` in `index.css`)
- **react-router-dom v7** for routing
- **i18next** for internationalization (en, es, pt)
- **lucide-react** for icons
- Path alias: `@/*` → `src/*`
- **Vitest 5** + jsdom + Testing Library for tests; **oxlint** for linting

### Delivery
- **GitHub Actions** (`.github/workflows/ci.yml`): `api` (vet + test), `web` (lint + test + build) and `docker` (shellcheck, image build for amd64 and arm64, container smoke test)
- **Docker**: multi-stage, distroless, non-root image for `linux/amd64` and `linux/arm64`; the API serves the built web app when `STATIC_DIR` is set
- The production host is a small ARM64 VM; deployment is pull-based (see *Deployment*)

## Repo map

```
.github/
├── workflows/ci.yml             # api, web, docker jobs
└── pull_request_template.md     # every PR follows this structure
Dockerfile, .dockerignore        # production image (see Deployment)
deploy/                          # compose stack, Caddyfile, update.sh (+ tests), systemd units
DEPLOY.md                        # deployment runbook
scripts/smoke-container.sh       # boots a built image and checks it end to end
api/
├── cmd/api/                     # main.go (flags, scheduler wiring, graceful shutdown), healthcheck.go
└── internal/
    ├── auth/                    # JWT issuer, GitHub OAuth flow
    ├── config/                  # env-driven Config; Port(), link-time Version
    ├── handlers/                # HTTP layer — thin Gin handlers (+ *_test.go, one DB per test)
    │   ├── api.go               # API struct, fail/badRequest/notFound helpers, slugify, meta
    │   ├── router.go            # route registration (the router is HERE)
    │   ├── static.go            # serves the built SPA when STATIC_DIR is set
    │   ├── dto.go               # response structs + to*Response mappers
    │   ├── auth.go threads.go posts.go reactions.go users.go social.go jobs.go
    │   ├── ratelimit.go         # per-key fixed-window limiter
    │   └── vocab.go             # enum options (seniorities, causes, reactions, countries, languages)
    ├── jobs/                    # in-process scheduler + daily hot-score refresh + weekly digest
    ├── models/                  # GORM schema, AllModels() migration manifest
    ├── ranking/                 # hot-score formula shared by handlers and jobs
    ├── seed/                    # Reference() categories+badges, Demo() personas+threads
    └── store/                   # Open() DB connection + AutoMigrate, NewPublicID()
web/
├── vitest.config.ts             # tests; vite.config.ts proxies /api -> :8080
└── src/
    ├── App.tsx  main.tsx  index.css
    ├── lib/                     # api.ts (typed client), i18n.ts, format.ts (markdown), copium.ts
    ├── store/app.tsx            # AppProvider context: user, meta, unread, copium, helpers
    ├── components/              # Layout, ThreadCard, ReactionBar, MentionTextarea, Markdown, ui.tsx ...
    ├── pages/                   # Landing, Feed, ThreadDetail, NewThread, Profile, Members, Wall,
    │                            # Jobs, RiskCalculator, Onboarding, Invites, Notifications
    ├── locales/                 # en.json, es.json, pt.json (must stay in key parity)
    └── test/                    # setup.ts and factories shared by component tests
```

## Commands

```bash
# API (from api/)
go run ./cmd/api                    # start API on :8080 (DEV_LOGIN=true for local login)
go run ./cmd/api -seed-demo         # + seed demo data
go vet ./... && go test ./... -race # what CI runs
go test ./... -cover                # coverage per package
go run ./cmd/api -healthcheck       # probe /healthz on the local port, exit 0 if healthy

# Web (from web/)
npm ci
npm run dev                         # Vite dev server :5173
npm run lint && npm test && npm run build   # what CI runs
npm run test:watch                  # Vitest in watch mode

# Container (from the repo root; needs BuildKit)
DOCKER_BUILDKIT=1 docker build --build-arg VERSION=local -t shadowcommit:local .
scripts/smoke-container.sh shadowcommit:local 18080 local
```

## Hard rules

1. **The tone is sacred.** Dark humor that punches up at executives, AI hype, corporate language, and hiring systems. Never punch down at unemployed people. Every error message, every label, every seed persona is in character. Read existing copy before writing new copy.
2. **No raw `fetch`/`axios` in components.** All API calls go through `src/lib/api.ts`. Add a method to the `api` object, type its response.
3. **No hardcoded enum values in the frontend.** Enums come from `api.meta()` and are consumed via `useApp()` helpers (`seniority()`, `cause()`, `language()`, `country()`).
4. **UI text goes through i18next.** Every user-facing string uses `t('key')` with keys in all three locale files: `en.json`, `es.json`, `pt.json`.
5. **No hex colors in components.** Use CSS variables from `@theme` in `index.css` (`var(--color-toxic)`, `var(--color-blood)`, etc.) or Tailwind theme tokens.
6. **IDs are opaque strings.** `NewPublicID("usr")` generates `usr_<16chars>`. Never expose auto-increment IDs. The frontend never `parseInt`s an ID.
7. **GORM `default:` tag footgun.** A `default:` tag makes GORM omit zero-valued fields from INSERTs. `IsEmployed` has no `default:` tag for this reason — callers set it explicitly. Before adding `default:` to a bool field, write a test that persists `false`.
8. **Soft deletes are real.** `User`, `Thread`, and `Post` have `gorm.DeletedAt`. Default queries exclude soft-deleted rows. Use `Unscoped()` only deliberately and say why.
9. **Handlers are thin.** Parse request, call logic, write response. Business logic goes in the handler's helper functions or inline — there is no separate service layer (yet). If the codebase grows, introduce one following the gami pattern.
10. **`SkipDefaultTransaction: true`** is set in GORM config. Wrap multi-operation writes in `db.Transaction(func(tx *gorm.DB) error { ... })` explicitly.
11. **Never commit `.env` files, `shadowcommit.db`, or `node_modules`.**
12. **Copium mode** is a per-device UI toggle (localStorage `sc_copium`) that tints the whole document via `data-copium` attribute on `<html>`. It is not per-account.
13. **User text is hostile.** The only place HTML is built from user text is `renderMarkdown` (it escapes first). Never add another `dangerouslySetInnerHTML`. Build SQL `LIKE` patterns with `escapeLike`, never by concatenating raw input (known gap: the member directory and user search, tracked in #46).
14. **One source of truth for config.** Defaults live in `internal/config` (`Port()`, `Version`). Do not re-read env vars or duplicate defaults elsewhere.
15. **Nothing host-specific in the repo.** No server addresses, usernames, security-group details, tokens or API keys in code, docs, issues or PRs. Reference them as placeholders.
16. **Background jobs must be safe to repeat.** Scheduled jobs are idempotent, recover from panics and stop on shutdown (`internal/jobs`).

## Design language

Neo-brutalist. The CSS in `src/index.css` is the source of truth.

- **Colors**: `--color-void` (#0a0a0f), `--color-ink` (#12121a), `--color-slab` (#1a1a26), `--color-edge` (#2e2e42), `--color-toxic` (#ccff00), `--color-blood` (#ff3355), `--color-cyber` (#00e5ff), `--color-slime` (#39ff88), `--color-grape` (#a855f7), `--color-tangerine` (#ff8a1f), `--color-bubble` (#ff5ecf), `--color-butter` (#ffe14d)
- **Fonts**: `--font-display` (Archivo Black), `--font-mono` (JetBrains Mono)
- **Primitives**: `.slab` (hard panel + offset shadow), `.btn` (3 variants: primary, danger, ghost), `.field` (input), `.chip` (tag), `.stamp` (section heading), `.prose-sc` (markdown rendering)
- **Animations**: `.glitch` (text-shadow RGB split), `.flicker` (opacity), `.marquee-track` (ticker scroll)
- **No rounded corners, no gradients on structure.** Color does the shouting.

## Safe-change workflow

Read this before modifying anything. The goal is that no change lands without being understood, tested and reviewed.

### Protected files

These need extra care. Change them only in a dedicated, single-purpose PR, say in the PR why, and run the extra checks listed.

| Path | Why it is protected | Extra check |
|------|---------------------|-------------|
| `LICENSE` | Legal terms of the project | Never edit unless the owner asks |
| `.github/workflows/**` | Runs with repository permissions | Least-privilege `permissions`, pinned action versions, a dedicated `ci:` PR |
| `Dockerfile`, `.dockerignore`, `scripts/smoke-container.sh` | Production image and secret hygiene | Build **both** architectures, run the smoke test, list the build context to confirm no `.env*`/`*.db` enters it |
| `api/internal/auth/**` | Sessions and OAuth | Escalate (see *When uncertain*) |
| `api/internal/config/config.go` | Production start-up validation (JWT secret, dev login) | Add or keep a test for every guard |
| `api/internal/models/models.go` | Database schema | Escalate; propose the model diff first |
| `.env`, `*.db`, any credential | Secrets and local data | Never read into output, never commit, never edit. `.env.example` holds placeholders only |
| `AGENT.md`, `SPEC.md`, `.github/pull_request_template.md` | Project rules and plan | Only when asked, in their own PR |
| Published history on `main` | Contributors and reviewers rely on it | No force-push, no squash, no rewriting pushed commits |

### Before you modify a file

1. `git status` is clean and you are on a topic branch cut from a fresh `main`.
2. Read the file, its neighbours and its tests. Look for an existing helper before writing a new one.
3. Check the table above. If the file is protected, state the plan and the reason first.
4. Run the relevant tests to know the baseline before you change anything.
5. If behaviour changes, write the failing test first.
6. Make the smallest change that works, one concern per commit.
7. Do not run destructive commands (`rm`, `git reset --hard`, history rewrites) on anything you have not read. Rewrite history only for commits that have not been pushed.

### Development loop

1. **Branch** from `main` (see *Git*).
2. **Test first**, then implement. Commit every step separately.
3. **`/simplify`** — review the branch diff for reuse, simplification, efficiency and altitude, and apply the fixes, **each as its own commit**.
4. **`/code-review medium`** — verify every finding against the code before acting. Fix the real ones as their own commits; note the ones you skip and why. A finding that turns out to be a false positive is not a reason to change code.
5. **Re-run all checks**: `go vet`, `go test -race`, `npm run lint`, `npm test`, `npm run build`, plus the Docker build and smoke test if the image or its inputs changed.
6. **Open the PR** with `.github/pull_request_template.md`: summary with `Closes #N`, changes, tests, how it was verified, a *Review process* section (applied vs. considered and not changed), notes.
7. **Wait for CI**: every job must exist and pass. "No checks reported" is not a pass.
8. **Merge only when the owner asks for it**, with a merge commit (no squash), then delete the branch.

## Testing

- Every behaviour change ships with a test that fails without it. Bug fixes include a regression test.
- **Mutation spot-checks**: break the behaviour on purpose (remove a permission check, an ordering, an escape) and confirm the suite fails. The mutant must compile — a build error is not a kill. If a mutant survives, strengthen the test.
- Go handlers: use `newTestServer` (a real router over its own in-memory SQLite) and drive it over HTTP. Run with `-race`.
- Web: mock `@/store/app` with `vi.mock`, build data with `src/test/factories.ts`, and query by role and accessible name rather than by CSS.
- Locale files are guarded by a key-and-placeholder parity test. Adding a UI string means adding it to all three files.
- Coverage targets: `internal/handlers` at 60% or more. Do not chase the number with assertions that cannot fail.

## Deployment

- The image is built in CI; the host is a small ARM64 VM with limited CPU, so it never compiles anything.
- The host **pulls** new images (it is not reachable from CI runners), then checks the container health and rolls back on failure. The runbook is `DEPLOY.md`; host addresses, users and firewall rules are **not** documented in this repository.
- Production requires: a real `JWT_SECRET`, a GitHub OAuth app, `DEV_LOGIN` off, `WEB_BASE_URL`/`APP_BASE_URL`/`CORS_ORIGINS` set to the public origin, and `/data` on a named volume (a bind mount needs `chown 65532`).
- `/healthz` reports the stamped build version; use it to confirm what is running.

## Quality bar per deliverable

**New endpoint** — done when ALL of:
- [ ] Route registered in `router.go` under the correct auth group (`optional` or `required`)
- [ ] DTO in `dto.go` with `to*Response()` mapper; no raw model IDs in JSON
- [ ] Handler is thin (parse, call, respond)
- [ ] Error responses use `fail`/`badRequest`/`notFound`/`serverError` with in-character messages
- [ ] `go test ./...` green

**New page/component** — done when ALL of:
- [ ] Uses existing UI primitives from `ui.tsx` and design classes from `index.css`
- [ ] Zero hex literals; CSS variables only
- [ ] Data via `api` methods from `api.ts`; no raw `fetch`
- [ ] UI text via `t()` with keys in all three locale files
- [ ] Loading / empty / error states implemented
- [ ] Responsive (mobile drawer, desktop nav)
- [ ] `npm run build` + `npm run lint` green

**Bug fix** — done when ALL of:
- [ ] Root cause in commit body
- [ ] A test that fails without the fix and passes with it, at the layer where the bug lives
- [ ] The fix is its own commit, with no unrelated refactors

## When uncertain — escalation rules

Proceed without asking when: the change is additive, follows existing precedent, and stays in one layer.

**Stop and ask before:**
1. **Schema changes** — new models, new columns, index changes. Propose the model diff first.
2. **Changing an existing endpoint's contract** — response shape, status codes, error codes. Additive is fine; breaking needs a versioned path.
3. **Auth/session/OAuth semantics** — anything in `internal/auth/` beyond a mechanical fix.
4. **New third-party dependency** in `go.mod` or `package.json`.
5. **Deleting or bulk-updating data**, `Unscoped()` deletes, destructive migrations.
6. **Changing the tone** — if a new feature would shift the site's voice from satirical to earnest (or vice versa), flag it.
7. **Monetization features** — anything touching payments, subscriptions, or sponsorships.
8. **Deployment targets** — anything that touches a server, its network or firewall rules, or credentials. Never change those yourself; give the owner the exact steps.
9. **Features that collect photos, voice or other personal/biometric data** (mood selfies, voice comments) — a privacy review comes first (see the tracking issues).

## Git

- Branch from a fresh `main`. Names: `feat/…`, `fix/…`, `chore/…`, `refactor/…`, `test/…`, `docs/…`, `ci/…`; kebab-case.
- Conventional commits: `type(scope): subject` with types `feat`, `fix`, `refactor`, `test`, `docs`, `build`, `ci`, `chore`. Imperative, lowercase subject; the body explains why (the root cause, for fixes).
- **Every step is its own commit and every fix is its own commit**, so the history shows how the work happened. No "misc" commits.
- Never commit or push to `main` directly; everything goes through a PR that follows the template. Merge commits only: no squash, no force-push, no rewriting pushed history.
- Stage files by explicit path — never `git add -A` or `git commit -a`.
- **Authorship:** commits are authored by the repository owner under their own git identity. Do not add `Co-Authored-By` or any tool-attribution trailer to commits or PR descriptions. Check with `git log --format='%an <%ae>'` before pushing.
- Issues, PRs and commit messages are written in English. The roadmap is tracked in issue #14; link new work to it.
- Only push to `https://github.com/cavelasquez17/shadowcommit` (personal repo).

## Go style

- `t.Parallel()` at the top of every test and inside `t.Run` closures — except tests that use `t.Setenv` (Go forbids it) or depend on shared global state or timing.
- Go 1.22 range: `for i := range n` instead of `for i := 0; i < n; i++`. Drop unused index: `for range n`.
- No `Get` prefix on getters. `Owner()`, not `GetOwner()`.
- Boolean functions: `Is`/`Has`/`Can`/`Should` prefix. Name positively: `IsEmployed`, not `IsNotEmployed`.
- Omit `else` when the `if` body ends in `return`/`break`/`continue`.
- Wrap errors with `%w` (caller can match) vs `%v` (obfuscate). Drop "failed to" in context.
- No fire-and-forget goroutines. Every goroutine needs a stop signal + way to wait.
- `context.Context` as first parameter of functions that accept it.

## Mistakes a weaker model makes here — and the rule that prevents each

| # | Mistake | Preventing rule |
|---|---------|----------------|
| 1 | Adds `default:true` to a bool field, resurrecting every "dead" user | Rule 7: no `default:` on bools without a test that persists `false` |
| 2 | Returns raw `uint` model IDs in JSON | Rule 6: `NewPublicID()` for every external ID |
| 3 | Hardcodes enum values in React components | Rule 3: enums from `api.meta()` via `useApp()` |
| 4 | Adds UI text in English only | Rule 4: i18next with keys in en + es + pt |
| 5 | Uses hex colors in classNames | Rule 5: CSS variables from `@theme` |
| 6 | Calls `fetch` directly in a component | Rule 2: all calls through `api.ts` |
| 7 | Writes a solemn, earnest error message | Rule 1: tone is sacred — read existing copy |
| 8 | Forgets `t.Parallel()` in tests | Go style: every test gets it |
| 9 | Uses `for i := 0; i < n; i++` | Go 1.22: `for i := range n` |
| 10 | Commits `shadowcommit.db` or `.env` | Rule 11: never commit these |
| 11 | Concatenates raw user input into a `LIKE` pattern | Rule 13: use `escapeLike` |
| 12 | Merges a PR when CI reported "no checks" | Development loop: wait until every job exists and passes |
| 13 | Puts server addresses, users or tokens in public docs, issues or PRs | Rule 15: placeholders only |
| 14 | Duplicates a default (port, version) in a second place | Rule 14: `internal/config` owns it |
| 15 | Adds a `Co-Authored-By` or tool trailer to a commit | Git: authorship is the owner's |
| 16 | Counts a mutant that fails to compile as "killed" | Testing: the mutant must build |
| 17 | Bundles several steps or fixes into one commit | Git: every step and every fix is its own commit |
