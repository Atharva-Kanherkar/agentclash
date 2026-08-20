#!/usr/bin/env bash
# Compatibility entrypoint. Prefer `make start` for the full lifecycle.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
exec "${ROOT_DIR}/scripts/dev/local-stack.sh" start "$@"
