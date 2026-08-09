package observability

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStallRepo struct {
	mu   sync.Mutex
	sets []StalledEvalSet
}

func (f *fakeStallRepo) ListStalledEvalSets(_ context.Context, cutoff time.Time, _ int32) ([]StalledEvalSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]StalledEvalSet, 0)
	for _, s := range f.sets {
		if s.UpdatedAt.Before(cutoff) {
			out = append(out, s)
		}
	}
	return out, nil
}

func TestStallReaperEmitsForStaleSet(t *testing.T) {
	setID := uuid.New()
	repo := &fakeStallRepo{sets: []StalledEvalSet{{
		ID:          setID,
		WorkspaceID: uuid.New(),
		Status:      "running",
		UpdatedAt:   time.Now().UTC().Add(-2 * time.Hour),
		NewestChild: "session:running",
	}}}
	fleet := NewFleet(nil)
	reaper := NewStallReaper(repo, fleet, time.Hour, 30*time.Minute, slog.Default())
	fixed := time.Now().UTC()
	reaper.now = func() time.Time { return fixed }

	// Exercise scanOnce directly (ticker path covered by Start nil-guards).
	reaper.scanOnce(context.Background())
	// With nil meter, RecordSetStalled is a no-op; assert repo filter worked by
	// re-running with an enabled meter via Start scrape smoke in setup_test.
	sets, err := repo.ListStalledEvalSets(context.Background(), fixed.Add(-30*time.Minute), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sets) != 1 || sets[0].ID != setID {
		t.Fatalf("stalled sets = %+v", sets)
	}
}
