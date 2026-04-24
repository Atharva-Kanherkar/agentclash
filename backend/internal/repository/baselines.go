package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	repositorysqlc "github.com/agentclash/agentclash/backend/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Baseline is the domain type for a named, workspace-scoped scorecard
// snapshot that `agentclash eval` compares future runs against.
type Baseline struct {
	ID                uuid.UUID
	WorkspaceID       uuid.UUID
	Name              string
	PackVersionID     uuid.UUID
	RunID             uuid.UUID
	ScorecardSnapshot json.RawMessage
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// UpsertBaselineParams is the repo-layer input for creating or overwriting
// a named baseline.
type UpsertBaselineParams struct {
	WorkspaceID       uuid.UUID
	Name              string
	PackVersionID     uuid.UUID
	RunID             uuid.UUID
	ScorecardSnapshot json.RawMessage
}

func (r *Repository) UpsertBaseline(ctx context.Context, params UpsertBaselineParams) (Baseline, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return Baseline{}, ErrBaselineNameRequired
	}

	row, err := r.queries.UpsertBaseline(ctx, repositorysqlc.UpsertBaselineParams{
		WorkspaceID:       params.WorkspaceID,
		Name:              name,
		PackVersionID:     params.PackVersionID,
		RunID:             params.RunID,
		ScorecardSnapshot: normalizeJSON(params.ScorecardSnapshot),
	})
	if err != nil {
		return Baseline{}, fmt.Errorf("upsert baseline: %w", err)
	}
	return baselineFromRow(row), nil
}

func (r *Repository) GetBaselineByName(ctx context.Context, workspaceID uuid.UUID, name string) (Baseline, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Baseline{}, ErrBaselineNameRequired
	}
	row, err := r.queries.GetBaselineByName(ctx, repositorysqlc.GetBaselineByNameParams{
		WorkspaceID: workspaceID,
		Name:        name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Baseline{}, ErrBaselineNotFound
		}
		return Baseline{}, fmt.Errorf("get baseline: %w", err)
	}
	return baselineFromRow(row), nil
}

func (r *Repository) ListBaselinesByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]Baseline, error) {
	rows, err := r.queries.ListBaselinesByWorkspace(ctx, repositorysqlc.ListBaselinesByWorkspaceParams{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("list baselines: %w", err)
	}
	out := make([]Baseline, len(rows))
	for i, row := range rows {
		out[i] = baselineFromRow(row)
	}
	return out, nil
}

func (r *Repository) DeleteBaselineByName(ctx context.Context, workspaceID uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrBaselineNameRequired
	}
	rows, err := r.queries.DeleteBaselineByName(ctx, repositorysqlc.DeleteBaselineByNameParams{
		WorkspaceID: workspaceID,
		Name:        name,
	})
	if err != nil {
		return fmt.Errorf("delete baseline: %w", err)
	}
	if rows == 0 {
		return ErrBaselineNotFound
	}
	return nil
}

func baselineFromRow(row repositorysqlc.Baseline) Baseline {
	snap := json.RawMessage(row.ScorecardSnapshot)
	if len(snap) == 0 {
		snap = json.RawMessage(`{}`)
	}
	return Baseline{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Name:              row.Name,
		PackVersionID:     row.PackVersionID,
		RunID:             row.RunID,
		ScorecardSnapshot: snap,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}
