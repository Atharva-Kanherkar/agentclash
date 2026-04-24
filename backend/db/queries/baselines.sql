-- name: UpsertBaseline :one
-- Save or replace a named baseline within a workspace. Same-name re-publish
-- overwrites — baselines are moving bookmarks by design (users retrain
-- "main" when their challenge pack evolves).
INSERT INTO baselines (
    workspace_id,
    name,
    pack_version_id,
    run_id,
    scorecard_snapshot
) VALUES (
    @workspace_id,
    @name,
    @pack_version_id,
    @run_id,
    @scorecard_snapshot
)
ON CONFLICT (workspace_id, name) DO UPDATE SET
    pack_version_id = EXCLUDED.pack_version_id,
    run_id = EXCLUDED.run_id,
    scorecard_snapshot = EXCLUDED.scorecard_snapshot,
    updated_at = now()
RETURNING *;

-- name: GetBaselineByName :one
SELECT *
FROM baselines
WHERE workspace_id = @workspace_id AND name = @name
LIMIT 1;

-- name: ListBaselinesByWorkspace :many
SELECT *
FROM baselines
WHERE workspace_id = @workspace_id
ORDER BY updated_at DESC, name ASC;

-- name: DeleteBaselineByName :execrows
DELETE FROM baselines
WHERE workspace_id = @workspace_id AND name = @name;
