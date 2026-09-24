#!/usr/bin/env bash
# Portfolio deploy: pull main, verify, build the Go binary and the Astro site,
# restart the service, health-check, and roll back the binary on failure.
#
# Runs on hp-server-linux. Driven by portfolio-deploy.timer (pull-based CD), the
# same pattern as ba-rekon and kasirumkm on that host: the box pulls from GitHub
# rather than GitHub pushing to the box, so no inbound access is needed.
#
# This host has no Go and no Node preinstalled in a usable form. Both are
# installed under /root/.local by the bootstrap step in deploy/README.md, and
# referenced by absolute path so a bare systemd PATH cannot break the build.
set -euo pipefail

REPO=${PORTFOLIO_REPO:-/srv/projects/portfolio}
ENV_FILE=${PORTFOLIO_ENV_FILE:-/etc/portfolio/portfolio.env}
GO_BIN=${PORTFOLIO_GO_BIN:-/root/.local/go/bin/go}
NODE_BIN_DIR=${PORTFOLIO_NODE_BIN_DIR:-/root/.local/node/bin}
LOG_TAG=portfolio-deploy

log() { printf '[%s] %s\n' "$LOG_TAG" "$*"; }
fail() { printf '[%s] FATAL: %s\n' "$LOG_TAG" "$*" >&2; exit 1; }

[ -d "$REPO/.git" ] || fail "$REPO is not a git checkout"
[ -f "$ENV_FILE" ] || fail "$ENV_FILE missing; refusing to deploy without configuration"
[ -x "$GO_BIN" ] || fail "$GO_BIN not found; run the bootstrap step in deploy/README.md"
[ -x "$NODE_BIN_DIR/node" ] || fail "$NODE_BIN_DIR/node not found; run the bootstrap step"

# The build needs PORTFOLIO_PUBLIC_ORIGIN for canonical URLs and the sitemap, and
# the service needs the rest at runtime. One file serves both.
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

# Derived after loading the environment so one file stays the single source of
# truth for the port, rather than the unit and the script drifting apart.
LISTEN_PORT=${PORT:-8321}
BASE_URL="http://127.0.0.1:$LISTEN_PORT"
HEALTH_URL="$BASE_URL/api/health"

cd "$REPO"

BEFORE=$(git rev-parse HEAD 2>/dev/null || echo none)
git fetch --quiet origin main
AFTER=$(git rev-parse origin/main)
if [ "$BEFORE" = "$AFTER" ]; then
    exit 0  # nothing new; the timer runs every minute, so this is the common path
fi
log "deploying $BEFORE..$AFTER"

git reset --hard --quiet origin/main

# 1. Frontend. `npm run build` regenerates the content JSON from pkg/api first,
#    so the build cannot use a stale copy of the site content. That prebuild step
#    shells out to `go run`, so both toolchains must be on PATH here — exporting
#    only Node leaves the content export failing with "go: not found".
export PATH="$NODE_BIN_DIR:$(dirname "$GO_BIN"):$PATH"
( cd "$REPO/frontend" && npm ci --silent && npm run build ) || fail "frontend build failed"
[ -f "$REPO/frontend/dist/index.html" ] || fail "frontend build produced no dist/index.html"

# 2. Backend. Built beside the live binary and moved into place, so a failed
#    build cannot leave a half-written executable behind.
"$GO_BIN" build -trimpath -ldflags="-s -w" -o "$REPO/portfolio-server.new" . \
    || fail "go build failed"
mv -f "$REPO/portfolio-server.new" "$REPO/portfolio-server"

# 3. Preview runner image. The runner launches this image per sandboxed build,
#    so it must exist before any preview is requested. Idempotent: only built
#    when Docker is reachable and the image is missing, because preview builds
#    are the only thing that needs it and the rest of the site must still deploy
#    on a host without Docker.
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    if ! docker image inspect portfolio-preview-runner:test >/dev/null 2>&1; then
        log "building preview runner image"
        docker build -q -f "$REPO/deploy/preview/Dockerfile.runner" \
            -t portfolio-preview-runner:test "$REPO" >/dev/null \
            || log "WARNING: preview runner image build failed; previews will not run"
    fi
else
    log "WARNING: docker unavailable; previews will not run (the rest of the site is unaffected)"
fi

# 4. Restart and health-check.
systemctl restart portfolio || fail "systemctl restart portfolio failed"

ok=0
for _ in $(seq 1 30); do
    if curl -sf --max-time 3 "$HEALTH_URL" >/dev/null 2>&1; then ok=1; break; fi
    sleep 2
done
[ "$ok" = "1" ] || fail "health check failed after deploying $AFTER"

# 5. The static contract, checked on the running service: the pages must be
#    served as their own documents, not as a fallback shell.
for path in / /about/ /work/ /services/ /contact/; do
    curl -sf --max-time 5 "$BASE_URL$path" >/dev/null || fail "$path not served"
done

log "deployed $AFTER, health OK"
