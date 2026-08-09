-- name: UpsertCaseResult :one
INSERT INTO case_results (
    workspace_id,
    organization_id,
    eval_set_id,
    eval_session_id,
    run_id,
    run_agent_id,
    matrix_key,
    pack_ref,
    case_key,
    agent_deployment_id,
    model,
    score,
    correctness,
    verdict,
    cost_usd,
    duration_ms,
    failure_class,
    transcript_artifact_ref,
    transcript_text
) VALUES (
    @workspace_id,
    @organization_id,
    sqlc.narg('eval_set_id'),
    sqlc.narg('eval_session_id'),
    @run_id,
    @run_agent_id,
    @matrix_key,
    @pack_ref,
    @case_key,
    sqlc.narg('agent_deployment_id'),
    @model,
    sqlc.narg('score'),
    sqlc.narg('correctness'),
    @verdict,
    sqlc.narg('cost_usd'),
    sqlc.narg('duration_ms'),
    @failure_class,
    @transcript_artifact_ref,
    @transcript_text
)
ON CONFLICT (run_agent_id, case_key) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    organization_id = EXCLUDED.organization_id,
    eval_set_id = EXCLUDED.eval_set_id,
    eval_session_id = EXCLUDED.eval_session_id,
    run_id = EXCLUDED.run_id,
    matrix_key = EXCLUDED.matrix_key,
    pack_ref = EXCLUDED.pack_ref,
    agent_deployment_id = EXCLUDED.agent_deployment_id,
    model = EXCLUDED.model,
    score = EXCLUDED.score,
    correctness = EXCLUDED.correctness,
    verdict = EXCLUDED.verdict,
    cost_usd = EXCLUDED.cost_usd,
    duration_ms = EXCLUDED.duration_ms,
    failure_class = EXCLUDED.failure_class,
    transcript_artifact_ref = EXCLUDED.transcript_artifact_ref,
    transcript_text = EXCLUDED.transcript_text,
    updated_at = now()
RETURNING *;

-- name: ListCaseResultsByEvalSetID :many
SELECT *
FROM case_results
WHERE eval_set_id = @eval_set_id
  AND workspace_id = @workspace_id
  AND (sqlc.narg('agent_deployment_id')::uuid IS NULL OR agent_deployment_id = sqlc.narg('agent_deployment_id'))
  AND (sqlc.narg('pack_ref')::text IS NULL OR pack_ref = sqlc.narg('pack_ref'))
  AND (sqlc.narg('verdict')::text IS NULL OR verdict = sqlc.narg('verdict'))
  AND (sqlc.narg('min_score')::float8 IS NULL OR score >= sqlc.narg('min_score'))
  AND (sqlc.narg('max_score')::float8 IS NULL OR score <= sqlc.narg('max_score'))
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR id > sqlc.narg('cursor_id'))
ORDER BY id ASC
LIMIT @limit_count;

-- name: SearchCaseResultsByEvalSetID :many
SELECT
    id, workspace_id, organization_id, eval_set_id, eval_session_id, run_id, run_agent_id,
    matrix_key, pack_ref, case_key, agent_deployment_id, model, score, correctness, verdict,
    cost_usd, duration_ms, failure_class, transcript_artifact_ref, transcript_text,
    created_at, updated_at,
    similarity(transcript_text, @query) AS rank
FROM case_results
WHERE eval_set_id = @eval_set_id
  AND workspace_id = @workspace_id
  AND transcript_text % @query
  AND (sqlc.narg('agent_deployment_id')::uuid IS NULL OR agent_deployment_id = sqlc.narg('agent_deployment_id'))
  AND (sqlc.narg('pack_ref')::text IS NULL OR pack_ref = sqlc.narg('pack_ref'))
  AND (sqlc.narg('verdict')::text IS NULL OR verdict = sqlc.narg('verdict'))
  AND (sqlc.narg('min_score')::float8 IS NULL OR score >= sqlc.narg('min_score'))
  AND (sqlc.narg('max_score')::float8 IS NULL OR score <= sqlc.narg('max_score'))
ORDER BY rank DESC, id ASC
LIMIT @limit_count;

-- name: ListCaseResultsForExport :many
SELECT *
FROM case_results
WHERE eval_set_id = @eval_set_id
  AND workspace_id = @workspace_id
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR id > sqlc.narg('cursor_id'))
ORDER BY id ASC
LIMIT @limit_count;

-- name: ListCaseResultsByEvalSetForCompare :many
SELECT matrix_key, pack_ref, case_key, agent_deployment_id, model, score, verdict, run_id
FROM case_results
WHERE eval_set_id = @eval_set_id
  AND workspace_id = @workspace_id
ORDER BY matrix_key ASC, case_key ASC;

-- name: AggregateCaseResultsByEvalSet :many
SELECT
    pack_ref,
    coalesce(agent_deployment_id::text, '') AS agent_deployment_id,
    model,
    count(*)::bigint AS n,
    coalesce(avg(score), 0)::float8 AS mean_score,
    coalesce(percentile_cont(0.5) WITHIN GROUP (ORDER BY score), 0)::float8 AS p50_score,
    coalesce(percentile_cont(0.95) WITHIN GROUP (ORDER BY score), 0)::float8 AS p95_score,
    coalesce(stddev_samp(score), 0)::float8 AS score_stddev,
    count(*) FILTER (WHERE verdict = 'pass' OR correctness IS TRUE)::bigint AS wins,
    count(*) FILTER (WHERE verdict = 'fail' OR correctness IS FALSE)::bigint AS losses
FROM case_results
WHERE eval_set_id = @eval_set_id
  AND workspace_id = @workspace_id
GROUP BY pack_ref, agent_deployment_id, model
ORDER BY pack_ref, agent_deployment_id, model;
