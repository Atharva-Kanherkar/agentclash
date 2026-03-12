#!/usr/bin/env bash
set -euo pipefail

repo_root="${1:-$(git rev-parse --show-toplevel)}"

if ! find "$repo_root" -type f -name '*.go' -print -quit | grep -q .; then
  echo "[skills/go] skipped: no Go files detected"
  exit 0
fi

if [ ! -f "$repo_root/go.mod" ]; then
  echo "[skills/go] skipped: Go files detected but no go.mod found"
  exit 0
fi

if ! command -v go >/dev/null 2>&1; then
  echo "[skills/go] skipped: go binary not available"
  exit 0
fi

echo "[skills/go] running: go test ./..."
(cd "$repo_root" && go test ./...)
