# codex/issue-358-eval-session-persistence — Test Contract

## Functional Behavior
- Add a durable `eval_sessions` persistence model that groups repeated runs under one parent record without changing legacy single-run behavior.
- Create sessions with `repetitions`, `aggregation_config`, `success_threshold_config`, `routing_task_snapshot`, `schema_version`, lifecycle timestamps, and a status drawn from the locked set: `queued`, `running`, `aggregating`, `completed`, `failed`, `cancelled`.
- Allow runs to optionally reference an eval session through a nullable `eval_session_id` foreign key while preserving existing rows where `eval_session_id` is `NULL`.
- Persist config snapshots as stored JSON values rather than reconstructing them from mutable state later.
- Enforce the explicit policy that `repetitions = 1` still creates a degenerate eval session consistently.
- Enforce legal session lifecycle transitions and reject illegal transitions with a typed error.
- Set `started_at` on first transition into `running`, set `finished_at` on transition into a terminal state, and update `updated_at` on every write.

## Unit Tests
- `TestEvalSessionStatus_AllowsLegalTransitions` — transition helper accepts the locked legal matrix.
- `TestEvalSessionStatus_RejectsIllegalTransitions` — illegal and terminal-state transitions return `IllegalSessionTransition`.
- `TestEvalSessionSnapshot_RoundTripsWithoutMutation` — snapshot values preserve nested JSON structure and schema version.

## Integration / Functional Tests
- Repository integration test creates an eval session, attaches child runs, and reads the session back with children in deterministic order.
- Repository integration test persists `queued -> running -> aggregating -> completed` and verifies `started_at`, `finished_at`, and `updated_at`.
- Repository integration test covers cancellation from `queued`, `running`, and `aggregating`.
- Repository integration test proves `repetitions = 1` still creates a durable single-run session.
- Migration-focused integration test verifies `eval_sessions` exists with expected columns, indexes, and constraints, and `runs.eval_session_id` is nullable with the intended foreign key behavior.
- Backward-compatibility integration test proves a legacy run with `NULL eval_session_id` still loads successfully.

## Smoke Tests
- `go test ./internal/repository/...` passes for repository and migration coverage touched by this change.
- `go test ./internal/domain/...` passes for any new lifecycle or snapshot domain helpers.
- SQL code generation remains in sync with the new migration and query files.

## E2E Tests
- N/A — this change is intentionally scoped to persistence and repository layers with no new user-facing flow.

## Manual / cURL Tests
- N/A unless implementation unexpectedly requires exercising an existing HTTP path. If that happens, start the local stack, capture the exact `curl` commands and responses, and include them in the PR description.
