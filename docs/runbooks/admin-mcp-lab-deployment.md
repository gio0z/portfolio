# Admin / MCP / Design Lab Publishing — Deployment Runbook

Date: 2026-09-13. Implements the spec (`docs/superpowers/specs/2026-09-03-admin-mcp-lab-publishing-design.md`)
against the code in `main.go`, `internal/adminauth`, `internal/adminapi`,
`internal/publishing`, `internal/publicapi`, and `internal/preview`.

Every public surface carries: `Independent redesign concept. Not affiliated with or endorsed by the original company.`

## 1. Prerequisites

- Go toolchain matching `go.mod`; SQLite available via `modernc.org/sqlite` (pure Go, no CGO).
- A GitHub OAuth App; a 256-bit random session signing key; DNS for the
  Admin, portfolio, Lab, and preview origins (four distinct origins).

## 2. Environment variables

| Variable | Required | Default | Consumed by |
|---|---|---|---|
| `ADMIN_GITHUB_CLIENT_ID` | Yes (all envs fail without it at `NewService`) | — | `adminauth.LoadConfigFromEnv`, `exchangeCode`, `HandleLogin` |
| `ADMIN_GITHUB_CLIENT_SECRET` | Yes | — | `adminauth.LoadConfigFromEnv`, `exchangeCode` |
| `ADMIN_SESSION_SIGNING_KEY` | Yes (HMAC secret for session + CSRF tokens) | — | `signSessionToken/verifySessionToken`, `IssueCSRFToken` |
| `ADMIN_ALLOWED_GITHUB_LOGIN` | No | `gio0z` | `OwnerMiddleware`, `HandleCallback` allowlist check |
| `ADMIN_PUBLIC_ORIGIN` | Yes in production (pins OAuth `redirect_uri` to `<origin>/api/admin/auth/callback`) | `""` | `getCallbackURL`, post-login redirect |
| `APP_ENV` (fallback `ENVIRONMENT`) | No | `production` | `LoadConfigFromEnv`, `main.go` fail-closed branch |
| `PORT` | No | `8080` | `main.go` (`-port` flag overrides) |

GitHub endpoints default to `https://github.com/login/oauth/*` and
`https://api.github.com/user` (`NewService` defaults); override only via
explicit `Config` construction, not env.

Generate the signing key:

```bash
python3 -c "import secrets; print(secrets.token_hex(32))"
```

## 3. Pre-deploy checklist

1. `ADMIN_PUBLIC_ORIGIN` is `https://<admin-host>` (no trailing slash issues —
   code trims it) and matches the GitHub OAuth App callback URL
   `<origin>/api/admin/auth/callback` exactly.
2. Secrets are provisioned via the platform secret manager, never baked into
   images or committed files.
3. `APP_ENV=production` is set (anything else weakens cookies and enables the
   dev step-up bypass — see §5).

## 4. Migration order

There is exactly one registry migration, applied automatically by
`publishing.OpenRegistry`:

1. Create the data directories first (registry + artifact store root):
   `mkdir -p /var/lib/portfolio /data/portfolio-artifacts`.
2. `OpenRegistry("<registry-path>")` opens SQLite with
   `foreign_keys(1), busy_timeout(5000)`, `MaxOpenConns(1)`, and executes
   `internal/publishing/migrations/001_registry.sql`, creating
   `lab_projects`, `submissions`, `approvals`, `idempotency_results`,
   `audit_events` (with append-only triggers), and `publications`.
3. Instantiate the store and service in this order:
   `NewSQLiteRepository(db)` → `NewLocalArtifactStore(root, policy)` →
   `NewPublishingService(repo, store, publisher)` (publisher must be
   non-nil or every `ApproveAndPublish` fails with `publication_failed`).
4. Start the server (`./portfolio -port "$PORT" -dist ./frontend/dist`).

Fresh deploy = steps 1–4. Upgrade with existing data = back up the registry
file, deploy the binary, restart; `CREATE TABLE IF NOT EXISTS` makes the
single migration idempotent. Never run two writers against one registry file
(`MaxOpenConns(1)` assumes a single process).

## 5. Verifier wiring (step-up auth)

- Production (`APP_ENV=production`): construct `adminauth.Config` with
  `ApprovalVerifier: adminauth.NewPasskeyApprovalVerifier()` and
  `SecureCookies: true`. `NewService` refuses to start with a nil verifier
  (`ErrMissingApprovalVerifier`) or the dev verifier
  (`ErrDevVerifierInProduction`), and `main.go` turns that into
  `log.Fatalf` — fail closed.
- Development/test only: `LoadConfigFromEnv` auto-wires
  `NewDevApprovalVerifier(env)`, which accepts `X-Admin-StepUp-Dev: true`.
  Never set `APP_ENV=development` in production: it also flips
  `SecureCookies` to false (cookies sent over plain HTTP).
- `RequireStepUp` additionally fails closed per-request when no verifier is
  configured (`forbidden: approval verifier not configured`).

## 6. Edge / origin layout (operator-owned, no in-code implementation)

- Four origins: Admin, portfolio, Lab (`LAB_PUBLIC_ORIGIN`), preview
  (isolated, unguessable, `noindex`). Enforce HTTPS, `Secure`/`HttpOnly`/
  `SameSite=Strict` host-only admin cookies, and never forward admin cookies
  or `Authorization` headers to Lab/preview origins.
- Sandboxed iframes + CSP for executable Lab projects; `allow-same-origin`
  withheld unless a reviewed project explicitly requires it.
- Rate-limit login, review-decision, and publication endpoints (spec-required;
  no in-app limiter exists — see threat-model residual risks).
- Request-size limits at the edge complement the 1 MiB `decodeJSON` cap and
  the 32 MiB default asset cap (`defaultMaxAssetBytes`).

## 7. Fail-closed checks (verify before opening traffic)

- `go build ./...` passes.
- Start with a bogus secret missing → process exits (`log.Fatalf ... Admin
  authentication configuration failed in production`). Restore secrets.
- Unauthenticated `GET /api/admin/overview` → 401; mutation without
  `X-CSRF-Token` → 403; approve-and-publish without `X-Admin-Passkey-Assertion`
  → 403 (`ErrStepUpRequired`).
- `GET /api/admin/auth/session` without a cookie returns
  `{"authenticated": false}` with 401.

## 8. Smoke verification

```bash
BASE=https://<admin-host>
# Health / public surface (no auth)
curl -sSf "$BASE/api/health"
curl -sSf "$BASE/api/lab/projects"
curl -sSf "$BASE/api/portfolio/design-lab" | grep -F "Independent redesign concept. Not affiliated with or endorsed by the original company."
# Admin (after GitHub login in a browser; pass the session cookie here; reads need no X-Request-ID)
curl -sSf -b admin_session.jar "$BASE/api/admin/overview"
curl -sSf -b admin_session.jar "$BASE/api/admin/reviews"
curl -sSf -b admin_session.jar "$BASE/api/admin/audit?limit=5"
# Approve-and-publish dry path (IN_REVIEW submission; NEW key per attempt)
curl -sSf -X POST -b admin_session.jar \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: smoke-4" -H "X-CSRF-Token: $CSRF" \
  -H "X-Admin-Passkey-Assertion: $ASSERTION" \
  -d '{"artifact_sha256":"<reviewed-sha256>","idempotency_key":"smoke-4"}' \
  "$BASE/api/admin/reviews/<submission-id>/approve-and-publish"
```

Each mutating call requires a fresh `X-Request-ID` (`requireRequestID`
rejects missing ones with 400) and the idempotency key may come from the
body or the `Idempotency-Key` header (`HandleApproveAndPublish`).

## 9. Incident revocation / token rotation

- User session: `POST /api/admin/auth/logout` (deletes the server-side
  session, clears cookies). For a suspected takeover, restart the process —
  the default `MemorySessionStore` drops all sessions — then rotate
  `ADMIN_SESSION_SIGNING_KEY` (invalidates every signed token at once).
- OAuth App: rotate the client secret in GitHub, update the secret manager,
  restart. In-flight OAuth cookies expire in 10 minutes.
- MCP credentials: revoke the token ID (`Revoke`), issue per-agent/profile
  replacements; forbidden scopes (`lab:approve`, `lab:publish`, `admin:*`,
  `secret:*`, `shell:*`, …) can never be re-issued (`auth_test.go` pins this).

## Ruling: runbook follows the actual env names, migration path, and verifier wiring in code — inventing friendlier names would break deploys — cost if wrong: production starts insecure or not at all.
