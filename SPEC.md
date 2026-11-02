# ShadowCommit — Product & Technical Specification

> **Purpose**: This document gives any AI model (or human developer) enough context to understand ShadowCommit's vision, architecture, data model, and roadmap to continue development without losing the thread. It is the single source of truth for product decisions.

## 1. Vision

**One line**: A dark-humor social forum where developers document how AI (or corporate AI-adjacent decisions) ended their careers — in their own language, with their own repos, and a community that gets the joke.

**The problem**: Developers displaced by AI-driven restructuring have nowhere to go. Reddit threads are ephemeral. LinkedIn demands a polished façade. Discord servers are fragmented and unsearchable. There is no dedicated, multilingual, data-rich community built around the *identity* of being replaced.

**The insight**: The pain is universal but the conversation is scattered. A platform that treats career death as a first-class data model — cause of death, epitaph, fired-at timestamp, employment status — can aggregate patterns, build community through shared grievance, and eventually monetize through job placement, sponsorships, and premium features.

## 2. Market originality

**Confirmed gap** (researched 2026-09-03):

- **r/Layoffs**, **r/singularity**, **r/webdev** — scattered threads, no dedicated platform, no structured data, no identity model
- **LinkedIn** — opposite tone (polished, professional, anti-humor)
- **Discord communities** — fragmented, ephemeral, unsearchable, no data aggregation
- **LayoffLink**, **LinkedOut** — job boards, not communities
- **No existing product** combines: career-obituary identity + multilingual forum + GitHub repo linking + aggregate "Wall of the Fallen" dashboard + satirical neo-brutalist design

ShadowCommit is original. The closest analogues are subreddit threads, not products.

## 3. Target audience

- **Primary**: Developers (frontend, backend, full-stack, DevOps, mobile) who were laid off, restructured, or had their role materially changed by AI adoption — globally, especially LATAM (es/pt) and Anglophone markets
- **Secondary**: Still-employed developers in denial ("Copium mode"), hiring managers looking for displaced talent, tech journalists tracking AI displacement trends
- **Tertiary**: Curious onlookers, meme consumers, the tech-curious public

## 4. Core product pillars

### 4.1 Career obituary identity
Every user declares:
- **Cause of death** (enum: `copilot_did_it_faster`, `ceo_saw_a_demo`, `shadow_work_exposed`, `replaced_by_a_prompt`, `replaced_by_an_agent_swarm`, `synergistic_restructuring`, `still_alive_somehow`, `left_before_they_could_fire_me`)
- **Seniority** at time of death (intern → CTO, plus `shadow` and `freelance_survivor`)
- **Epitaph** (≤180 chars, user-written)
- **FiredAt** timestamp (nullable — "not yet, keep refreshing")
- **IsEmployed** bool (the Wall of the Fallen depends on this being `false`)
- **Current role** (what they're doing now, if anything)

### 4.2 Multilingual forum
- **9 post locales** (en, es, pt, fr, de, ja, hi, zh, ar) — filter the feed by language
- **3 UI locales** (en, es, pt) — full interface translation
- Posts have a `language` field; the feed filters by it. Nobody translates suffering well.

### 4.3 GitHub-native onboarding
- OAuth pulls the user's GitHub identity and repos
- Repos can be marked `WrittenByAI` — the confession field
- Repos link from the profile page: "The code that outlived your contract"

### 4.4 Wall of the Fallen
Aggregate dashboard powered by `GET /api/stats`:
- Total users, total fired, total survivors, total reactions
- Breakdown by cause of death, seniority, country, language
- All charts are CSS bars — no charting library for four bar lists
- Recently fallen users with epitaphs

### 4.5 Reaction vocabulary
Seven emoji-shaped screams, unique per (user, target):
- `f` 💀 — press F to pay respects
- `relate` 🫠 — too real
- `laugh` 😂
- `robot` 🤖 — the machine did this
- `copium` 🫡 — it will be fine
- `brutal` 🔥 — no notes
- `lawyer` ⚖️ — you should sue

### 4.6 Social graph
- Follow/unfollow users
- @mentions in threads and posts (autocomplete via user search)
- Notifications (mention, reply, follow, reaction)
- Invites (invite-only gating configurable via `INVITE_ONLY` env var)

### 4.7 Copium mode
Per-device UI toggle (localStorage `sc_copium`). Flips the site's emotional register from "reality" to "copium" via `data-copium` attribute on `<html>`. Not per-account — it's a coping mechanism, not a profile setting.

## 5. Data model

12 GORM models in `api/internal/models/models.go`:

| Model | Purpose | Key fields |
|-------|---------|------------|
| User | A survivor, casualty, or person in denial | PublicID, Username, IsEmployed, CauseOfDeath, Seniority, Epitaph, FiredAt, Reputation, Role |
| Repo | Code that outlived its author's employment | UserID, Name, URL, Language, Stars, WrittenByAI |
| Badge | Participation trophies | Slug, Label, Emoji, Color; many2many with User |
| Category | Thread grouping (10 seeded categories) | Slug, Name, Emoji, Color, SortOrder, ThreadCount |
| Tag | Free-form thread labels | Slug, Label, UsageCount; many2many with Thread |
| Thread | One story of obsolescence | AuthorID, CategoryID, Title, Body, Language, Sarcasm, HotScore, NSFWCorporate |
| Post | Reply or eulogy (1 level nesting via ParentID) | ThreadID, AuthorID, ParentID, Body, IsAnswer, EditedByAI |
| Reaction | Emoji-shaped scream (unique per user+target+kind) | UserID, Kind, TargetType, TargetID |
| Mention | @tag notification record | MentionedUserID, ByUserID, ThreadID, PostID |
| Invite | Golden ticket to the funeral | Code, CreatedByID, UsedByID, ExpiresAt |
| Notification | Bell icon content | UserID, Kind, Title, Body, Link, IsRead |
| Follow | Social graph edge | FollowerID, FollowingID (unique pair) |

### Key invariants
- `IsEmployed` has **no `default:` GORM tag** — GORM omits zero-valued fields from INSERTs when a default exists, which would silently resurrect every dead user. Test: `TestUserIsEmployedPersistsFalse`.
- All external IDs use `NewPublicID(prefix)` → `prefix_<16 base32 chars>`. Auto-increment IDs never leave the API.
- Soft deletes on User, Thread, Post via `gorm.DeletedAt`.
- `SkipDefaultTransaction: true` in GORM config — multi-write operations wrap in `db.Transaction()` explicitly.

## 6. API surface

Base path: `/api`. Auth via HTTP-only cookies (JWT). `AuthOptional` middleware lets public responses include "did I react to this" without forcing login.

```
# Auth
GET    /auth/github              → redirect to GitHub OAuth
GET    /auth/github/callback     → OAuth callback, upsert user, redirect to web
POST   /auth/dev-login           → dev-only passwordless login
POST   /auth/logout              → clear session
GET    /auth/me                  → current user (optional auth)

# Reference
GET    /meta                     → enum vocabulary + flags
GET    /categories               → all categories
GET    /tags                     → all tags
GET    /stats                    → Wall of the Fallen aggregate data
GET    /invites/check?code=X     → validate invite code

# Threads
GET    /threads                  → feed (sort=hot|new|top|active, filters, pagination)
GET    /threads/:id              → single thread with posts
POST   /threads                  → create (auth required)
PATCH  /threads/:id              → update (owner only)
DELETE /threads/:id              → soft delete (owner only)
POST   /threads/:id/react        → toggle reaction (auth required)
POST   /threads/:id/posts        → create reply (auth required)

# Posts
PATCH  /posts/:id                → update (owner only)
DELETE /posts/:id                → soft delete (owner only)
POST   /posts/:id/react          → toggle reaction (auth required)

# Users
GET    /users                    → member directory (filters: country, seniority)
GET    /users/search?q=X         → @mention autocomplete
GET    /users/:username          → public profile
POST   /users/:username/follow   → toggle follow (auth required)
PATCH  /profile                  → update own profile (auth required)
POST   /profile/repos            → add repo (auth required)
DELETE /profile/repos/:id        → remove repo (auth required)

# Social
GET    /notifications            → inbox (auth required)
POST   /notifications/read       → mark all read (auth required)
GET    /mentions                 → mentions (auth required)
GET    /invites                  → list own invites (auth required)
POST   /invites                  → create invite (auth required)
```

## 7. Frontend routes

```
/              Landing (hero, stats, how-it-works, hot threads)
/feed          Thread feed (sort, filter, infinite scroll)
/t/:id         Thread detail (posts, reactions, reply)
/new           New thread composer (auth required)
/u/:username  Public profile (repos, badges, follow)
/members       Member directory
/wall          Wall of the Fallen dashboard
/onboarding    First-login profile setup (auth required)
/invites       Invite management (auth required)
/join          Join with invite code
/notifications Notification inbox (auth required)
/threads       → redirect to /feed
```

## 8. Design language

Neo-brutalist, loud, unapologetically multicolour. Source of truth: `web/src/index.css`.

- **Background**: `--color-void` (#0a0a0f) with a faint 44px terminal grid
- **Structure**: hard 2px borders, offset box-shadows (4px 4px 0 0), zero rounded corners, zero gradients
- **Color is semantic**: toxic lime = CTA, blood red = fallen, cyber cyan = info, grape purple = mentions, butter yellow = highlights, slime green = survivors, tangerine = hot, bubble pink = emphasis
- **Typography**: Archivo Black for display (uppercase, tight tracking), JetBrains Mono for body
- **Animations**: glitch (RGB split text-shadow), flicker (opacity), marquee (ticker scroll)
- **Primitives**: `.slab`, `.slab-hover`, `.btn` (primary/danger/ghost), `.field`, `.chip`, `.stamp`, `.prose-sc`

## 9. Monetization roadmap (target: $3,000+/month)

### Phase 1: Community growth (current)
- Invite-only beta to build scarcity and quality
- Seed with demo personas and threads
- Focus on organic sharing — the tone IS the marketing
- **Revenue: $0** — growth phase

### Phase 2: Job board + sponsorships (months 2-4)
- Companies post real openings with salary transparency
- "Community-reported" layoff history per company
- Sponsored threads from hiring companies (clearly labeled)
- **Revenue target: $500-1,000/month** from job posts + sponsorships

### Phase 3: Premium profiles + features (months 4-6)
- "Open to Work" premium profiles (not LinkedIn-style — community-style)
- Portfolio hosting (link repos, mark AI-written parts)
- Advanced stats dashboard (personal analytics)
- Ad-free experience
- **Revenue target: $1,500-2,000/month** from premium subscriptions ($5-9/mo)

### Phase 4: Data products + API (months 6-12)
- Anonymized layoff pattern data (sold to journalists, researchers, VCs)
- API access for tech journalists tracking AI displacement
- "Verified layoff log" — crowdsourced, anonymized dataset
- **Revenue target: $3,000+/month** from data licensing + API + premium + jobs

### Phase 5: Ecosystem (year 2+)
- Freelance marketplace (displaced devs → companies needing short-term work)
- "AI replacement audit" consulting (companies pay to assess their AI exposure)
- Merch (the tone sells t-shirts)
- Annual "State of the Developer Apocalypse" report
- **Revenue target: $10,000+/month**

## 10. Roadmap — next features

### Immediate (next session)
- [ ] **Search API**: full-text search across threads and posts (currently only title/body LIKE)
- [ ] **Thread pinning/locking** for admin/moderator role (model fields exist, no admin UI)
- [ ] **Email notifications** (currently only in-app bell icon)
- [x] **Rate limiting** on auth and write endpoints (per-key fixed window, `internal/handlers/ratelimit.go`)
- [ ] **Image uploads** in thread bodies (currently markdown text only)

### Short-term (weeks)
- [x] **Job board** — `JobPosting` model with company, salary, remote; list/detail/create/sponsor/filled/delete endpoints (layoff history still pending, see company profiles)
- [ ] **Company profiles** — track layoff history, hiring patterns, "community-reported" data
- [ ] **Moderation tools** — report/flag system, admin dashboard, content moderation queue
- [ ] **SEO optimization** — server-side rendering for thread pages, OpenGraph tags
- [ ] **PWA** — offline reading, push notifications, installable

### Medium-term (months)
- [ ] **Premium subscriptions** — Stripe integration, premium profile features
- [ ] **Data dashboard** — public aggregate stats page (anonymized), embeddable widgets
- [ ] **API v2** — versioned endpoints, rate-limited public API for journalists
- [ ] **Mobile app** — React Native or PWA-first, push notifications
- [ ] **Multilingual expansion** — add fr, de, ja, hi, zh, ar UI locales (post locales already supported)

### Long-term
- [ ] **Freelance marketplace** — connect displaced devs with companies
- [ ] **AI replacement audit** — B2B consulting product
- [ ] **Annual report** — "State of the Developer Apocalypse"
- [ ] **Merch store** — t-shirts, stickers, mugs with the site's best one-liners

## 11. Technical decisions log

| Decision | Rationale | Date |
|----------|-----------|------|
| Gin over Fiber/Echo | Mature, middleware ecosystem, GORM-friendly | Initial |
| GORM over sqlx | AutoMigrate, soft deletes, hooks — speed of development | Initial |
| SQLite for dev, Postgres for prod | Zero-config local dev, proven prod path | Initial |
| JWT in cookies, not localStorage | XSS-resistant, no token babysitting in frontend | Initial |
| React 19 + Vite 8 | Latest stable, fast HMR, no CRA baggage | Initial |
| Tailwind v4 (`@theme` in CSS) | No config file, CSS-first, design tokens in one place | Initial |
| No charting library | CSS bars for 4 bar lists — a library is not worth the weight | Initial |
| `NewPublicID` over UUID | Opaque, URL-safe, non-sequential, prefixed for readability | Initial |
| Invite-only beta | Scarcity + quality control during growth phase | Initial |
| No service layer (yet) | Handlers + helpers suffice at current scale; introduce when complexity demands | Initial |
| Go 1.22 range loops | `for i := range n` is cleaner; module requires 1.22+ | 2026-09-03 |
| Store test suite (19 tests) | Guard GORM footguns, model relationships, constraints, soft deletes | 2026-09-03 |

## 12. Environment variables

| Var | Default | Purpose |
|-----|---------|---------|
| `PORT` | `8080` | API port |
| `APP_ENV` | `development` | `production` enables safety checks |
| `DB_DRIVER` | `sqlite` | `sqlite` or `postgres` |
| `DATABASE_URL` | `shadowcommit.db` | DB connection string |
| `JWT_SECRET` | `dev-secret-rotate-me-or-get-fired` | JWT signing key (must change in prod) |
| `APP_BASE_URL` | `http://localhost:8080` | API base URL for OAuth callback |
| `WEB_BASE_URL` | `http://localhost:5173` | Web base URL for CORS + OAuth redirect |
| `CORS_ORIGINS` | `WEB_BASE_URL,http://localhost:4173` | Comma-separated allowed origins |
| `GITHUB_CLIENT_ID` | (empty) | GitHub OAuth app client ID |
| `GITHUB_CLIENT_SECRET` | (empty) | GitHub OAuth app client secret |
| `DEV_LOGIN` | `true` in dev | Enable passwordless dev login (never in prod) |
| `INVITE_ONLY` | `false` | Gate registration behind invite codes |
| `SEED_DEMO` | `false` | Seed demo personas on boot |
| `VITE_API_BASE` | `/api` | Frontend API base URL |

## 13. Glossary

- **Wall of the Fallen**: Aggregate dashboard showing displacement statistics
- **Cause of death**: Enum describing how a career ended (AI, restructuring, etc.)
- **Epitaph**: User-written ≤180 char memorial for their career
- **Copium mode**: Per-device UI toggle that softens the site's tone
- **Shadow work**: Being employed but doing work that isn't officially recognized
- **NSFW Corporate**: Flag for threads that name specific companies/people
- **Sarcasm score**: 0-100 self-declared sarcasm rating on threads
- **Hot score**: HN-style decay algorithm (reactions + recency) for feed sorting
- **Respects**: Reactions, especially the 💀 "press F" reaction
