-- +goose Up
-- Named baselines are immutable-by-name scorecards that `agentclash eval`
-- compares future runs against. Workspace-scoped; uniqueness is by
-- (workspace_id, name) so "main" means different things in different
-- workspaces. scorecard_snapshot holds a point-in-time JSON copy so that
-- deleting or re-running the underlying run does not invalidate the
-- comparison.
CREATE TABLE baselines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    name text NOT NULL,
    pack_version_id uuid NOT NULL REFERENCES challenge_pack_versions (id) ON DELETE RESTRICT,
    run_id uuid NOT NULL REFERENCES runs (id) ON DELETE RESTRICT,
    scorecard_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT baselines_name_not_empty CHECK (length(btrim(name)) > 0),
    UNIQUE (workspace_id, name)
);

CREATE INDEX baselines_workspace_id_idx ON baselines (workspace_id);
CREATE INDEX baselines_run_id_idx ON baselines (run_id);

CREATE TRIGGER baselines_set_updated_at
BEFORE UPDATE ON baselines
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS baselines_set_updated_at ON baselines;
DROP INDEX IF EXISTS baselines_run_id_idx;
DROP INDEX IF EXISTS baselines_workspace_id_idx;
DROP TABLE IF EXISTS baselines;
