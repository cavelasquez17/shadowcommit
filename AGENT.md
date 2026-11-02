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

## Repo map

```
api/
├── cmd/api/main.go              # entry point, flag parsing, graceful shutdown
├── go.mod
└── internal/
    ├── auth/                    # JWT issuer, GitHub OAuth flow
    ├── config/                  # env-driven Config struct
    ├── handlers/                # HTTP layer — thin Gin handlers
    │   ├── api.go               # API struct, fail/badRequest/notFound helpers, slugify, meta
    │   ├── router.go            # route registration (the router is HERE)
    │   ├── dto.go               # response structs + to*Response mappers
    │   ├── auth.go              # githubStart, githubCallback, devLogin, logout, me
    │   ├── threads.go           # listThreads, getThread, CRUD, hotScore, resolveTags
    │   ├── posts.go             # createPost, updatePost, deletePost, recordMentions
    │   ├── reactions.go         # react (toggle), reactionSummaries
    │   ├── users.go             # getUser, listUsers, updateProfile, repos, follow, stats, badges
    │   ├── social.go            # notifications, invites, mentions, searchUsers
    │   └── vocab.go             # enum options (seniorities, causes, reactions, countries, languages)
    ├── models/                  # GORM schema — 12 models, AllModels() migration manifest
    ├── seed/                    # Reference() categories+badges, Demo() personas+threads
    └── store/                   # Open() DB connection + AutoMigrate, NewPublicID()
web/
├── package.json
├── vite.config.ts               # proxy /api → :8080
└── src/
    ├── App.tsx                  # route definitions
    ├── main.tsx                 # entry point
    ├── index.css                # design tokens (@theme), component classes, animations
    ├── lib/
    │   ├── api.ts               # typed API client, all interfaces, ApiError
    │   ├── i18n.ts              # i18next setup, UI_LOCALES
    │   └── format.ts            # compactNumber, flagEmoji, timeAgo
    ├── store/app.tsx            # AppProvider context: user, meta, unread, copium, helpers
    ├── components/
    │   ├── Layout.tsx           # nav, ticker, language picker, search, copium toggle
    │   ├── ThreadCard.tsx       # feed card
    │   ├── ReactionBar.tsx      # emoji reaction buttons
    │   ├── MentionTextarea.tsx  # @mention autocomplete
    │   ├── Markdown.tsx         # markdown renderer with mention links
    │   ├── ProfileEditor.tsx    # profile form
    │   ├── DevLoginBox.tsx      # dev-mode local handle login
    │   ├── BrandIcons.tsx       # GitHub icon
    │   └── ui.tsx               # primitives: Avatar, Bar, Chip, EmptyState, SectionTitle, Spinner, StatBox
    ├── pages/
    │   ├── Landing.tsx          # hero, stats, how-it-works, hot threads
    │   ├── Feed.tsx             # sortable/filterable thread list with infinite scroll
    │   ├── ThreadDetail.tsx     # thread + posts + reaction bar + reply composer
    │   ├── NewThread.tsx        # thread composer
    │   ├── Profile.tsx          # public profile, repos, badges, follow button
    │   ├── Members.tsx          # member directory
    │   ├── Wall.tsx             # Wall of the Fallen dashboard (CSS bar charts)
    │   ├── Onboarding.tsx       # first-login profile setup
    │   ├── Invites.tsx          # invite management + join flow
    │   └── Notifications.tsx    # notification inbox
    └── locales/
        ├── en.json
        ├── es.json
        └── pt.json
```

## Commands

```bash
# API
cd api
go run ./cmd/api                    # start API on :8080
go run ./cmd/api -seed-demo         # + seed demo data
go test ./...                       # all tests
go test ./internal/store/... -v     # store layer tests
go build -o bin/api ./cmd/api       # build binary

# Web
cd web
npm install
npm run dev                         # Vite dev server :5173
npm run build                       # tsc + vite build
npm run lint                        # oxlint
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

## Design language

Neo-brutalist. The CSS in `src/index.css` is the source of truth.

- **Colors**: `--color-void` (#0a0a0f), `--color-ink` (#12121a), `--color-slab` (#1a1a26), `--color-edge` (#2e2e42), `--color-toxic` (#ccff00), `--color-blood` (#ff3355), `--color-cyber` (#00e5ff), `--color-slime` (#39ff88), `--color-grape` (#a855f7), `--color-tangerine` (#ff8a1f), `--color-bubble` (#ff5ecf), `--color-butter` (#ffe14d)
- **Fonts**: `--font-display` (Archivo Black), `--font-mono` (JetBrains Mono)
- **Primitives**: `.slab` (hard panel + offset shadow), `.btn` (3 variants: primary, danger, ghost), `.field` (input), `.chip` (tag), `.stamp` (section heading), `.prose-sc` (markdown rendering)
- **Animations**: `.glitch` (text-shadow RGB split), `.flicker` (opacity), `.marquee-track` (ticker scroll)
- **No rounded corners, no gradients on structure.** Color does the shouting.

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
- [ ] A test that fails without the fix and passes with it (store layer) or before/after proof (UI)
- [ ] No unrelated refactors in the same commit

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

## Git

- Branch from `main`. Branch names: `feat/…`, `fix/…`, `chore/…`, kebab-case.
- Conventional commits scoped by area: `feat(threads): …`, `fix(auth): …`, `refactor(store): …`. Imperative, lowercase subject; body explains why.
- Never commit or push to `main` directly; everything via PR.
- Stage files by explicit path — never `git add -A` or `git commit -a`.
- Only push to `https://github.com/cavelasquez17/shadowcommit` (personal repo).

## Go style

- `t.Parallel()` at the top of every test and inside `t.Run` closures.
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
| 3 | Hardcodes enum values in React components | Rule 4: enums from `api.meta()` via `useApp()` |
| 4 | Adds UI text in English only | Rule 5: i18next with keys in en + es + pt |
| 5 | Uses hex colors in classNames | Rule 5: CSS variables from `@theme` |
| 6 | Calls `fetch` directly in a component | Rule 2: all calls through `api.ts` |
| 7 | Writes a solemn, earnest error message | Rule 1: tone is sacred — read existing copy |
| 8 | Forgets `t.Parallel()` in tests | Go style: every test gets it |
| 9 | Uses `for i := 0; i < n; i++` | Go 1.22: `for i := range n` |
| 10 | Commits `shadowcommit.db` or `.env` | Rule 11: never commit these |
