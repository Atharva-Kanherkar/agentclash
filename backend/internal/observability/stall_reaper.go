package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// StalledEvalSet is a non-terminal set past the stall threshold.
type StalledEvalSet struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Status      string
	UpdatedAt   time.Time
	NewestChild string
}

// StallRepository loads candidates for stuck-set detection.
type StallRepository interface {
	ListStalledEvalSets(ctx context.Context, cutoff time.Time, limit int32) ([]StalledEvalSet, error)
}

// StallReaper flags stalled eval sets (metric + log). Never auto-kills.
type StallReaper struct {
	repo      StallRepository
	fleet     *Fleet
	interval  time.Duration
	threshold time.Duration
	logger    *slog.Logger
	now       func() time.Time
}

func NewStallReaper(repo StallRepository, fleet *Fleet, interval, threshold time.Duration, logger *slog.Logger) *StallReaper {
	if logger == nil {
		logger = slog.Default()
	}
	if fleet == nil {
		fleet = NewFleet(nil)
	}
	return &StallReaper{
		repo:      repo,
		fleet:     fleet,
		interval:  interval,
		threshold: threshold,
		logger:    logger,
		now:       time.Now,
	}
}

// Start implements worker.OrphanRunReaper.
func (r *StallReaper) Start(ctx context.Context) {
	if r == nil || r.repo == nil || r.interval <= 0 || r.threshold <= 0 {
		return
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.scanOnce(ctx)
		}
	}
}

func (r *StallReaper) scanOnce(ctx context.Context) {
	cutoff := r.now().UTC().Add(-r.threshold)
	sets, err := r.repo.ListStalledEvalSets(ctx, cutoff, 200)
	if err != nil {
		r.logger.Error("fleet stall reaper failed", "error", err)
		return
	}
	for _, set := range sets {
		r.fleet.RecordSetStalled(ctx)
		r.logger.Warn("fleet.set.stalled",
			"eval_set_id", set.ID.String(),
			"workspace_id", set.WorkspaceID.String(),
			"status", set.Status,
			"updated_at", set.UpdatedAt.UTC().Format(time.RFC3339),
			"newest_child_state", set.NewestChild,
			"stall_threshold", r.threshold.String(),
		)
	}
}
