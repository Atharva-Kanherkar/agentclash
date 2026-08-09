package repository

import (
	"context"
	"fmt"

	repositorysqlc "github.com/agentclash/agentclash/backend/internal/repository/sqlc"
	"github.com/agentclash/agentclash/backend/internal/storage"
	"github.com/google/uuid"
)

// PurgeRunEventPayloadObjects deletes offloaded run-event payload objects for a
// run, then removes the corresponding artifact rows. Call before hard-deleting
// a run so CASCADE on run_events does not leave orphaned objects.
func (r *Repository) PurgeRunEventPayloadObjects(ctx context.Context, store storage.Store, runID uuid.UUID) (int, error) {
	if store == nil {
		return 0, fmt.Errorf("storage store is required")
	}
	rows, err := r.queries.ListRunEventPayloadArtifactsByRunID(ctx, repositorysqlc.ListRunEventPayloadArtifactsByRunIDParams{
		RunID: &runID,
	})
	if err != nil {
		return 0, fmt.Errorf("list run event payload artifacts: %w", err)
	}
	deleted := 0
	for _, row := range rows {
		if err := store.DeleteObject(ctx, row.StorageKey); err != nil {
			return deleted, fmt.Errorf("delete offloaded payload %q: %w", row.StorageKey, err)
		}
		if _, err := r.db.Exec(ctx, `DELETE FROM artifacts WHERE id = $1`, row.ID); err != nil {
			return deleted, fmt.Errorf("delete artifact row %s: %w", row.ID, err)
		}
		deleted++
	}
	return deleted, nil
}

// HardDeleteRun removes a run after purging offloaded event payloads. Intended
// for retention/reaper paths and tests; production product deletes are rare.
func (r *Repository) HardDeleteRun(ctx context.Context, store storage.Store, runID uuid.UUID) error {
	if _, err := r.PurgeRunEventPayloadObjects(ctx, store, runID); err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM runs WHERE id = $1`, runID)
	if err != nil {
		return fmt.Errorf("delete run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRunNotFound
	}
	return nil
}
