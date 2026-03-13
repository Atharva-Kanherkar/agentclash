-- +goose Up
CREATE TABLE hosted_run_executions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id uuid NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    run_agent_id uuid NOT NULL UNIQUE REFERENCES run_agents (id) ON DELETE CASCADE,
    endpoint_url text NOT NULL,
    trace_level text NOT NULL CHECK (trace_level IN ('black_box', 'structured_trace')),
    status text NOT NULL CHECK (status IN ('starting', 'accepted', 'running', 'completed', 'failed', 'timed_out')),
    external_run_id text,
    accepted_response jsonb NOT NULL DEFAULT '{}'::jsonb,
    last_event_type text,
    last_event_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    result_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_message text,
    deadline_at timestamptz NOT NULL,
    accepted_at timestamptz,
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (run_agent_id, run_id) REFERENCES run_agents (id, run_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX hosted_run_executions_external_run_id_uq
ON hosted_run_executions (external_run_id)
WHERE external_run_id IS NOT NULL;

CREATE TRIGGER hosted_run_executions_set_updated_at
BEFORE UPDATE ON hosted_run_executions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS hosted_run_executions_set_updated_at ON hosted_run_executions;
DROP TABLE IF EXISTS hosted_run_executions;
