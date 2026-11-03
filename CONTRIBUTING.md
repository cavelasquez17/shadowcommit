# Contributing to ShadowCommit

Thanks for wanting to help. This is a small project, so the process is short, but it is strict about
two things: **every change has a test**, and **every pull request does one thing**.

## Find something to work on

- The plan lives in the open [issues](https://github.com/cavelasquez17/shadowcommit/issues), starting from
  the [roadmap](https://github.com/cavelasquez17/shadowcommit/issues/14).
- Comment on an issue before starting so two people do not build the same thing.
- Bugs are welcome with steps to reproduce. For anything security-related, please do not post exploit
  details in a public issue; reach the maintainer through their GitHub profile first.

## Set up

You need Go (see `api/go.mod`), Node 20+ and, optionally, Docker.

```bash
# API
cd api
DEV_LOGIN=true go run ./cmd/api -seed-demo     # http://localhost:8080 with demo data

# Web (another terminal)
cd web
npm ci
npm run dev                                     # http://localhost:5173, proxies /api
```

The full list of commands is in the [README](README.md#quick-start).

## Before you open a pull request

Run what CI runs:

```bash
cd api && gofmt -l . && go vet ./... && go test ./... -race
golangci-lint run ./...                         # same pinned version as CI: v2.14.0
cd ../web && npm run lint && npm test && npm run build
```

If you touched the container image or the `deploy/` directory, also run `scripts/smoke-container.sh` and
`deploy/test/test-update.sh`.

## How we work

- **Branch** from an up-to-date `main`: `feat/…`, `fix/…`, `refactor/…`, `test/…`, `docs/…`, `ci/…`,
  `chore/…` (kebab-case).
- **Commits** follow [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): subject`,
  imperative and lower case, with a body that explains *why*. Keep each step and each fix in its own commit.
- **Tests first** for behaviour changes. A bug fix includes a regression test that fails without the fix.
  Break the behaviour on purpose once to check that the suite really notices.
- **One topic per pull request.** Fill in the pull request template: what changed, how it was tested and
  how you verified it. Link the issue with `Closes #N`.
- **CI must be green** before merging. We merge with a merge commit (no squash) so the history stays readable.
- **No secrets, personal data or real credentials** anywhere: code, tests, docs, issues or screenshots.
  Use the fictional demo data.

## The project's voice

ShadowCommit is satire that punches **up** at executives and AI hype, never down at people who lost their
jobs. Read the existing copy before writing new user-facing text, and put every UI string through i18n with
keys in all three locale files (`en`, `es`, `pt`).

## Working with an AI assistant

Contributions made with an AI coding assistant are welcome. Point it at [AGENT.md](AGENT.md), which holds
the project's rules (protected files, the review loop, testing expectations). You are responsible for what
you submit: read the diff, run the checks and make sure the tests prove something.

## License

By contributing you agree that your work is released under the [MIT License](LICENSE).
