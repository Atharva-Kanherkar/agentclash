#!/usr/bin/env bash
# Docker-free contract tests for scripts/dev/local-stack.sh.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONTROLLER="${ROOT_DIR}/scripts/dev/local-stack.sh"
TEST_BASE="${TMPDIR:-/tmp}"
TEST_BASE="${TEST_BASE%/}"
TEST_TMP="$(mktemp -d "${TEST_BASE}/agentclash-stack-test.XXXXXX")"
FAKE_BIN="${TEST_TMP}/bin"
FAKE_DOCKER_STATE_DIR="${TEST_TMP}/docker-state"
FAKE_RUNTIME_DIR="${TEST_TMP}/runtime"
FAKE_DOCKER_LOG="${TEST_TMP}/docker.log"
FAKE_MAKE_LOG="${TEST_TMP}/make.log"
TEST_ENV_FILE="${TEST_TMP}/backend.env"
LAST_OUTPUT="${TEST_TMP}/last-output"
CURRENT_STATE_DIR=""
TESTS=0

export FAKE_DOCKER_STATE_DIR FAKE_RUNTIME_DIR FAKE_DOCKER_LOG FAKE_MAKE_LOG

cleanup() {
  local pid_file pid command
  while IFS= read -r pid_file; do
    IFS= read -r pid <"${pid_file}" || true
    if [[ "${pid:-}" =~ ^[1-9][0-9]*$ ]]; then
      command="$(ps -ww -p "${pid}" -o command= 2>/dev/null || true)"
      case "${command}" in
        *"${TEST_TMP}"*) kill -KILL "${pid}" >/dev/null 2>&1 || true ;;
      esac
    fi
  done < <(find "${TEST_TMP}" -name '*.pid' -type f 2>/dev/null || true)
  if [[ -n "${FOREIGN_PID:-}" ]]; then
    kill -KILL "${FOREIGN_PID}" >/dev/null 2>&1 || true
    wait "${FOREIGN_PID}" 2>/dev/null || true
  fi
  case "${TEST_TMP}" in
    "${TEST_BASE}"/agentclash-stack-test.*) rm -rf -- "${TEST_TMP}" ;;
    *) printf 'refusing to remove unexpected test path: %s\n' "${TEST_TMP}" >&2 ;;
  esac
}
trap cleanup EXIT INT TERM

mkdir -p "${FAKE_BIN}" "${FAKE_DOCKER_STATE_DIR}" "${FAKE_RUNTIME_DIR}"
: >"${FAKE_DOCKER_LOG}"
: >"${FAKE_MAKE_LOG}"
cat >"${TEST_ENV_FILE}" <<'EOF'
API_SERVER_BIND_ADDRESS=:8080
TEMPORAL_HOST_PORT=localhost:7233
TEMPORAL_NAMESPACE=default
DATABASE_URL=postgres://agentclash:agentclash@localhost:5432/agentclash?sslmode=disable
REDIS_URL=redis://localhost:6379
EOF

cat >"${FAKE_BIN}/docker" <<'EOF'
#!/usr/bin/env bash
set -u

state_dir="${FAKE_DOCKER_STATE_DIR}"
log="${FAKE_DOCKER_LOG}"
printf '%s\n' "$*" >>"${log}"

if [[ "${1:-}" == "inspect" ]]; then
  id=""
  for argument in "$@"; do id="${argument}"; done
  service="${id#fake-}"
  [[ -f "${state_dir}/${service}.exists" ]] || exit 1
  status="$(cat "${state_dir}/${service}.status")"
  health="$(cat "${state_dir}/${service}.health")"
  printf '%s|%s\n' "${status}" "${health}"
  exit 0
fi

[[ "${1:-}" == "compose" ]] || exit 2
shift
command="${1:-}"
shift || true

case "${command}" in
  ps)
    [[ ! -f "${state_dir}/unavailable" ]] || exit 1
    service=""
    for argument in "$@"; do service="${argument}"; done
    if [[ "$*" == *"-q"* && -f "${state_dir}/${service}.exists" ]]; then
      printf 'fake-%s\n' "${service}"
    fi
    ;;
  pull)
    exit 0
    ;;
  up)
    failed=0
    for service in "$@"; do
      [[ "${service}" == -* ]] && continue
      : >"${state_dir}/${service}.exists"
      printf 'running\n' >"${state_dir}/${service}.status"
      if [[ "${service}" == "temporal" && -f "${state_dir}/temporal.fail" ]]; then
        printf 'unhealthy\n' >"${state_dir}/${service}.health"
        failed=1
      else
        printf 'healthy\n' >"${state_dir}/${service}.health"
      fi
    done
    # An unhealthy container can still be created successfully by compose up.
    [[ "${failed}" == "0" ]] || exit 0
    ;;
  stop)
    for service in "$@"; do
      [[ "${service}" == -* || "${service}" =~ ^[0-9]+$ ]] && continue
      if [[ -f "${state_dir}/${service}.exists" ]]; then
        printf 'exited\n' >"${state_dir}/${service}.status"
        printf 'none\n' >"${state_dir}/${service}.health"
      fi
    done
    ;;
  logs)
    for service in postgres redis temporal; do
      if [[ "$*" == *"${service}"* && -f "${state_dir}/${service}.exists" ]]; then
        printf '%s | %s-log\n' "${service}" "${service}"
      fi
    done
    if [[ "$*" == *"--follow"* ]]; then
      follower_file="${state_dir}/logs-follower.pid"
      printf '%s\n' "$$" >"${follower_file}"
      trap 'rm -f "${follower_file}"; exit 0' INT TERM EXIT
      while true; do sleep 1; done
    fi
    ;;
  *) exit 2 ;;
esac
EOF

cat >"${FAKE_BIN}/go" <<'EOF'
#!/usr/bin/env bash
set -eu

out=""
role=""
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    ./cmd/api-server) role="api"; shift ;;
    ./cmd/worker) role="worker"; shift ;;
    *) shift ;;
  esac
done
[[ -n "${out}" && -n "${role}" ]]

cat >"${out}" <<RUNTIME
#!/usr/bin/env bash
echo "${role}-log"
if [[ -f "\${FAKE_RUNTIME_DIR}/${role}.exit" ]]; then
  exit 1
fi
if [[ -f "\${FAKE_RUNTIME_DIR}/${role}.ignore-term" ]]; then
  trap '' TERM
else
  trap 'exit 0' TERM INT
fi
while true; do sleep 1; done
RUNTIME
chmod +x "${out}"
EOF

cat >"${FAKE_BIN}/make" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_MAKE_LOG}"
[[ ! -f "${FAKE_RUNTIME_DIR}/migration.fail" ]]
EOF

cat >"${FAKE_BIN}/curl" <<'EOF'
#!/usr/bin/env bash
pid=""
if [[ -f "${STATE_DIR}/api-server.pid" ]]; then
  IFS= read -r pid <"${STATE_DIR}/api-server.pid" || true
fi
if [[ -f "${FAKE_RUNTIME_DIR}/api-unready" ]] || \
   [[ ! "${pid}" =~ ^[1-9][0-9]*$ ]] || ! kill -0 "${pid}" >/dev/null 2>&1; then
  [[ "$*" == *"%{http_code}"* ]] && printf '000'
  exit 7
fi
[[ "$*" == *"%{http_code}"* ]] && printf '200'
EOF

cat >"${FAKE_BIN}/timeout" <<'EOF'
#!/usr/bin/env bash
probe=""
for argument in "$@"; do probe="${argument}"; done
port=""
case "${probe}" in
  *'/5432') service=postgres; port=5432 ;;
  *'/6379') service=redis; port=6379 ;;
  *'/7233') service=temporal; port=7233 ;;
  *'/8080') service=api; port=8080 ;;
  *) exit 1 ;;
esac

if [[ -f "${FAKE_RUNTIME_DIR}/foreign-${port}" ]]; then
  exit 0
fi
if [[ "${service}" == "api" ]]; then
  [[ -f "${STATE_DIR}/api-server.pid" ]] || exit 1
  pid="$(cat "${STATE_DIR}/api-server.pid")"
  kill -0 "${pid}" >/dev/null 2>&1
  exit $?
fi
if [[ "${service}" == "temporal" && -f "${STATE_DIR}/temporal.pid" ]]; then
  pid="$(cat "${STATE_DIR}/temporal.pid")"
  if kill -0 "${pid}" >/dev/null 2>&1; then
    exit 0
  fi
fi
[[ -f "${FAKE_DOCKER_STATE_DIR}/${service}.status" ]] || exit 1
[[ "$(cat "${FAKE_DOCKER_STATE_DIR}/${service}.status")" == "running" ]]
EOF

cat >"${FAKE_BIN}/temporal" <<'EOF'
#!/usr/bin/env bash
echo 'temporal-host-log'
trap 'exit 0' TERM INT
while true; do sleep 1; done
EOF

chmod +x "${FAKE_BIN}"/*

fail() {
  printf 'not ok - %s\n' "$*" >&2
  if [[ -f "${LAST_OUTPUT}" ]]; then
    sed 's/^/  | /' "${LAST_OUTPUT}" >&2
  fi
  exit 1
}

pass() {
  TESTS=$((TESTS + 1))
  printf 'ok %d - %s\n' "${TESTS}" "$*"
}

assert_contains() {
  local file="$1" expected="$2"
  grep -Fq -- "${expected}" "${file}" || fail "expected ${file} to contain: ${expected}"
}

assert_not_contains() {
  local file="$1" unexpected="$2"
  if grep -Fq -- "${unexpected}" "${file}"; then
    fail "expected ${file} not to contain: ${unexpected}"
  fi
}

assert_file_value() {
  local file="$1" expected="$2" actual=""
  [[ -f "${file}" ]] || fail "missing file ${file}"
  actual="$(cat "${file}")"
  [[ "${actual}" == "${expected}" ]] || fail "${file}: got ${actual}, want ${expected}"
}

new_case() {
  local name="$1"
  CURRENT_STATE_DIR="${TEST_TMP}/state-${name}"
  mkdir -p "${CURRENT_STATE_DIR}"
  rm -f "${FAKE_DOCKER_STATE_DIR}"/* "${FAKE_RUNTIME_DIR}"/*
  : >"${FAKE_DOCKER_LOG}"
  : >"${FAKE_MAKE_LOG}"
  : >"${LAST_OUTPUT}"
}

run_ok() {
  if ! PATH="${FAKE_BIN}:${PATH}" \
    STATE_DIR="${CURRENT_STATE_DIR}" \
    AGENTCLASH_STACK_ROOT="${ROOT_DIR}" \
    AGENTCLASH_STACK_ENV_FILE="${TEST_ENV_FILE}" \
    STACK_DOCKER_WAIT_ATTEMPTS=1 \
    STACK_TEMPORAL_WAIT_ATTEMPTS=1 \
    STACK_API_WAIT_ATTEMPTS=1 \
    STACK_WAIT_INTERVAL=0 \
    STACK_STOP_WAIT_SECONDS=1 \
    "${CONTROLLER}" "$@" >"${LAST_OUTPUT}" 2>&1; then
    fail "command should succeed: $*"
  fi
}

run_fails() {
  if PATH="${FAKE_BIN}:${PATH}" \
    STATE_DIR="${CURRENT_STATE_DIR}" \
    AGENTCLASH_STACK_ROOT="${ROOT_DIR}" \
    AGENTCLASH_STACK_ENV_FILE="${TEST_ENV_FILE}" \
    STACK_DOCKER_WAIT_ATTEMPTS=1 \
    STACK_TEMPORAL_WAIT_ATTEMPTS=1 \
    STACK_API_WAIT_ATTEMPTS=1 \
    STACK_WAIT_INTERVAL=0 \
    STACK_STOP_WAIT_SECONDS=1 \
    "${CONTROLLER}" "$@" >"${LAST_OUTPUT}" 2>&1; then
    fail "command should fail: $*"
  fi
}

run_fails_from_root() {
  local stack_root="$1"
  shift
  if PATH="${FAKE_BIN}:${PATH}" \
    STATE_DIR="${CURRENT_STATE_DIR}" \
    AGENTCLASH_STACK_ROOT="${stack_root}" \
    AGENTCLASH_STACK_ENV_FILE="${TEST_ENV_FILE}" \
    STACK_STOP_WAIT_SECONDS=1 \
    "${CONTROLLER}" "$@" >"${LAST_OUTPUT}" 2>&1; then
    fail "command should fail from ${stack_root}: $*"
  fi
}

# Docker Temporal: start, repeat, inspect, aggregate logs, stop, and repeat stop.
new_case docker
run_ok start
assert_contains "${CURRENT_STATE_DIR}/state" 'lifecycle=running'
assert_contains "${CURRENT_STATE_DIR}/state" 'temporal_mode=docker'
api_pid="$(cat "${CURRENT_STATE_DIR}/api-server.pid")"
worker_pid="$(cat "${CURRENT_STATE_DIR}/worker.pid")"
assert_contains "${CURRENT_STATE_DIR}/state" "api_pid=${api_pid}"
assert_contains "${CURRENT_STATE_DIR}/state" "worker_pid=${worker_pid}"
assert_contains "${CURRENT_STATE_DIR}/state" "api_log=${CURRENT_STATE_DIR}/api-server.log"
run_ok start
assert_file_value "${CURRENT_STATE_DIR}/api-server.pid" "${api_pid}"
assert_file_value "${CURRENT_STATE_DIR}/worker.pid" "${worker_pid}"
assert_contains "${LAST_OUTPUT}" 'already running'
run_ok status
assert_contains "${LAST_OUTPUT}" 'All five stack services are operational.'
run_fails_from_root "${ROOT_DIR}/other-checkout" stop
assert_contains "${LAST_OUTPUT}" 'owned by another checkout'
kill -0 "${api_pid}" >/dev/null 2>&1 || fail 'cross-checkout stop killed the API process'
assert_not_contains "${FAKE_DOCKER_LOG}" 'compose stop'
FOLLOW=0 TAIL=10 run_ok logs
assert_contains "${LAST_OUTPUT}" 'postgres-log'
assert_contains "${LAST_OUTPUT}" '[api'
assert_contains "${LAST_OUTPUT}" '[worker'
PATH="${FAKE_BIN}:${PATH}" \
  STATE_DIR="${CURRENT_STATE_DIR}" \
  AGENTCLASH_STACK_ROOT="${ROOT_DIR}" \
  AGENTCLASH_STACK_ENV_FILE="${TEST_ENV_FILE}" \
  FOLLOW=1 TAIL=10 \
  "${CONTROLLER}" logs >"${LAST_OUTPUT}" 2>&1 &
logs_pid=$!
for ((i = 0; i < 50; i++)); do
  [[ -f "${FAKE_DOCKER_STATE_DIR}/logs-follower.pid" ]] && break
  sleep 0.02
done
[[ -f "${FAKE_DOCKER_STATE_DIR}/logs-follower.pid" ]] || fail 'Docker log follower did not start'
docker_logs_pid="$(cat "${FAKE_DOCKER_STATE_DIR}/logs-follower.pid")"
# Background jobs inherit SIGINT as ignored from non-interactive Bash, so use
# SIGTERM here to exercise the same cleanup trap that Ctrl-C reaches in the
# normal foreground workflow.
kill -TERM "${logs_pid}"
for ((i = 0; i < 50; i++)); do
  kill -0 "${logs_pid}" >/dev/null 2>&1 || break
  sleep 0.02
done
wait "${logs_pid}" 2>/dev/null || true
if kill -0 "${docker_logs_pid}" >/dev/null 2>&1; then
  fail 'interrupt left the Docker log follower running'
fi
run_ok stop
assert_contains "${FAKE_DOCKER_LOG}" 'compose stop -t 1 temporal redis postgres'
assert_not_contains "${FAKE_DOCKER_LOG}" 'compose down'
assert_contains "${CURRENT_STATE_DIR}/state" 'lifecycle=stopped'
run_fails status
run_ok stop
pass 'Docker lifecycle is idempotent and preserves Compose resources'

# The shared default namespace must never follow attacker-controlled artifacts.
new_case unsafe-state
victim="${TEST_TMP}/symlink-victim"
printf 'unchanged\n' >"${victim}"
ln -s "${victim}" "${CURRENT_STATE_DIR}/api-server.log"
run_fails start
assert_contains "${LAST_OUTPUT}" 'Refusing symbolic-link lifecycle artifact'
assert_file_value "${victim}" 'unchanged'
pass 'state directory rejects symbolic-link lifecycle artifacts'

# Restart must stop before bringing up a new process generation.
CURRENT_STATE_DIR="${TEST_TMP}/state-docker"
run_ok restart
new_api_pid="$(cat "${CURRENT_STATE_DIR}/api-server.pid")"
[[ "${new_api_pid}" != "${api_pid}" ]] || fail 'restart reused the API PID'
stop_line="$(grep -n 'compose stop -t 1 temporal redis postgres' "${FAKE_DOCKER_LOG}" | tail -n 1 | cut -d: -f1)"
up_line="$(grep -n 'compose up -d postgres redis' "${FAKE_DOCKER_LOG}" | tail -n 1 | cut -d: -f1)"
(( stop_line < up_line )) || fail 'restart did not stop before start'
run_ok stop
pass 'restart creates a clean generation in stop-then-start order'

# Docker Temporal failure must use and track the host CLI log/PID.
new_case fallback
: >"${FAKE_DOCKER_STATE_DIR}/temporal.fail"
run_ok start
assert_contains "${CURRENT_STATE_DIR}/state" 'temporal_mode=host'
[[ -f "${CURRENT_STATE_DIR}/temporal.pid" ]] || fail 'host fallback PID was not recorded'
FOLLOW=0 TAIL=10 run_ok logs
assert_contains "${LAST_OUTPUT}" '[temporal'
assert_contains "${LAST_OUTPUT}" 'temporal-host-log'
run_ok status
assert_contains "${LAST_OUTPUT}" 'Temporal     host'
run_ok stop
[[ ! -f "${CURRENT_STATE_DIR}/temporal.pid" ]] || fail 'host fallback PID file survived stop'
assert_contains "${FAKE_DOCKER_LOG}" 'compose stop -t 5 temporal'
pass 'Temporal host fallback is owned, logged, inspected, and stopped'

# Failed readiness remains partial and cannot be mixed with a second start.
new_case partial
: >"${FAKE_RUNTIME_DIR}/api-unready"
run_fails start
assert_contains "${CURRENT_STATE_DIR}/state" 'lifecycle=partial'
assert_contains "${LAST_OUTPUT}" 'API server did not become ready'
run_fails start
assert_contains "${LAST_OUTPUT}" 'previous startup left a partial stack generation'
rm -f "${FAKE_RUNTIME_DIR}/api-unready"
run_ok stop
pass 'partial startup remains inspectable and requires restart or stop'

# A foreign PID is reported and never signalled; stale files are removable.
new_case ownership
sleep 30 &
FOREIGN_PID=$!
printf '%s\n' "${FOREIGN_PID}" >"${CURRENT_STATE_DIR}/api-server.pid"
run_fails stop
kill -0 "${FOREIGN_PID}" >/dev/null 2>&1 || fail 'foreign PID was killed'
assert_contains "${LAST_OUTPUT}" 'Refusing to stop API server'
kill "${FOREIGN_PID}" >/dev/null 2>&1 || true
wait "${FOREIGN_PID}" 2>/dev/null || true
FOREIGN_PID=""
printf '99999999\n' >"${CURRENT_STATE_DIR}/worker.pid"
run_ok stop
[[ ! -f "${CURRENT_STATE_DIR}/worker.pid" ]] || fail 'stale PID file survived stop'
pass 'reused PIDs are protected and stale PID files are cleaned'

# An unexplained port occupant must block startup before Compose changes state.
new_case collision
: >"${FAKE_RUNTIME_DIR}/foreign-8080"
run_fails start
assert_contains "${LAST_OUTPUT}" 'API port 127.0.0.1:8080 is occupied by an unmanaged process'
assert_not_contains "${FAKE_DOCKER_LOG}" 'compose up -d postgres redis'
pass 'unmanaged port collisions fail before startup'

# A process that ignores TERM is killed after the bounded grace period.
new_case forced
: >"${FAKE_RUNTIME_DIR}/api.ignore-term"
run_ok start
stubborn_pid="$(cat "${CURRENT_STATE_DIR}/api-server.pid")"
run_ok stop
assert_contains "${LAST_OUTPUT}" 'did not stop gracefully; killing PID'
sleep 0.1
if kill -0 "${stubborn_pid}" >/dev/null 2>&1; then
  fail 'forced shutdown left the stubborn API process running'
fi
pass 'shutdown escalates only after the graceful timeout'

printf '1..%d\n' "${TESTS}"
