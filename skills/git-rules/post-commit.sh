#!/usr/bin/env bash
set -euo pipefail

repo_root="${1:-$(git rev-parse --show-toplevel)}"
cd "$repo_root"

echo "[skills/git-rules] commit: $(git rev-parse --short HEAD)"

if git rev-parse HEAD~1 >/dev/null 2>&1; then
  if git diff --check HEAD~1 HEAD >/dev/null 2>&1; then
    echo "[skills/git-rules] diff sanity: ok"
  else
    echo "[skills/git-rules] diff sanity: issues found"
    git diff --check HEAD~1 HEAD || true
  fi
else
  echo "[skills/git-rules] diff sanity: skipped (no parent commit)"
fi

if [ -n "$(git status --short)" ]; then
  echo "[skills/git-rules] working tree still has changes after commit"
else
  echo "[skills/git-rules] working tree clean"
fi
