#!/usr/bin/env bash
# Task 17 verification gate: MCP -> Admin publication flow, plus the static
# output contract of the prerendered site.
set -euo pipefail
cd "$(dirname "$0")/.."

# The Astro build fails closed without a canonical origin, because a wrong
# canonical points search engines at a host this project does not control.
# Local gate runs get an explicit localhost origin; deploys set the real one.
export PORTFOLIO_PUBLIC_ORIGIN="${PORTFOLIO_PUBLIC_ORIGIN:-http://localhost:8080}"

# The committed frontend content must match the Go source it is derived from.
# Checked before the build, because npm run build regenerates it and would
# otherwise mask a stale file.
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
go run ./tools/contentexport/main.go > "$tmp"
cmp -s frontend/src/content/portfolio.json "$tmp" \
  || { echo "frontend/src/content/portfolio.json is stale: re-run npm run prebuild" >&2; exit 1; }

go test -race ./...
go vet ./...
cd frontend && npm test -- --run && npm run build
cd ..

# Derive project slugs dynamically from committed content so assertions never drift.
mapfile -t project_slugs < <(python3 -c "import json;[print(p['id']) for p in json.load(open('frontend/src/content/portfolio.json'))['projects']]")
[ "${#project_slugs[@]}" -gt 0 ] || { echo "no project slugs found in content" >&2; exit 1; }

project_pages=()
for slug in "${project_slugs[@]}"; do
  project_pages+=("work/$slug/index")
done

# Static output gate: every public page must exist and carry its own metadata.
# A build that emitted one document for every route, or dropped a page, would
# still pass the application tests above.
dist=frontend/dist
all_expected_pages=(index about/index work/index services/index contact/index "${project_pages[@]}" admin/index lab/index)
for page in "${all_expected_pages[@]}"; do
  [ -f "$dist/$page.html" ] || { echo "missing prerendered page: $dist/$page.html" >&2; exit 1; }
done

public_content_pages=(index about/index work/index services/index contact/index "${project_pages[@]}")
for page in "${public_content_pages[@]}"; do
  grep -q '<title>[^<][^<]*</title>' "$dist/$page.html" \
    || { echo "no <title> in $dist/$page.html" >&2; exit 1; }
  grep -q 'rel="canonical"' "$dist/$page.html" \
    || { echo "no canonical link in $dist/$page.html" >&2; exit 1; }
  grep -q 'name="description"' "$dist/$page.html" \
    || { echo "no description in $dist/$page.html" >&2; exit 1; }
done

# Distinct titles: identical ones would mean the metadata layer regressed to
# the single-shell behaviour this replaced.
titles=$(for page in "${public_content_pages[@]}"; do
  grep -o '<title>[^<]*</title>' "$dist/$page.html"
done)
expected="${#public_content_pages[@]}"
if [ "$(printf '%s\n' "$titles" | sort -u | wc -l)" -ne "$expected" ]; then
  echo "page titles are not distinct" >&2
  exit 1
fi

# The public pages are content, not JavaScript: a regressed build would start
# shipping client scripts to pages that never needed hydration. Every <script>
# tag emitted on static content pages must be structured data (ld+json) only;
# any runtime, module, relative, or third-party script fails the gate.
static_pages=(about/index work/index services/index "${project_pages[@]}")
for page in "${static_pages[@]}"; do
  f="$dist/$page.html"
  disallowed_scripts=$(grep -o '<script[^>]*>' "$f" | grep -v 'type="application/ld+json"' || true)
  if [ -n "$disallowed_scripts" ]; then
    echo "$f unexpectedly contains non-ld+json scripts: $disallowed_scripts" >&2
    exit 1
  fi
done

[ -f "$dist/robots.txt" ] || { echo "missing robots.txt" >&2; exit 1; }
[ -f "$dist/sitemap-0.xml" ] || { echo "missing sitemap" >&2; exit 1; }
for slug in "${project_slugs[@]}"; do
  grep -q "/work/$slug/" "$dist/sitemap-0.xml" \
    || { echo "sitemap missing project route for $slug" >&2; exit 1; }
done
# Segment-anchored: a bare substring test for /lab would also match
# /work/labtu-web/, which is a real page and must be listed.
if grep -qE '<loc>[^<]*/(admin|lab)/' "$dist/sitemap-0.xml"; then
  echo "sitemap must not list the admin or lab areas" >&2
  exit 1
fi
go test ./e2e -run TestPublishingFlow -v

echo "gate OK: build, tests, prerendered pages, metadata, sitemap, content freshness"
