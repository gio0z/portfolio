# Portfolio — Regio Dani Pangestu

Personal engineering portfolio and a small content-publishing pipeline behind it.

The site is a prerendered static build served by a single Go binary. That binary
also exposes the JSON API, the authenticated admin area, the Design Lab, and the
MCP endpoint agents use to submit design work for review.

## Stack

| Layer | Choice | Why |
|---|---|---|
| Public pages | Astro 7 (static output) | Real HTML per route: own title, description, canonical, and sitemap entry, without JavaScript |
| Interactive parts | React 19 islands | Hydrate only where a user actually interacts |
| Server | Go, standard library `net/http` | One binary serves the API, the admin app, and the static build |
| Registry | SQLite (`modernc.org/sqlite`) | Pure Go, no CGO |
| Admin auth | GitHub OAuth + signed session cookies + CSRF | Single-owner allowlist, fail-closed in production |

No frontend framework at runtime for content: `/about`, `/work`, `/work/<id>`,
and `/services` ship **zero JavaScript**.

## Layout

```
main.go                     entrypoint: flags, signals, listen
internal/app/               composition, config, static file serving
internal/adminapi/          admin JSON API (session-gated)
internal/adminauth/         GitHub OAuth, sessions, CSRF, step-up approval
internal/publicapi/         public Lab read endpoints
internal/publishing/        review state machine, SQLite registry, artifact store
internal/preview/           preview bundle runner
internal/deployment/        immutable version paths + atomic pointer swap
internal/mcppublisher/      MCP server: agent-facing tools
pkg/api/                    public portfolio API + the authored content
tools/contentexport/        exports that content for the static build
frontend/                   Astro site + React islands
e2e/                        end-to-end publishing flow test
docs/                       specs, plans, runbooks, threat model
```

## Content has one source of truth

`pkg/api/content.go` holds the profile, the six projects, and the four skill
categories. The public API serves them, and the static build derives its content
file from the same Go values:

```bash
go run ./tools/contentexport/main.go > frontend/src/content/portfolio.json
```

`npm run build` runs this as its `prebuild` step. `frontend/src/content/portfolio.json`
is committed but generated — **never edit it by hand**. Edit the Go content and
rebuild. The verification gate fails if the committed file is stale.

This exists because the project requires the static build to render without a
backend: a build that fetched `/api/*` would need a running server, so the
content is a build input instead.

## Getting started

```bash
# frontend
cd frontend
npm ci

# server (development: admin auth is relaxed, registry is in-memory)
cd ..
APP_ENV=development go run . -port 8080 -dist ./frontend/dist
```

`frontend` and `APP_ENV=development` are for local work only. In production the
server refuses to start unless every secret and origin is configured; see
`docs/runbooks/admin-mcp-lab-deployment.md`.

### Building the site

```bash
cd frontend
PORTFOLIO_PUBLIC_ORIGIN=https://your-domain.example npm run build
```

`PORTFOLIO_PUBLIC_ORIGIN` is **required for a build**. It becomes every page's
canonical URL and the sitemap origin, so the build fails rather than guessing: a
wrong canonical tells search engines the real pages live elsewhere. Development,
tests, and type-checking do not need it.

## Verification

```bash
bash scripts/run-publishing-e2e.sh
```

Runs the whole contract: `go test -race ./...`, `go vet ./...`, the frontend
suite, the Astro build, `astro check`, `tsc`, `oxlint`, the static output
assertions (every page exists with a distinct title, canonical, and description;
no JavaScript on content pages; sitemap coverage), content freshness, and the
end-to-end publishing flow.

The same script runs in CI (`.github/workflows/verify.yml`), so a green badge and
a green local run mean the same thing.

### Frontend checks

```bash
cd frontend
npm test              # vitest
npm run check         # astro check
npm run lint          # oxlint
npm run build         # content export + astro build + tsc
```

## Configuration

`.env.example` documents every variable and is the only tracking point for them.
`APP_ENV` (or `ENVIRONMENT`) defaults to **production**, which fails closed:
a missing signing key, OAuth credential, owner allowlist entry, approval
verifier, public origin, registry path, artifact root, or MCP token secret aborts
startup with an error naming the variable — never its value.

Set `APP_ENV=development` only outside production. It also disables secure
cookies, so admin sessions would travel over plain HTTP.

## Deployment

The site runs on `hp-server-linux` as this Go binary behind an existing
Cloudflare tunnel on `ginapps.my.id`. Deployment is pull-based: a systemd timer
checks `origin/main` each minute and runs `deploy/deploy.sh`, so GitHub never
connects to the server.

See **`deploy/README.md`** for the full procedure, including the hostnames, the
one-time server bootstrap, and the operations commands.

## Documentation

| Document | Contents |
|---|---|
| `CONTEXT.md` | Domain vocabulary and the system seams |
| `docs/runbooks/admin-mcp-lab-deployment.md` | Deploy order, environment variables, verifier wiring |
| `docs/runbooks/admin-mcp-lab-rollback.md` | Rollback procedure |
| `docs/security/admin-mcp-lab-threat-model.md` | Threat model |
| `docs/superpowers/specs/` | Design specs |
| `docs/superpowers/plans/` | Implementation plans, including the Astro migration |
