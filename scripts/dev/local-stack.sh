#!/usr/bin/env bash
#
# State-aware lifecycle controller for the local AgentClash backend stack.
#
# The web development server is intentionally not managed here. See
# CONTRIBUTING.md for its separate `web/.env.local` and `pnpm dev` workflow.
set -euo pipefail

ROOT_DIR="${AGENTCLASH_STACK_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
BACKEND_DIR="${ROOT_DIR}/backend"
STATE_DIR="${STATE_DIR:-/tmp/agentclash-local-stack}"
BIN_DIR="${STATE_DIR}/bin"
STATE_FILE="${STATE_DIR}/state"
LOCK_DIR="${STATE_DIR}/lock"

API_LOG="${STATE_DIR}/api-server.log"
WORKER_LOG="${STATE_DIR}/worker.log"
TEMPORAL_LOG="${STATE_DIR}/temporal.log"
API_PID_FILE="${STATE_DIR}/api-server.pid"
WORKER_PID_FILE="${STATE_DIR}/worker.pid"
TEMPORAL_PID_FILE="${STATE_DIR}/temporal.pid"
API_BIN="${BIN_DIR}/api-server"
WORKER_BIN="${BIN_DIR}/worker"

FOLLOW="${FOLLOW:-1}"
TAIL="${TAIL:-100}"
DOCKER_WAIT_ATTEMPTS="${STACK_DOCKER_WAIT_ATTEMPTS:-60}"
TEMPORAL_WAIT_ATTEMPTS="${STACK_TEMPORAL_WAIT_ATTEMPTS:-60}"
API_WAIT_ATTEMPTS="${STACK_API_WAIT_ATTEMPTS:-30}"
WAIT_INTERVAL="${STACK_WAIT_INTERVAL:-1}"
STOP_WAIT_SECONDS="${STACK_STOP_WAIT_SECONDS:-15}"

STACK_VERSION="1"
STACK_ROOT=""
STACK_GENERATION=""
STACK_LIFECYCLE="stopped"
TEMPORAL_MODE="none"
API_EXECUTABLE=""
WORKER_EXECUTABLE=""
TEMPORAL_EXECUTABLE=""
LOCK_HELD=0

note() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
ok() { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m!!\033[0m %s\n' "$*" >&2; }
err() { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; }

usage() {
  cat <<'EOF'
Usage: scripts/dev/local-stack.sh <start|status|logs|stop|restart>

  start    start Postgres, Redis, Temporal, the API server, and the worker
  status   report runtime ownership and health for every managed service
  logs     show all service logs (FOLLOW=0 for a snapshot, TAIL=100 by default)
  stop     gracefully stop host processes and Docker Compose services
  restart  stop and then start the complete stack

The web development server is separate: cd web && pnpm dev
EOF
}

load_backend_env() {
  local env_file=""
  if [[ -n "${AGENTCLASH_STACK_ENV_FILE:-}" ]]; then
    if [[ ! -f "${AGENTCLASH_STACK_ENV_FILE}" ]]; then
      err "Configured stack env file does not exist: ${AGENTCLASH_STACK_ENV_FILE}"
      return 1
    fi
    env_file="${AGENTCLASH_STACK_ENV_FILE}"
  elif [[ -f "${BACKEND_DIR}/.env" ]]; then
    env_file="${BACKEND_DIR}/.env"
  elif [[ -f "${BACKEND_DIR}/.env.example" ]]; then
    env_file="${BACKEND_DIR}/.env.example"
  fi

  if [[ -n "${env_file}" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "${env_file}"
    set +a
  fi

  export OPENAI_MODEL="${OPENAI_MODEL:-gpt-4.1-mini}"
  export TEMPORAL_HOST_PORT="${TEMPORAL_HOST_PORT:-localhost:7233}"
  export TEMPORAL_NAMESPACE="${TEMPORAL_NAMESPACE:-default}"
  export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"
  export GOCACHE="${GOCACHE:-/tmp/go-build}"

  if [[ -n "${OPENAI_API_KEY:-}" && -z "${AGENTCLASH_SECRET_OPENAI:-}" ]]; then
    export AGENTCLASH_SECRET_OPENAI="${OPENAI_API_KEY}"
  fi
}

load_state() {
  [[ -f "${STATE_FILE}" ]] || return 0

  local key value
  while IFS='=' read -r key value; do
    case "${key}" in
      version) STACK_VERSION="${value}" ;;
      root) STACK_ROOT="${value}" ;;
      generation) STACK_GENERATION="${value}" ;;
      lifecycle) STACK_LIFECYCLE="${value}" ;;
      temporal_mode) TEMPORAL_MODE="${value}" ;;
      api_executable) API_EXECUTABLE="${value}" ;;
      worker_executable) WORKER_EXECUTABLE="${value}" ;;
      temporal_executable) TEMPORAL_EXECUTABLE="${value}" ;;
    esac
  done <"${STATE_FILE}"
}

save_state() {
  mkdir -p "${STATE_DIR}"
  local tmp="${STATE_FILE}.tmp.$$" api_pid="" worker_pid="" temporal_pid=""
  api_pid="$(read_pid "${API_PID_FILE}" 2>/dev/null || true)"
  worker_pid="$(read_pid "${WORKER_PID_FILE}" 2>/dev/null || true)"
  temporal_pid="$(read_pid "${TEMPORAL_PID_FILE}" 2>/dev/null || true)"
  umask 077
  {
    printf 'version=%s\n' "${STACK_VERSION}"
    printf 'root=%s\n' "${STACK_ROOT:-${ROOT_DIR}}"
    printf 'generation=%s\n' "${STACK_GENERATION}"
    printf 'lifecycle=%s\n' "${STACK_LIFECYCLE}"
    printf 'temporal_mode=%s\n' "${TEMPORAL_MODE}"
    printf 'api_executable=%s\n' "${API_EXECUTABLE}"
    printf 'worker_executable=%s\n' "${WORKER_EXECUTABLE}"
    printf 'temporal_executable=%s\n' "${TEMPORAL_EXECUTABLE}"
    printf 'api_pid=%s\n' "${api_pid}"
    printf 'worker_pid=%s\n' "${worker_pid}"
    printf 'temporal_pid=%s\n' "${temporal_pid}"
    printf 'api_log=%s\n' "${API_LOG}"
    printf 'worker_log=%s\n' "${WORKER_LOG}"
    printf 'temporal_log=%s\n' "${TEMPORAL_LOG}"
  } >"${tmp}"
  mv "${tmp}" "${STATE_FILE}"
}

write_pid() {
  local file="$1" pid="$2" tmp
  tmp="${file}.tmp.$$"
  printf '%s\n' "${pid}" >"${tmp}"
  mv "${tmp}" "${file}"
}

read_pid() {
  local file="$1" pid=""
  [[ -f "${file}" ]] || return 1
  IFS= read -r pid <"${file}" || true
  [[ "${pid}" =~ ^[1-9][0-9]*$ ]] || return 1
  printf '%s\n' "${pid}"
}

process_alive() {
  local pid="$1" state=""
  kill -0 "${pid}" >/dev/null 2>&1 || return 1
  state="$(ps -p "${pid}" -o stat= 2>/dev/null | awk '{print $1}')"
  [[ -n "${state}" && "${state}" != Z* ]]
}

process_command() {
  ps -ww -p "$1" -o command= 2>/dev/null | awk '{$1=$1; print}'
}

process_matches_executable() {
  local pid="$1" expected="$2" command=""
  [[ -n "${expected}" ]] || return 1
  process_alive "${pid}" || return 1
  command="$(process_command "${pid}")"
  case "${command}" in
    "${expected}"|"${expected} "*|*"bash ${expected}"|*"bash ${expected} "*) return 0 ;;
    *) return 1 ;;
  esac
}

legacy_process_matches() {
  local role="$1" pid="$2" command=""
  process_alive "${pid}" || return 1
  command="$(process_command "${pid}")"
  case "${role}:${command}" in
    api:*"go run ./cmd/api-server"*) return 0 ;;
    worker:*"go run ./cmd/worker"*) return 0 ;;
    temporal:*"temporal server start-dev --ip 127.0.0.1 --port 7233"*) return 0 ;;
    *) return 1 ;;
  esac
}

inspect_host_process() {
  # Sets PROCESS_STATE, PROCESS_PID, and PROCESS_DETAIL.
  local role="$1" pid_file="$2" expected="$3" pid=""
  PROCESS_STATE="stopped"
  PROCESS_PID="-"
  PROCESS_DETAIL="no PID file"

  if [[ ! -f "${pid_file}" ]]; then
    return 0
  fi
  if ! pid="$(read_pid "${pid_file}")"; then
    PROCESS_STATE="stale"
    PROCESS_DETAIL="invalid PID file ${pid_file}"
    return 0
  fi

  PROCESS_PID="${pid}"
  if ! process_alive "${pid}"; then
    PROCESS_STATE="stale"
    PROCESS_DETAIL="PID ${pid} is not running"
  elif process_matches_executable "${pid}" "${expected}"; then
    PROCESS_STATE="managed"
    PROCESS_DETAIL="PID ${pid}"
  elif legacy_process_matches "${role}" "${pid}"; then
    PROCESS_STATE="legacy"
    PROCESS_DETAIL="legacy PID ${pid}"
  else
    PROCESS_STATE="foreign"
    PROCESS_DETAIL="PID ${pid} was reused: $(process_command "${pid}")"
  fi
}

cleanup_dead_pid_file() {
  local file="$1" pid=""
  [[ -f "${file}" ]] || return 0
  if ! pid="$(read_pid "${file}")" || ! process_alive "${pid}"; then
    rm -f "${file}"
  fi
}

port_open() {
  local host="$1" port="$2"

  if command -v timeout >/dev/null 2>&1; then
    timeout 1 bash -c "exec 3<>/dev/tcp/${host}/${port}" >/dev/null 2>&1
    return $?
  fi

  ( exec 3<>"/dev/tcp/${host}/${port}" ) >/dev/null 2>&1 &
  local probe=$!
  ( sleep 1 && kill -9 "${probe}" ) >/dev/null 2>&1 &
  local watchdog=$!
  local rc=0
  wait "${probe}" 2>/dev/null || rc=1
  kill "${watchdog}" >/dev/null 2>&1 || true
  wait "${watchdog}" 2>/dev/null || true
  return "${rc}"
}

temporal_target() {
  TEMPORAL_HOST="${TEMPORAL_HOST_PORT%:*}"
  TEMPORAL_PORT="${TEMPORAL_HOST_PORT##*:}"
  [[ "${TEMPORAL_HOST}" != "${TEMPORAL_HOST_PORT}" ]] || TEMPORAL_HOST="localhost"
}

api_target() {
  local bind="${API_SERVER_BIND_ADDRESS:-:8080}"
  API_PORT="${bind##*:}"
  API_HOST="${bind%:*}"
  [[ "${API_HOST}" != "${bind}" && -n "${API_HOST}" ]] || API_HOST="127.0.0.1"
  case "${API_HOST}" in
    0.0.0.0|localhost) API_HOST="127.0.0.1" ;;
  esac
}

compose() {
  (cd "${ROOT_DIR}" && docker compose "$@")
}

docker_service_info() {
  # Sets DOCKER_STATE, DOCKER_HEALTH, and DOCKER_ID.
  local service="$1" inspection=""
  DOCKER_STATE="missing"
  DOCKER_HEALTH="none"
  DOCKER_ID=""

  if ! DOCKER_ID="$(compose ps -a -q "${service}" 2>/dev/null | head -n 1)"; then
    DOCKER_STATE="unavailable"
    return 1
  fi
  [[ -n "${DOCKER_ID}" ]] || return 0

  if ! inspection="$(docker inspect --format '{{.State.Status}}|{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "${DOCKER_ID}" 2>/dev/null)"; then
    DOCKER_STATE="unknown"
    return 1
  fi
  DOCKER_STATE="${inspection%%|*}"
  DOCKER_HEALTH="${inspection#*|}"
}

docker_service_healthy() {
  docker_service_info "$1" >/dev/null 2>&1 || return 1
  [[ "${DOCKER_STATE}" == "running" && "${DOCKER_HEALTH}" == "healthy" ]]
}

docker_service_running() {
  docker_service_info "$1" >/dev/null 2>&1 || return 1
  [[ "${DOCKER_STATE}" == "running" ]]
}

wait_for_docker_health() {
  local service="$1" attempts="$2" i
  for ((i = 1; i <= attempts; i++)); do
    if docker_service_healthy "${service}"; then
      return 0
    fi
    sleep "${WAIT_INTERVAL}"
  done
  return 1
}

wait_for_port() {
  local host="$1" port="$2" attempts="$3" i
  for ((i = 1; i <= attempts; i++)); do
    if port_open "${host}" "${port}"; then
      return 0
    fi
    sleep "${WAIT_INTERVAL}"
  done
  return 1
}

api_ready() {
  local code=""
  api_target
  code="$(curl -s -o /dev/null -w '%{http_code}' "http://${API_HOST}:${API_PORT}/healthz/ready" 2>/dev/null || true)"
  [[ "${code}" == "200" ]]
}

wait_for_api() {
  local attempts="$1" i
  for ((i = 1; i <= attempts; i++)); do
    if api_ready; then
      return 0
    fi
    sleep "${WAIT_INTERVAL}"
  done
  return 1
}

acquire_lock() {
  mkdir -p "${STATE_DIR}"
  if mkdir "${LOCK_DIR}" 2>/dev/null; then
    printf '%s\n' "$$" >"${LOCK_DIR}/pid"
    printf '%s\n' "${ROOT_DIR}" >"${LOCK_DIR}/root"
    LOCK_HELD=1
    return 0
  fi

  local owner=""
  if [[ -f "${LOCK_DIR}/pid" ]]; then
    IFS= read -r owner <"${LOCK_DIR}/pid" || true
  fi
  if [[ "${owner}" =~ ^[1-9][0-9]*$ ]] && process_alive "${owner}"; then
    case "$(process_command "${owner}")" in
      *"scripts/dev/local-stack.sh"*)
        err "Another local-stack operation is running (PID ${owner})."
        return 1
        ;;
      *)
        warn "Recovering a stale lifecycle lock whose PID was reused (${owner})."
        ;;
    esac
  fi

  rm -f "${LOCK_DIR}/pid" "${LOCK_DIR}/root"
  if ! rmdir "${LOCK_DIR}" 2>/dev/null || ! mkdir "${LOCK_DIR}" 2>/dev/null; then
    err "Cannot recover stale lifecycle lock at ${LOCK_DIR}."
    return 1
  fi
  printf '%s\n' "$$" >"${LOCK_DIR}/pid"
  printf '%s\n' "${ROOT_DIR}" >"${LOCK_DIR}/root"
  LOCK_HELD=1
}

release_lock() {
  [[ "${LOCK_HELD}" == "1" ]] || return 0
  local owner=""
  if [[ -f "${LOCK_DIR}/pid" ]]; then
    IFS= read -r owner <"${LOCK_DIR}/pid" || true
  fi
  if [[ "${owner}" == "$$" ]]; then
    rm -f "${LOCK_DIR}/pid" "${LOCK_DIR}/root"
    rmdir "${LOCK_DIR}" 2>/dev/null || true
  fi
  LOCK_HELD=0
}

append_log_marker() {
  local file="$1" service="$2"
  {
    printf '\n=== %s generation %s started %s ===\n' \
      "${service}" "${STACK_GENERATION}" "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  } >>"${file}"
}

start_failure() {
  STACK_LIFECYCLE="partial"
  save_state
  err "$*"
  printf '\nInspect the partial stack with:\n' >&2
  printf '  make status\n  make logs FOLLOW=0\n' >&2
  printf 'Recover with `make restart`, or turn everything off with `make stop`.\n' >&2
  return 1
}

temporal_is_operational() {
  temporal_target
  case "${TEMPORAL_MODE}" in
    host)
      inspect_host_process temporal "${TEMPORAL_PID_FILE}" "${TEMPORAL_EXECUTABLE}"
      [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]] && \
        port_open "${TEMPORAL_HOST}" "${TEMPORAL_PORT}"
      ;;
    external)
      port_open "${TEMPORAL_HOST}" "${TEMPORAL_PORT}"
      ;;
    docker)
      docker_service_healthy temporal
      ;;
    *)
      docker_service_healthy temporal
      ;;
  esac
}

stack_operational() {
  docker_service_healthy postgres || return 1
  docker_service_healthy redis || return 1
  temporal_is_operational || return 1

  inspect_host_process api "${API_PID_FILE}" "${API_EXECUTABLE:-${API_BIN}}"
  [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]] || return 1
  api_ready || return 1

  inspect_host_process worker "${WORKER_PID_FILE}" "${WORKER_EXECUTABLE:-${WORKER_BIN}}"
  [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]]
}

any_host_process_present() {
  inspect_host_process api "${API_PID_FILE}" "${API_EXECUTABLE:-${API_BIN}}"
  [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" || "${PROCESS_STATE}" == "foreign" ]] && return 0
  inspect_host_process worker "${WORKER_PID_FILE}" "${WORKER_EXECUTABLE:-${WORKER_BIN}}"
  [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" || "${PROCESS_STATE}" == "foreign" ]] && return 0
  inspect_host_process temporal "${TEMPORAL_PID_FILE}" "${TEMPORAL_EXECUTABLE}"
  [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" || "${PROCESS_STATE}" == "foreign" ]]
}

ensure_port_available_for_compose() {
  local service="$1" host="$2" port="$3"
  if port_open "${host}" "${port}" && ! docker_service_running "${service}"; then
    err "Port ${host}:${port} is occupied by something outside this Compose service (${service})."
    return 1
  fi
}

build_host_binaries() {
  mkdir -p "${BIN_DIR}"
  note "Building API server and worker"
  if ! (cd "${BACKEND_DIR}" && go build -o "${API_BIN}.tmp" ./cmd/api-server); then
    rm -f "${API_BIN}.tmp"
    return 1
  fi
  if ! (cd "${BACKEND_DIR}" && go build -o "${WORKER_BIN}.tmp" ./cmd/worker); then
    rm -f "${API_BIN}.tmp" "${WORKER_BIN}.tmp"
    return 1
  fi
  mv "${API_BIN}.tmp" "${API_BIN}"
  mv "${WORKER_BIN}.tmp" "${WORKER_BIN}"
  API_EXECUTABLE="${API_BIN}"
  WORKER_EXECUTABLE="${WORKER_BIN}"
  save_state
}

start_temporal() {
  temporal_target

  if [[ "${TEMPORAL_HOST}" != "localhost" && "${TEMPORAL_HOST}" != "127.0.0.1" ]] || [[ "${TEMPORAL_PORT}" != "7233" ]]; then
    note "Using configured external Temporal at ${TEMPORAL_HOST_PORT}"
    if ! wait_for_port "${TEMPORAL_HOST}" "${TEMPORAL_PORT}" "${TEMPORAL_WAIT_ATTEMPTS}"; then
      return 1
    fi
    TEMPORAL_MODE="external"
    TEMPORAL_EXECUTABLE=""
    save_state
    return 0
  fi

  if docker_service_healthy temporal; then
    TEMPORAL_MODE="docker"
    TEMPORAL_EXECUTABLE=""
    save_state
    ok "Temporal is healthy in Docker"
    return 0
  fi

  inspect_host_process temporal "${TEMPORAL_PID_FILE}" "${TEMPORAL_EXECUTABLE}"
  if [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]]; then
    if port_open "${TEMPORAL_HOST}" "${TEMPORAL_PORT}"; then
      TEMPORAL_MODE="host"
      save_state
      ok "Temporal host fallback is already running"
      return 0
    fi
    err "A managed Temporal process exists but is not reachable on ${TEMPORAL_HOST_PORT}."
    return 1
  fi

  if port_open "${TEMPORAL_HOST}" "${TEMPORAL_PORT}" && ! docker_service_running temporal; then
    err "Port ${TEMPORAL_HOST_PORT} is occupied by an unmanaged process; refusing to adopt it."
    return 1
  fi

  note "Pulling the Temporal image (the first run can take a few minutes)"
  compose pull temporal || warn "Temporal image pull failed; trying the available local image."

  note "Starting Temporal in Docker"
  if compose up -d temporal && wait_for_docker_health temporal "${TEMPORAL_WAIT_ATTEMPTS}"; then
    TEMPORAL_MODE="docker"
    TEMPORAL_EXECUTABLE=""
    save_state
    ok "Temporal is healthy in Docker"
    return 0
  fi

  warn "Docker Temporal did not become healthy; stopping it before host fallback."
  compose stop -t 5 temporal >/dev/null 2>&1 || true

  local temporal_bin=""
  temporal_bin="$(command -v temporal 2>/dev/null || true)"
  if [[ -z "${temporal_bin}" ]]; then
    err "Temporal failed in Docker and the optional host CLI is not installed."
    return 1
  fi

  note "Starting Temporal with the host CLI fallback"
  TEMPORAL_MODE="host"
  TEMPORAL_EXECUTABLE="${temporal_bin}"
  append_log_marker "${TEMPORAL_LOG}" temporal
  nohup "${temporal_bin}" server start-dev \
    --ip 127.0.0.1 \
    --port "${TEMPORAL_PORT}" \
    --namespace "${TEMPORAL_NAMESPACE}" \
    >>"${TEMPORAL_LOG}" 2>&1 &
  write_pid "${TEMPORAL_PID_FILE}" "$!"
  save_state

  if ! wait_for_port "${TEMPORAL_HOST}" "${TEMPORAL_PORT}" "${TEMPORAL_WAIT_ATTEMPTS}"; then
    return 1
  fi
  inspect_host_process temporal "${TEMPORAL_PID_FILE}" "${TEMPORAL_EXECUTABLE}"
  [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]]
}

start_stack() {
  load_state
  cleanup_dead_pid_file "${API_PID_FILE}"
  cleanup_dead_pid_file "${WORKER_PID_FILE}"
  cleanup_dead_pid_file "${TEMPORAL_PID_FILE}"

  if stack_operational; then
    if [[ -n "${STACK_ROOT}" && "${STACK_ROOT}" != "${ROOT_DIR}" ]]; then
      err "A healthy stack is already running from ${STACK_ROOT}."
      return 1
    fi
    STACK_ROOT="${ROOT_DIR}"
    STACK_LIFECYCLE="running"
    [[ "${TEMPORAL_MODE}" != "none" ]] || TEMPORAL_MODE="docker"
    save_state
    ok "Local stack is already running."
    printf 'Run `make status` for details or `make logs` to follow output.\n'
    return 0
  fi

  if [[ "${STACK_LIFECYCLE}" == "partial" || "${STACK_LIFECYCLE}" == "starting" ]]; then
    err "The previous startup left a ${STACK_LIFECYCLE} stack generation."
    printf 'Run `make status` and `make logs FOLLOW=0`, then use `make restart` to recover.\n' >&2
    return 1
  fi

  if any_host_process_present; then
    err "A partial, foreign, or legacy host stack is already present."
    printf 'Run `make status`, then use `make restart` to avoid mixing process generations.\n' >&2
    return 1
  fi

  if [[ -n "${STACK_ROOT}" && "${STACK_ROOT}" != "${ROOT_DIR}" && "${STACK_LIFECYCLE}" != "stopped" ]]; then
    err "Stack state belongs to another checkout: ${STACK_ROOT}"
    printf 'Use that checkout to inspect it, or run `make stop` before starting this one.\n' >&2
    return 1
  fi

  if ! compose ps >/dev/null 2>&1; then
    err "Docker Compose cannot reach the Docker daemon. Start Docker and retry."
    return 1
  fi
  ensure_port_available_for_compose postgres 127.0.0.1 5432 || return 1
  ensure_port_available_for_compose redis 127.0.0.1 6379 || return 1

  api_target
  if port_open "${API_HOST}" "${API_PORT}"; then
    err "API port ${API_HOST}:${API_PORT} is occupied by an unmanaged process."
    return 1
  fi

  mkdir -p "${STATE_DIR}" "${BIN_DIR}"
  STACK_ROOT="${ROOT_DIR}"
  STACK_GENERATION="$(date -u '+%Y%m%dT%H%M%SZ')-$$"
  STACK_LIFECYCLE="starting"
  TEMPORAL_MODE="none"
  TEMPORAL_EXECUTABLE=""
  save_state

  build_host_binaries || { start_failure "Failed to build the API server or worker."; return 1; }

  note "Starting Postgres and Redis"
  if ! compose up -d postgres redis; then
    start_failure "Docker Compose could not start Postgres and Redis."
    return 1
  fi
  if ! wait_for_docker_health postgres "${DOCKER_WAIT_ATTEMPTS}"; then
    start_failure "Postgres did not become healthy. Run: docker compose logs postgres"
    return 1
  fi
  if ! wait_for_docker_health redis "${DOCKER_WAIT_ATTEMPTS}"; then
    start_failure "Redis did not become healthy. Run: docker compose logs redis"
    return 1
  fi

  note "Applying database migrations"
  if ! (cd "${ROOT_DIR}" && make --no-print-directory db-migrate); then
    start_failure "Database migrations failed."
    return 1
  fi

  if ! start_temporal; then
    start_failure "Temporal did not become ready at ${TEMPORAL_HOST_PORT}."
    return 1
  fi

  append_log_marker "${API_LOG}" api-server
  append_log_marker "${WORKER_LOG}" worker

  note "Starting API server"
  (cd "${BACKEND_DIR}" && exec nohup "${API_BIN}") >>"${API_LOG}" 2>&1 &
  write_pid "${API_PID_FILE}" "$!"
  save_state

  note "Starting worker"
  (cd "${BACKEND_DIR}" && exec nohup "${WORKER_BIN}") >>"${WORKER_LOG}" 2>&1 &
  write_pid "${WORKER_PID_FILE}" "$!"
  save_state

  if ! wait_for_api "${API_WAIT_ATTEMPTS}"; then
    start_failure "API server did not become ready. See ${API_LOG}"
    return 1
  fi
  inspect_host_process worker "${WORKER_PID_FILE}" "${WORKER_EXECUTABLE}"
  if [[ "${PROCESS_STATE}" != "managed" ]]; then
    start_failure "Worker exited during startup. See ${WORKER_LOG}"
    return 1
  fi

  STACK_LIFECYCLE="running"
  save_state
  echo
  ok "Local stack is running."
  printf '  Inspect:  make status\n'
  printf '  Logs:     make logs\n'
  printf '  Stop:     make stop\n'
  printf '  API:      http://%s:%s\n' "${API_HOST}" "${API_PORT}"
  if [[ "${TEMPORAL_MODE}" == "docker" ]]; then
    printf '  Temporal: http://localhost:8233 (Docker)\n'
  else
    printf '  Temporal: %s (%s)\n' "${TEMPORAL_HOST_PORT}" "${TEMPORAL_MODE}"
  fi
  echo
  printf 'The web server is separate: cd web && pnpm dev\n'
}

print_row() {
  printf '  %-12s %-10s %-14s %s\n' "$1" "$2" "$3" "$4"
}

status_docker_service() {
  local label="$1" service="$2" host="$3" port="$4"
  docker_service_info "${service}" >/dev/null 2>&1 || true
  case "${DOCKER_STATE}:${DOCKER_HEALTH}" in
    running:healthy)
      print_row "${label}" docker healthy "container ${DOCKER_ID}; docker compose logs ${service}"
      return 0
      ;;
    running:*)
      print_row "${label}" docker "${DOCKER_HEALTH}" "container ${DOCKER_ID}; state ${DOCKER_STATE}"
      return 1
      ;;
    unavailable:*)
      print_row "${label}" docker unavailable "Docker daemon is not reachable"
      return 1
      ;;
    *)
      if port_open "${host}" "${port}"; then
        print_row "${label}" external occupied "${host}:${port}; not managed by Compose"
      else
        print_row "${label}" docker stopped "${DOCKER_STATE}; docker compose logs ${service}"
      fi
      return 1
      ;;
  esac
}

status_stack() {
  load_state
  temporal_target
  api_target
  local failed=0

  printf 'AgentClash local stack\n'
  printf '  State:      %s\n' "${STACK_LIFECYCLE}"
  printf '  Generation: %s\n' "${STACK_GENERATION:--}"
  printf '  Owner:      %s\n' "${STACK_ROOT:--}"
  printf '  State dir:  %s\n\n' "${STATE_DIR}"
  printf '  %-12s %-10s %-14s %s\n' SERVICE RUNTIME STATE DETAILS

  status_docker_service Postgres postgres 127.0.0.1 5432 || failed=1
  status_docker_service Redis redis 127.0.0.1 6379 || failed=1

  case "${TEMPORAL_MODE}" in
    host)
      inspect_host_process temporal "${TEMPORAL_PID_FILE}" "${TEMPORAL_EXECUTABLE}"
      if [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]] && \
        port_open "${TEMPORAL_HOST}" "${TEMPORAL_PORT}"; then
        print_row Temporal host reachable "${PROCESS_DETAIL}; ${TEMPORAL_LOG}"
      else
        print_row Temporal host "${PROCESS_STATE}" "${PROCESS_DETAIL}; ${TEMPORAL_LOG}"
        failed=1
      fi
      ;;
    external)
      if port_open "${TEMPORAL_HOST}" "${TEMPORAL_PORT}"; then
        print_row Temporal external reachable "${TEMPORAL_HOST_PORT}; logs are externally managed"
      else
        print_row Temporal external unreachable "${TEMPORAL_HOST_PORT}"
        failed=1
      fi
      ;;
    *)
      status_docker_service Temporal temporal "${TEMPORAL_HOST}" "${TEMPORAL_PORT}" || failed=1
      ;;
  esac

  inspect_host_process api "${API_PID_FILE}" "${API_EXECUTABLE:-${API_BIN}}"
  if [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]]; then
    if api_ready; then
      print_row API host ready "${PROCESS_DETAIL}; ${API_LOG}"
    else
      print_row API host unready "${PROCESS_DETAIL}; ${API_LOG}"
      failed=1
    fi
  elif [[ "${PROCESS_STATE}" == "stopped" || "${PROCESS_STATE}" == "stale" ]] && port_open "${API_HOST}" "${API_PORT}"; then
    print_row API external occupied "${API_HOST}:${API_PORT}; no managed PID"
    failed=1
  else
    print_row API host "${PROCESS_STATE}" "${PROCESS_DETAIL}; ${API_LOG}"
    failed=1
  fi

  inspect_host_process worker "${WORKER_PID_FILE}" "${WORKER_EXECUTABLE:-${WORKER_BIN}}"
  if [[ "${PROCESS_STATE}" == "managed" || "${PROCESS_STATE}" == "legacy" ]]; then
    print_row Worker host running "${PROCESS_DETAIL}; process liveness only; ${WORKER_LOG}"
  else
    print_row Worker host "${PROCESS_STATE}" "${PROCESS_DETAIL}; ${WORKER_LOG}"
    failed=1
  fi

  echo
  if [[ "${failed}" == "0" ]]; then
    ok "All five stack services are operational."
    if [[ "${TEMPORAL_MODE}" != "external" ]]; then
      printf 'Temporal UI: http://localhost:8233\n'
    fi
    printf 'Web (separate): cd web && pnpm dev\n'
    return 0
  fi

  err "The local stack is not fully operational."
  printf 'Use `make logs FOLLOW=0` to inspect failures, `make restart` to recover, or `make stop` to turn it off.\n' >&2
  return 1
}

prefix_stream() {
  local label="$1" line=""
  while IFS= read -r line || [[ -n "${line}" ]]; do
    printf '[%-10s] %s\n' "${label}" "${line}"
  done
}

snapshot_host_log() {
  local label="$1" file="$2"
  if [[ -f "${file}" ]]; then
    tail -n "${TAIL}" "${file}" | prefix_stream "${label}"
    return 0
  fi
  warn "No ${label} log exists yet (${file})."
  return 1
}

follow_host_log() {
  # Polling avoids orphaning a `tail -F` child when the caller presses Ctrl-C.
  local label="$1" file="$2" seen=0 current=0 start=1 sleeper=""
  [[ -f "${file}" ]] || return 0

  trap 'if [[ -n "${sleeper:-}" ]]; then kill "${sleeper}" >/dev/null 2>&1 || true; fi; exit 0' INT TERM

  tail -n "${TAIL}" "${file}" | prefix_stream "${label}"
  seen="$(wc -l <"${file}" | awk '{print $1}')"
  while true; do
    sleep 1 &
    sleeper=$!
    wait "${sleeper}" 2>/dev/null || true
    sleeper=""
    [[ -f "${file}" ]] || continue
    current="$(wc -l <"${file}" | awk '{print $1}')"
    if (( current < seen )); then
      seen=0
    fi
    if (( current > seen )); then
      start=$((seen + 1))
      sed -n "${start},${current}p" "${file}" | prefix_stream "${label}"
      seen="${current}"
    fi
  done
}

logs_stack() {
  load_state
  if [[ ! "${TAIL}" =~ ^[1-9][0-9]*$ ]]; then
    err "TAIL must be a positive integer (got ${TAIL})."
    return 2
  fi
  if [[ "${FOLLOW}" != "0" && "${FOLLOW}" != "1" ]]; then
    err "FOLLOW must be 0 or 1 (got ${FOLLOW})."
    return 2
  fi

  local docker_services=(postgres redis)
  if [[ "${TEMPORAL_MODE}" != "host" && "${TEMPORAL_MODE}" != "external" ]]; then
    docker_services+=(temporal)
  fi

  if [[ "${FOLLOW}" == "0" ]]; then
    note "Docker logs"
    compose logs --no-color --tail "${TAIL}" "${docker_services[@]}" 2>&1 || warn "Docker logs are unavailable."
    echo
    note "Host logs"
    snapshot_host_log api "${API_LOG}" || true
    snapshot_host_log worker "${WORKER_LOG}" || true
    if [[ "${TEMPORAL_MODE}" == "host" ]]; then
      snapshot_host_log temporal "${TEMPORAL_LOG}" || true
    fi
    return 0
  fi

  note "Following local stack logs (Ctrl-C stops only the log followers)"
  local followers=()
  # Exec Compose in the background process itself. Running the `compose`
  # wrapper here would add a nested shell; killing that wrapper on Ctrl-C can
  # otherwise orphan the real `docker compose logs --follow` process.
  (
    cd "${ROOT_DIR}"
    exec docker compose logs --no-color --follow --tail "${TAIL}" "${docker_services[@]}"
  ) 2>&1 &
  followers+=("$!")
  follow_host_log api "${API_LOG}" &
  followers+=("$!")
  follow_host_log worker "${WORKER_LOG}" &
  followers+=("$!")
  if [[ "${TEMPORAL_MODE}" == "host" ]]; then
    follow_host_log temporal "${TEMPORAL_LOG}" &
    followers+=("$!")
  fi

  cleanup_log_followers() {
    local pid
    trap - INT TERM EXIT
    for pid in "${followers[@]}"; do
      kill "${pid}" >/dev/null 2>&1 || true
    done
    for pid in "${followers[@]}"; do
      wait "${pid}" 2>/dev/null || true
    done
  }
  trap cleanup_log_followers INT TERM EXIT
  wait "${followers[@]}" 2>/dev/null || true
  cleanup_log_followers
}

descendant_pids() {
  local parent="$1" child
  while IFS= read -r child; do
    [[ "${child}" =~ ^[1-9][0-9]*$ ]] || continue
    printf '%s\n' "${child}"
    descendant_pids "${child}"
  done < <(ps -eo pid=,ppid= 2>/dev/null | awk -v parent="${parent}" '$2 == parent {print $1}')
}

stop_host_process() {
  local label="$1" role="$2" pid_file="$3" expected="$4"
  local pid="" candidate command i index reused=0
  local targets=() commands=()

  [[ -f "${pid_file}" ]] || { note "${label} is already stopped"; return 0; }
  if ! pid="$(read_pid "${pid_file}")"; then
    warn "Removing invalid ${label} PID file: ${pid_file}"
    rm -f "${pid_file}"
    return 0
  fi
  if ! process_alive "${pid}"; then
    warn "Removing stale ${label} PID ${pid}"
    rm -f "${pid_file}"
    return 0
  fi
  if ! process_matches_executable "${pid}" "${expected}" && ! legacy_process_matches "${role}" "${pid}"; then
    err "Refusing to stop ${label}: PID ${pid} belongs to $(process_command "${pid}")"
    return 1
  fi

  targets+=("${pid}")
  commands+=("$(process_command "${pid}")")
  while IFS= read -r candidate; do
    [[ "${candidate}" =~ ^[1-9][0-9]*$ ]] || continue
    if process_alive "${candidate}"; then
      targets+=("${candidate}")
      commands+=("$(process_command "${candidate}")")
    fi
  done < <(descendant_pids "${pid}")

  note "Stopping ${label} (PID ${pid})"
  for index in "${!targets[@]}"; do
    candidate="${targets[${index}]}"
    command="${commands[${index}]}"
    if process_alive "${candidate}" && [[ "$(process_command "${candidate}")" == "${command}" ]]; then
      kill -TERM "${candidate}" >/dev/null 2>&1 || true
    fi
  done

  for ((i = 0; i < STOP_WAIT_SECONDS; i++)); do
    local live=0
    for candidate in "${targets[@]}"; do
      if process_alive "${candidate}"; then
        live=1
        break
      fi
    done
    [[ "${live}" == "0" ]] && break
    sleep 1
  done

  for index in "${!targets[@]}"; do
    candidate="${targets[${index}]}"
    command="${commands[${index}]}"
    if process_alive "${candidate}"; then
      if [[ "$(process_command "${candidate}")" == "${command}" ]]; then
        warn "${label} did not stop gracefully; killing PID ${candidate}."
        kill -KILL "${candidate}" >/dev/null 2>&1 || true
      else
        err "Refusing to kill reused PID ${candidate} while stopping ${label}."
        reused=1
      fi
    fi
  done
  if [[ "${reused}" == "0" ]]; then
    rm -f "${pid_file}"
    return 0
  fi
  return 1
}

stop_stack() {
  load_state
  local failed=0

  stop_host_process "API server" api "${API_PID_FILE}" "${API_EXECUTABLE:-${API_BIN}}" || failed=1
  stop_host_process Worker worker "${WORKER_PID_FILE}" "${WORKER_EXECUTABLE:-${WORKER_BIN}}" || failed=1
  if [[ "${TEMPORAL_MODE}" == "host" || -f "${TEMPORAL_PID_FILE}" ]]; then
    stop_host_process "Temporal host fallback" temporal "${TEMPORAL_PID_FILE}" "${TEMPORAL_EXECUTABLE}" || failed=1
  fi

  note "Stopping Docker services (containers, volumes, and logs are retained)"
  if ! compose stop -t "${STOP_WAIT_SECONDS}" temporal redis postgres; then
    err "Docker Compose could not stop every service."
    failed=1
  fi

  STACK_ROOT="${STACK_ROOT:-${ROOT_DIR}}"
  STACK_LIFECYCLE="stopped"
  save_state

  if [[ "${failed}" == "0" ]]; then
    ok "Local stack is stopped. Logs remain available with `make logs FOLLOW=0`."
    return 0
  fi
  err "The managed shutdown was incomplete; `make status` shows what remains."
  return 1
}

main() {
  local command="${1:-}"
  case "${command}" in
    start|status|logs|stop|restart) ;;
    -h|--help|help|"") usage; [[ -n "${command}" ]] || return 2; return 0 ;;
    *) err "Unknown local-stack command: ${command}"; usage; return 2 ;;
  esac

  load_backend_env
  temporal_target
  api_target

  case "${command}" in
    status) status_stack ;;
    logs) logs_stack ;;
    start)
      acquire_lock || return 1
      trap release_lock EXIT
      trap 'release_lock; exit 130' INT TERM
      start_stack
      ;;
    stop)
      acquire_lock || return 1
      trap release_lock EXIT
      trap 'release_lock; exit 130' INT TERM
      stop_stack
      ;;
    restart)
      acquire_lock || return 1
      trap release_lock EXIT
      trap 'release_lock; exit 130' INT TERM
      stop_stack || return 1
      start_stack
      ;;
  esac
}

main "$@"
