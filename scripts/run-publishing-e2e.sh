#!/usr/bin/env bash
# Task 17 verification gate: MCP -> Admin publication flow.
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="/home/regio/.local/share/mise/installs/go/latest/bin:$PATH"
go test -race ./...
go vet ./...
cd frontend && npm test -- --run && npm run build
cd .. && go test ./e2e -run TestPublishingFlow -v
