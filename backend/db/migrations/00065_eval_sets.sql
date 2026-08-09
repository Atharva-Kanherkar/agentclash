-- +goose Up
CREATE TABLE eval_sets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'expanding', 'running', 'aggregating', 'completed', 'failed', 'cancelled')),
    manifest jsonb NOT NULL,
    expansion jsonb NOT NULL DEFAULT '{}'::jsonb,
    max_concurrent_runs integer NOT NULL DEFAULT 0 CHECK (max_concurrent_runs >= 0),
    budget_usd numeric,
    case_fanout boolean NOT NULL DEFAULT false,
    combination_count integer NOT NULL DEFAULT 0 CHECK (combination_count >= 0),
    created_by_user_id uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    finished_at timestamptz,
    failure_reason text
);

CREATE INDEX eval_sets_workspace_created_idx ON eval_sets (workspace_id, created_at DESC);

CREATE TABLE eval_set_sessions (
    eval_set_id uuid NOT NULL REFERENCES eval_sets (id) ON DELETE CASCADE,
    eval_session_id uuid NOT NULL REFERENCES eval_sessions (id) ON DELETE CASCADE,
    pack_ref text NOT NULL,
    PRIMARY KEY (eval_set_id, eval_session_id)
);

CREATE INDEX eval_set_sessions_session_idx ON eval_set_sessions (eval_session_id);

CREATE TABLE eval_set_results (
    eval_set_id uuid PRIMARY KEY REFERENCES eval_sets (id) ON DELETE CASCADE,
    aggregate jsonb NOT NULL DEFAULT '{}'::jsonb,
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    session_count integer NOT NULL DEFAULT 0,
    run_count integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS eval_set_results;
DROP TABLE IF EXISTS eval_set_sessions;
DROP TABLE IF EXISTS eval_sets;
