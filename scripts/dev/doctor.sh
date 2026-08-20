#!/usr/bin/env bash
# Compatibility entrypoint. `make doctor` and `make status` share one truth.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
exec "${ROOT_DIR}/scripts/dev/local-stack.sh" status "$@"
