#!/usr/bin/env bash
#
# AgentClash local-stack health check.
#
# Confirms the *running* dev stack is reachable and prints next steps. This is a
# RUNTIME check — for build/vet/lint/test gating use `make check` instead.
set -uo pipefail
cd "$(dirname "$0")/../.."

green(){ printf '\033[1;32m✓\033[0m %s\n' "$*"; }
red(){   printf '\033[1;31m✗\033[0m %s\n' "$*"; }
note(){  printf '\033[1;34m==>\033[0m %s\n' "$*"; }

fail=0

# Probe a TCP port portably.
#
# This used to be `timeout 1 bash -c ">/dev/tcp/$1/$2"`, but `timeout` is GNU
# coreutils and is NOT on stock macOS (only via `brew install coreutils`), so
# every check died with command-not-found and doctor reported a healthy stack as
# broken. `nc` is not a usable fallback either: macOS `nc` has no -G flag, and
# its -w does not bound connect() (measured: 76s against an unresponsive host).
# So: use timeout(1) when present, else a bash /dev/tcp connect in a subshell
# with a background watchdog that kills it after 1s.
port_open(){ # host port
  local host="$1" port="$2"

  if command -v timeout >/dev/null 2>&1; then
    timeout 1 bash -c "exec 3<>/dev/tcp/${host}/${port}" >/dev/null 2>&1
    return $?
  fi

  ( exec 3<>"/dev/tcp/${host}/${port}" ) >/dev/null 2>&1 &
  local probe=$!
  ( sleep 1 && kill -9 "$probe" ) >/dev/null 2>&1 &
  local watchdog=$!
  local rc=0
  wait "$probe" 2>/dev/null || rc=1
  kill "$watchdog" >/dev/null 2>&1 || true
  wait "$watchdog" 2>/dev/null || true
  return "$rc"
}

check_port(){ # label host port
  if port_open "$2" "$3"; then
    green "$1 reachable ($2:$3)"
  else
    red "$1 not reachable ($2:$3)"; fail=1
  fi
}

note "Checking AgentClash local stack"
check_port "Postgres" 127.0.0.1 5432
check_port "Redis"    127.0.0.1 6379
check_port "Temporal" 127.0.0.1 7233

# Only /healthz is a registered route on the API server (there is no /healthz/ready).
if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then
  green "API server healthy (http://localhost:8080/healthz)"
else
  red "API server not responding on http://localhost:8080/healthz"; fail=1
fi

echo
if [ "$fail" -eq 0 ]; then
  green "Stack looks healthy."
  echo "   → open http://localhost:3000   (web)"
  echo "   → Temporal UI: http://localhost:8233"
else
  red "Some checks failed. Bring the stack up with 'make start' (run 'make setup' first if you haven't)."
  exit 1
fi
