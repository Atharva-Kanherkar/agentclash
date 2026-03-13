# Worker Local Development

Purpose: give the repo one short path for starting the first Temporal worker introduced in issue `#21`.

## What exists in this step

This worker slice is intentionally thin.

It currently provides:

- config bootstrapping from environment
- Postgres connection setup for workflow activities
- Temporal client connection and worker startup
- registration of `RunWorkflow`, `RunAgentWorkflow`, and the current fake activity set from `backend/internal/workflow`
- polling of the `RunWorkflow` task queue used by API-started runs
- graceful shutdown for local development
- one injection point for later hosted/native execution hooks without changing the worker bootstrap shape

It does not yet provide:

- hosted external execution
- provider adapters
- native execution
- sandbox integration
- replay or scorecard generation

## Local startup

From the repository root:

```bash
cp backend/.env.example backend/.env
cd backend
set -a
source .env
set +a
go run ./cmd/worker
```

Or from the repository root with the make target:

```bash
make worker
```

The worker expects:

- Postgres reachable through `DATABASE_URL`
- Temporal reachable through `TEMPORAL_HOST_PORT` and `TEMPORAL_NAMESPACE`

For local development, the intended setup is:

1. `make db-up`
2. `make db-migrate`
3. start a Temporal local dev server or point the env vars at a dev namespace
4. run `make worker`
5. separately run `make api-server`

Once both processes are running, `POST /v1/runs` can create a queued run and the worker will execute the current fake workflow path end to end.

## Worker env vars

- `DATABASE_URL`: Postgres connection string. Default `postgres://agentclash:agentclash@localhost:5432/agentclash?sslmode=disable`
- `TEMPORAL_HOST_PORT`: Temporal target. Default `localhost:7233`
- `TEMPORAL_NAMESPACE`: Temporal namespace. Default `default`
- `WORKER_IDENTITY`: Temporal worker identity string. Default `agentclash-worker@<hostname>`
- `WORKER_SHUTDOWN_TIMEOUT`: graceful shutdown timeout duration. Default `10s`
