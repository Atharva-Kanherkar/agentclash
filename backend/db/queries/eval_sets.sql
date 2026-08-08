-- name: InsertEvalSet :one
INSERT INTO eval_sets (
    workspace_id,
    organization_id,
    name,
    status,
    manifest,
    expansion,
    max_concurrent_runs,
    budget_usd,
    case_fanout,
    combination_count,
    created_by_user_id
) VALUES (
    @workspace_id,
    @organization_id,
    @name,
    @status,
    @manifest,
    @expansion,
    @max_concurrent_runs,
    sqlc.narg('budget_usd'),
    @case_fanout,
    @combination_count,
    sqlc.narg('created_by_user_id')
)
RETURNING *;

-- name: GetEvalSetByID :one
SELECT * FROM eval_sets WHERE id = @id;

-- name: ListEvalSetsByWorkspaceID :many
SELECT * FROM eval_sets
WHERE workspace_id = @workspace_id
ORDER BY created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: CountEvalSetsByWorkspaceID :one
SELECT count(*)::bigint FROM eval_sets WHERE workspace_id = @workspace_id;

-- name: UpdateEvalSetStatus :one
UPDATE eval_sets
SET status = @to_status,
    updated_at = now(),
    started_at = CASE
        WHEN @to_status = 'expanding' AND started_at IS NULL THEN now()
        ELSE started_at
    END,
    finished_at = CASE
        WHEN @to_status IN ('completed', 'failed', 'cancelled') THEN now()
        ELSE finished_at
    END,
    failure_reason = sqlc.narg('failure_reason')
WHERE id = @id AND status = @from_status
RETURNING *;

-- name: AttachEvalSessionToEvalSet :exec
INSERT INTO eval_set_sessions (eval_set_id, eval_session_id, pack_ref)
VALUES (@eval_set_id, @eval_session_id, @pack_ref);

-- name: ListEvalSessionsByEvalSetID :many
SELECT eval_session_id, pack_ref
FROM eval_set_sessions
WHERE eval_set_id = @eval_set_id
ORDER BY pack_ref ASC;

-- name: UpsertEvalSetResult :one
INSERT INTO eval_set_results (
    eval_set_id, aggregate, evidence, session_count, run_count
) VALUES (
    @eval_set_id, @aggregate, @evidence, @session_count, @run_count
)
ON CONFLICT (eval_set_id) DO UPDATE SET
    aggregate = EXCLUDED.aggregate,
    evidence = EXCLUDED.evidence,
    session_count = EXCLUDED.session_count,
    run_count = EXCLUDED.run_count,
    updated_at = now()
RETURNING *;

-- name: GetEvalSetResultByEvalSetID :one
SELECT * FROM eval_set_results WHERE eval_set_id = @eval_set_id;
