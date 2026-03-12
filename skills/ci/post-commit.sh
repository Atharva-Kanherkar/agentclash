#!/usr/bin/env bash
set -euo pipefail

repo_root="${1:-$(git rev-parse --show-toplevel)}"

has_ci="false"

if [ -d "$repo_root/.github/workflows" ]; then
  has_ci="true"
fi

if [ -f "$repo_root/Makefile" ] || [ -f "$repo_root/package.json" ] || [ -f "$repo_root/go.mod" ]; then
  has_ci="true"
fi

if [ "$has_ci" = "true" ]; then
  echo "[skills/ci] CI-relevant entrypoints detected; review or automate the checks described in skills/ci/SKILL.md"
else
  echo "[skills/ci] skipped: no CI entrypoints detected"
fi
