-- name: GetRunAgentReplayByRunAgentID :one
SELECT
    id,
    run_agent_id,
    artifact_id,
    summary,
    latest_sequence_number,
    event_count,
    created_at,
    updated_at
FROM run_agent_replays
WHERE run_agent_id = @run_agent_id
LIMIT 1;

-- name: GetRunAgentScorecardByRunAgentID :one
SELECT
    id,
    run_agent_id,
    evaluation_spec_id,
    CASE WHEN overall_score IS NULL THEN NULL ELSE overall_score::double precision END AS overall_score,
    CASE WHEN correctness_score IS NULL THEN NULL ELSE correctness_score::double precision END AS correctness_score,
    CASE WHEN reliability_score IS NULL THEN NULL ELSE reliability_score::double precision END AS reliability_score,
    CASE WHEN latency_score IS NULL THEN NULL ELSE latency_score::double precision END AS latency_score,
    CASE WHEN cost_score IS NULL THEN NULL ELSE cost_score::double precision END AS cost_score,
    scorecard,
    created_at,
    updated_at
FROM run_agent_scorecards
WHERE run_agent_id = @run_agent_id
LIMIT 1;
