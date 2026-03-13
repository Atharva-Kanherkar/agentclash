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
    overall_score,
    correctness_score,
    reliability_score,
    latency_score,
    cost_score,
    scorecard,
    created_at,
    updated_at
FROM run_agent_scorecards
WHERE run_agent_id = @run_agent_id
LIMIT 1;
