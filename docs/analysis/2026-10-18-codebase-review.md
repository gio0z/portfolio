# Portfolio codebase review — 2026-10-18

Method: 50 parallel read-only analysis slices over the whole repo (Go backend, Astro/React frontend,
deploy, CI, tools, tests, docs). **Static analysis only** — no builds, tests, or containers were run
during the sweep. Claims below are marked:

- **[V]** = verified by the report author reading the cited code directly.
- **[R]** = reported by a slice and consistent with surrounding evidence, not independently re-read.
- **[I]** = inference about runtime behavior from static reading (not observed).

Verified live state at review time: `https://ginapps.my.id/` `/work` `/api/health` all return 200;
`portfolio-deploy.timer` active on `hp-server-linux`, firing every minute.

---

## Fixes applied in this pass (2026-10-18)

Gate: `bash scripts/run-publishing-e2e.sh` — **EXIT=0** after `go clean -testcache`, so the race
suite, `go vet`, vitest (73 tests), the Astro build and the static-output contract all ran fresh.
`gofmt -l` clean, `npm run lint` (oxlint) silent.

| Fix | Files | Verified by |
|-----|-------|-------------|
| Blocker 3 — rollback restores the previous pointer instead of unpublishing; rollback errors propagated instead of discarded | `internal/deployment/publisher.go` (`PrevVersion` on both staged structs, restored in both `Rollback`s, `errors.Join` on the publish failure paths), `internal/deployment/publisher_test.go` (+7 tests) | `go test ./internal/deployment/` |
| Blocker 4 — sandbox output refuses symlink entries before reading, failing closed | `internal/preview/docker_runner.go`, new `internal/preview/docker_runner_test.go` | `go test ./internal/preview/` |
| Row 5 — gate now probes every project page from `portfolio.json`, permits only `ld+json` script tags on static pages, asserts `/admin` and `/lab` are absent from the sitemap, and uses `mktemp` for the freshness check | `scripts/run-publishing-e2e.sh` | full gate run |
| SEO — `/lab` excluded from the sitemap and disallowed in robots.txt | `frontend/astro.config.mjs`, `frontend/src/pages/robots.txt.ts` | full gate run |

New defect found *by* the broadened gate, and fixed in the same pass: the sitemap filter used
`page.includes('/lab')`, which also matched `/work/labtu-web/`, silently dropping a real project
page from the sitemap. The filter is now anchored to the first path segment
(`/^\/(admin|lab)(\/|$)/`) and the gate's exclusion check is `<loc>`-anchored for the same reason.
Proof: the sitemap now lists all six work routes and zero `/admin` or `/lab` entries.

Still open from this report: blocker 1 (step-up is a presence check, and no client sends the
header — a real fix needs the WebAuthn ceremony, not a patch I should improvise), blocker 2
(preview image is a stub, so pipeline "evidence" is self-reported), and rows 6-12.

---

## Blockers — fix before the next real publish

### 1. The human approval gate does not hold [V]

- `internal/adminauth/middleware.go:72-79` — production `PasskeyApprovalVerifier.VerifyApproval`
  only checks that `X-Admin-Passkey-Assertion` is non-empty. No signature, challenge, or credential
  check. Any string passes.
- `frontend/src/admin/AdminApi.ts:205-222` — `approveAndPublish` sends only `Idempotency-Key`; the
  admin UI never sends `X-Admin-Passkey-Assertion` (prod) or `X-Admin-StepUp-Dev` (dev).
- Repo-wide grep for both header names: only docs, tests, and the CORS allowlist
  (`pkg/api/server.go:113`). No emitter exists in the app.

Net effect: the UI approve path is dead (403 both in dev and production), and the production gate is
satisfiable by any caller that sends a junk header. The publish gate is therefore reachable *only*
by hand-written curl (as documented in `docs/runbooks/admin-mcp-lab-deployment.md:125`) with a
header that is not actually verified.

Fix direction: implement real WebAuthn assertion verification (challenge, RP ID, signature), or gate
on a server-side minted approval token; and send whatever proof the UI is supposed to send.

### 2. The preview runner cannot build anything [V]

- `deploy/preview/Dockerfile.runner:11-20` — `FROM alpine:3.20`, `CMD ["/bin/true"]`, no toolchain.
- `internal/preview/policy.go:97-111` — `DockerRunArgs` appends no command after the image.
- `e2e/publishing_flow_test.go:100` — the E2E passes `build_result` / `test_result` /
  `security_scan_result` as literals into `ToolDeployLabPreview`.

Net effect: the container exits 0 immediately and `/out` is empty, so the "build, test, security
scan passed" evidence used to unblock approval is agent self-report. The sandbox itself is genuinely
locked down (`--network=none --read-only --cap-drop=ALL --pids-limit --memory --cpus`,
non-root); the *payload* is a stub. [I] for the empty-output consequence.

### 3. Rollback takes the previously live version offline [V for the code path, I for the interleaving] — **FIXED**

- `internal/deployment/publisher.go:195-207` (`labPublisher.Rollback`, mirrored for portfolio at
  :329-341) — if the pointer names the staged version, it calls `os.Remove(ptr)`. The pointer *is*
  the publication record; deleting it unpublishes the slug instead of restoring the previous
  release. Comment claims it "keeps a rollback from unpublishing an older live release", but no
  previous pointer value is preserved anywhere.
- Same file :201, :204, :335, :338 — `_ =` on every `os.Remove`/`os.RemoveAll`; and
  `internal/deployment/publisher.go:400` (`_ = rollbackLab(...)`) discards rollback failures.

Fix direction: read the current pointer JSON before the swap, restore it on failure, and propagate
rollback errors.

### 4. Preview output collection follows symlinks out of the sandbox [V for the code, I for exploit] — **FIXED**

- `internal/preview/docker_runner.go:92-115` — `readOutputDir` checks `item.IsDir()` then
  `os.ReadFile(full)`. `os.ReadDir` uses `Lstat` semantics, so a symlink entry is treated as a file
  and dereferenced **on the host**. A build script running in the sandbox can drop
  `ln -s /etc/passwd /out/leak.html`; the host copies the target's bytes into a served preview.
- `os.ReadFile` happens before any extension allowlist check (`internal/preview/runner.go:285`).

Fix direction: reject `item.Type()&os.ModeSymlink != 0`, or open with `O_NOFOLLOW` and verify the
entry is a regular file.

---

## High

| # | Finding | Evidence | Status |
|---|---------|----------|--------|
| 5 | Verification gate probes 1 of 6 project pages; zero-JS regex matches only `src="/_astro/*.js"`; sitemap check never asserts `/lab` exclusion; fixed `/tmp/contentexport-fresh.json` path with no trap | `scripts/run-publishing-e2e.sh:15,28-33,60,68-71`; project ids verified: `jam-nguar, inven-kab-blitar, labtu-web, cs-portal, tour-travel-web, nusantara-botanica` | [V] **FIXED** |
| 6 | Deploy has no rollback despite the header claim: `frontend/dist` is rebuilt in place, `git reset --hard origin/main` runs *before* any build, previous binary never backed up; health-check failure leaves the broken revision in place and the next timer run sees `BEFORE == AFTER` | `deploy/deploy.sh:3,70-76,86,119-126` | [R] |
| 7 | Unbounded in-memory growth on the public surface: contact submissions appended and never read/evicted, no `MaxBytesReader` on `/api/contact`; `MemorySessionStore` never evicts un-`Get`-ed sessions | `pkg/api/server.go:22,238,258`; `internal/adminauth/session.go:43-79` | [V] |
| 8 | State machine bypassed: `UpdateDraft` blocks only PUBLISHED/ARCHIVED/REJECTED, then sets `State = Draft`, so an agent can pull a submission out of `IN_REVIEW` unilaterally; `APPROVED` is unreachable (publish jumps `IN_REVIEW → PUBLISHED`) | `internal/publishing/service.go:284,324,809`; `internal/publishing/state.go:27-40` | [V] |
| 9 | JSON-LD interpolated raw into an inline `<script>`: `JSON.stringify` output reaches `set:html` unescaped, so any `</script>` in content breaks the document head | `frontend/src/lib/seo.ts:96,114`; `frontend/src/layouts/BaseLayout.astro:52` | [V] (exploitability limited: content is author-authored) |
| 10 | `HOST` defaults to `0.0.0.0`, binding the admin API and MCP endpoint on every interface | `internal/app/config.go:117-119`; documented risk in `.env.example:8-12` | [V] |
| 11 | CI runs the gate but not `npm run lint` / `npm run check`; actions pinned to mutable tags; no `concurrency` group; no failure artifacts | `.github/workflows/verify.yml:17,19,26,39-40` | [R] |
| 12 | SQLite and query shape: no WAL, `ListAuditEvents` loads all rows then filters in Go (and skips `LIMIT` when filters are set), missing indexes for `ListSubmissions` / `ListLabProjects`, public lab endpoints load every project then filter in Go, single slug lookups do a full list scan | `internal/publishing/sqlite_repository.go:45,139,403-437`; `internal/publicapi/handler.go:69,90` | [R] |

---

## Medium (backlog, by area)

**Correctness / robustness**
- Rollback and pointer writes: `writePointerFile` and artifact/manifest writes never `f.Sync()`
  before rename (`internal/deployment/publisher.go:440-449`, `internal/publishing/local_artifact_store.go:130,247-262`);
  artifact metadata write error swallowed (`local_artifact_store.go:130` [V]); archive handler
  falls back to `subs[0]` and discards `UpdateLabProject` errors (`internal/adminapi/handler.go:424,426-433` [V]).
- Publish is not atomic across disk and DB: `Promote`/`Publish` run outside `WithTx`, comment in
  service claims the opposite ordering is safe; failed-approval audit append is `_ =`-discarded
  (`internal/publishing/service.go:771-878,789-802`).
- Idempotency result returned without checking it belongs to the same submission
  (`internal/publishing/service.go:721-731`).
- Publishing idempotency cache is process-local RAM with no slug lock; concurrent publishes of one
  slug can interleave (`internal/deployment/publisher.go:353-434`).
- API shape/behavior: `/api/health|profile|projects|skills` accept any method (only contact handlers
  check) (`pkg/api/server.go:191-227` [V]); unmatched public paths return 200 + homepage (soft 404)
  (`internal/app/app.go:318-322`, asserted in `internal/app/static_test.go:185`); no `Cache-Control`
  on static assets (`internal/app/app.go:349`); wildcard `Access-Control-Allow-Origin: *` set even on
  `/api/admin/*` (`pkg/api/server.go:111`).
- Admin UI: `actionSuccess`/`actionError`/`activeTab` not reset when the review id changes;
  `refreshDetail` and `handleRefresh` lack cancellation guards
  (`frontend/src/admin/ReviewDetail.tsx:28-45,47-66`; `ReviewQueue.tsx:18-32`).
- `AdminApi.mapError` has no `428` case; DTO drift: `portfolio_metadata` narrowed vs Go
  `map[string]any`, `DecisionResponse.submission` typed `unknown`
  (`frontend/src/admin/AdminApi.ts:75-98`, `types.ts:34-39,119`).
- Asset pipeline: full 32 MB reads into RAM per asset and per zip entry
  (`internal/publishing/asset_validation.go:47,155`); artifact/metadata files world-readable
  (`local_artifact_store.go:118,130`); zip SVG scanner misses event handlers
  (`asset_validation.go:159`); `NormalizeSourceURLs` has no count/length cap and preserves
  userinfo (`internal/publishing/source_urls.go:22,42`).
- MCP: no `recover()` in `CallTool`/`ServeHTTP` (`internal/mcppublisher/server.go:98` [V-ish: no
  recover middleware exists]); response encode errors discarded (`:372,387`).
- `statsgen` output has no freshness check in CI and depends on `$HOME/projects`
  (`tools/statsgen/main.go:75`; `pkg/api/stats_generated.go`).

**SEO / docs**
- `/lab` is prerendered with `noindex` yet not excluded from the sitemap and not disallowed in
  robots.txt (`frontend/astro.config.mjs:52` [V], `frontend/src/pages/robots.txt.ts`).
- `canonicalUrl` does not strip `?query`/`#fragment` (`frontend/src/lib/seo.ts:30`).
- Runbook env table omits 6 variables that `validateProduction` requires
  (`docs/runbooks/admin-mcp-lab-deployment.md:17-26` vs `internal/app/config.go:167-178` [V];
  all six are in `.env.example`); rollback runbook cites stale artifact/preview paths
  (`docs/runbooks/admin-mcp-lab-rollback.md:33,35`).
- `CONTEXT.md` Seam A list omits `GET /api/contact/whatsapp`.

**Repo weight / dead code**
- 23 screenshots totalling ~8.0 MB tracked in git, including pre-migration artifacts, plus
  `capture_film.mjs`, `capture_sections.mjs` (hardcoded `localhost:8090`, `/home/gio/...`);
  `tmp/` missing from `.gitignore` [V for size and file list].
- Unreferenced Go surface: `SetAdminAPI`, `SetRepository`, `SetPublicAPIReader`,
  `WithPublicAPIReader` (`pkg/api/server.go:38,68,81,88`), `deployment.Rollback` /
  `RollbackRequest` (`internal/deployment/rollback.go:45`), `publicationsDir`
  (`internal/publishing/local_artifact_store.go:50`); `onlyPublishedProjects`
  (`frontend/src/lab/LabApi.ts:37`); `DesignLabCard` default export.
- Three near-duplicate project type definitions (`frontend/src/types.ts`,
  `lab/LabApi.ts`, `admin/types.ts`).

**Accessibility**
- Contact form labels have no `htmlFor`/`id` pairing; status banners have no `role`/`aria-live`
  (`frontend/src/components/ContactSection.tsx:135,144,154-198`).
- `ExpertiseSection` and `WorkTabs` tab strips lack `role="tablist"/"tab"/"aria-selected"/
  `aria-controls`/`role="tabpanel"` and arrow-key navigation
  (`frontend/src/components/ExpertiseSection.tsx:64-131`; `WorkTabs.tsx:84-117`).
- Coverflow cards are clickable `div`s with no keyboard affordance
  (`frontend/src/components/CoverflowSection.tsx:141-143`); no skip-to-content link
  (`BaseLayout.astro:63`); `ApprovalDialog` has no Escape handler/focus trap
  (`ApprovalDialog.tsx:32`); Lab "missing" state lacks `role="alert"` (`LabCaseStudy.tsx:135-139`).
- Nav id `#about` in the hero collides with the `/about` route (`Hero.tsx:44`).

**Consistency / drift**
- External-link policy applied inconsistently: `rel="noreferrer"` without `noopener`
  (`CoverflowSection.tsx:254,266`), `rel="me noopener"` without `target` (`about.astro:78`),
  `rel="noopener"` only (`work/[id].astro:88`), bare `noreferrer` (`DesignLabCard.tsx:86`).
  Admin `EvidencePanel` binds `href={asset.url}` with no scheme check (`EvidencePanel.tsx:143`);
  `LabCaseStudy` validates `source_urls` but not `cover_image` / `live_demo_url`
  (`LabCaseStudy.tsx:166-187`).
- Hero hardcodes its two metrics (`9`, `653`) instead of reading `profile.stats`, so the values are
  duplicated: currently in sync with `portfolio.json`, but a change in `pkg/api/content.go` will not
  reach the hero, and nothing fails when it diverges (`Hero.tsx:124,137` [V]).
- Coverflow computes `window.innerWidth` during render (`CoverflowSection.tsx:129`), so the
  prerendered markup and the hydrated mobile layout disagree. [I: hydration mismatch/flash]
- No `Content-Security-Policy` anywhere; the only XSS mitigations are escaping and iframe `sandbox`.

---

## Agent claims that did not survive checking

Recorded so the report is not trusted blindly:

- "`internal/mcppublisher` is not mounted in `internal/app`" — **false**. `app.go:158-167`
  constructs the authenticator/server and `app.go:293-294` routes `MCPRoute` when
  `MCP_TOKEN_SECRET` is set.
- "`WorkTabs.tsx` is unused" — **false**. Imported and rendered by
  `frontend/src/components/CoverflowSection.tsx:4,86`.
- Project slug lists invented by two slices (`kembara`, `sipandu-desa`, `resto-cafe`) — **false**;
  see the verified id list in row 5. The underlying point (gate tests one page of six) stands.
- "Queue approve bypasses step-up" — true about the dialog (`ReviewQueue.tsx:306`), but the request
  would fail anyway per blocker 1.

---

## Limits of this review

- No build, test run, container run, or browser session was executed; every runtime behavior claim
  is static inference and is marked `[I]` or `[R]`.
- Nothing was confirmed against the live server beyond the three HTTP probes above.
- Coverage-gap findings are static reference counts, not measured coverage.
- Slices did not read `frontend/node_modules`, `frontend/dist`, or `.superpowers/` ledgers.

## Suggested order of work

1. Blocker 1 (step-up) and blocker 3 (rollback) — both are one-file changes with test surface.
2. Blocker 2 + 4 (preview image payload and symlink rejection) — they decide whether "preview
   evidence" means anything.
3. Rows 5-8: gate coverage, deploy rollback, unbounded memory, state machine.
4. Then the a11y and dead-weight batches, which are mechanical.
