package worker

import (
	"context"
	"time"

	"github.com/agentclash/agentclash/backend/internal/observability"
	"github.com/agentclash/agentclash/backend/internal/repository"
)

// StallEvalSetRepo adapts repository.Repository to observability.StallRepository.
type StallEvalSetRepo struct {
	Repo *repository.Repository
}

func (s StallEvalSetRepo) ListStalledEvalSets(ctx context.Context, cutoff time.Time, limit int32) ([]observability.StalledEvalSet, error) {
	rows, err := s.Repo.ListStalledEvalSets(ctx, cutoff, limit)
	if err != nil {
		return nil, err
	}
	out := make([]observability.StalledEvalSet, 0, len(rows))
	for _, row := range rows {
		out = append(out, observability.StalledEvalSet{
			ID:          row.ID,
			WorkspaceID: row.WorkspaceID,
			Status:      row.Status,
			UpdatedAt:   row.UpdatedAt,
			NewestChild: row.NewestChildState,
		})
	}
	return out, nil
}
