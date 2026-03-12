#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"

echo "[skills] running post-commit skills from $repo_root"

for skill in git-rules ci go; do
  script="$repo_root/skills/$skill/post-commit.sh"
  if [ -x "$script" ]; then
    "$script" "$repo_root"
  else
    echo "[skills/$skill] skipped: no executable post-commit.sh found"
  fi
done
