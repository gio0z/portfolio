#!/usr/bin/env bash
# Task 17 verification gate: MCP -> Admin publication flow, plus the static
# output contract of the prerendered site.
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="/home/regio/.local/share/mise/installs/go/latest/bin:$PATH"

# The committed frontend content must match the Go source it is derived from.
# Checked before the build, because npm run build regenerates it and would
# otherwise mask a stale file.
go run ./tools/contentexport/main.go > /tmp/contentexport-fresh.json
cmp -s frontend/src/content/portfolio.json /tmp/contentexport-fresh.json \
  || { echo "frontend/src/content/portfolio.json is stale: re-run npm run prebuild" >&2; exit 1; }

go test -race ./...
go vet ./...
cd frontend && npm test -- --run && npm run build
cd ..

# Static output gate: every public page must exist and carry its own metadata.
# A build that emitted one document for every route, or dropped a page, would
# still pass the application tests above.
dist=frontend/dist
for page in index about/index work/index services/index contact/index \
            work/jam-nguar/index admin/index lab/index; do
  [ -f "$dist/$page.html" ] || { echo "missing prerendered page: $dist/$page.html" >&2; exit 1; }
done

for page in index about/index work/index services/index contact/index work/jam-nguar/index; do
  grep -q '<title>[^<][^<]*</title>' "$dist/$page.html" \
    || { echo "no <title> in $dist/$page.html" >&2; exit 1; }
  grep -q 'rel="canonical"' "$dist/$page.html" \
    || { echo "no canonical link in $dist/$page.html" >&2; exit 1; }
  grep -q 'name="description"' "$dist/$page.html" \
    || { echo "no description in $dist/$page.html" >&2; exit 1; }
done

# Distinct titles: identical ones would mean the metadata layer regressed to
# the single-shell behaviour this replaced.
pages='index about/index work/index services/index contact/index work/jam-nguar/index'
titles=$(for page in $pages; do
  grep -o '<title>[^<]*</title>' "$dist/$page.html"
done)
expected=$(printf '%s\n' $pages | wc -l)
if [ "$(printf '%s\n' "$titles" | sort -u | wc -l)" -ne "$expected" ]; then
  echo "page titles are not distinct" >&2
  exit 1
fi

# The public pages are content, not JavaScript: a regressed build would start
# shipping the React runtime to pages that never needed it. Matches an external
# script reference only — an inline ld+json block is content, and the tech tags
# include the literal string "Next.js".
for page in about/index work/index services/index work/jam-nguar/index; do
  if grep -q 'src="/_astro/[^"]*\.js"' "$dist/$page.html"; then
    echo "$dist/$page.html unexpectedly loads JavaScript" >&2
    exit 1
  fi
done

[ -f "$dist/robots.txt" ] || { echo "missing robots.txt" >&2; exit 1; }
[ -f "$dist/sitemap-0.xml" ] || { echo "missing sitemap" >&2; exit 1; }
grep -q '/work/jam-nguar/' "$dist/sitemap-0.xml" \
  || { echo "sitemap missing project routes" >&2; exit 1; }
if grep -q '/admin' "$dist/sitemap-0.xml"; then
  echo "sitemap must not list the admin area" >&2
  exit 1
fi

go test ./e2e -run TestPublishingFlow -v

echo "gate OK: build, tests, prerendered pages, metadata, sitemap, content freshness"
