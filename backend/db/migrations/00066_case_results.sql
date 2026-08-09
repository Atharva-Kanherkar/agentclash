-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE case_results (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    eval_set_id uuid REFERENCES eval_sets (id) ON DELETE CASCADE,
    eval_session_id uuid REFERENCES eval_sessions (id) ON DELETE SET NULL,
    run_id uuid NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    run_agent_id uuid NOT NULL REFERENCES run_agents (id) ON DELETE CASCADE,
    matrix_key text NOT NULL DEFAULT '',
    pack_ref text NOT NULL DEFAULT '',
    case_key text NOT NULL DEFAULT '',
    agent_deployment_id uuid,
    model text NOT NULL DEFAULT '',
    score double precision,
    correctness boolean,
    verdict text NOT NULL DEFAULT '',
    cost_usd double precision,
    duration_ms bigint,
    failure_class text NOT NULL DEFAULT '',
    transcript_artifact_ref text NOT NULL DEFAULT '',
    transcript_text text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (run_agent_id, case_key)
);

CREATE INDEX case_results_eval_set_idx ON case_results (eval_set_id, matrix_key);
CREATE INDEX case_results_workspace_set_idx ON case_results (workspace_id, eval_set_id);
CREATE INDEX case_results_score_idx ON case_results (eval_set_id, score);
CREATE INDEX case_results_transcript_trgm_idx ON case_results USING gin (transcript_text gin_trgm_ops);

-- +goose Down
DROP TABLE IF EXISTS case_results;
-- leave pg_trgm installed (shared extension)
