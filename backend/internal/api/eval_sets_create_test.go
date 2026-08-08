package api

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/agentclash/agentclash/runtime/evalset"
	"github.com/google/uuid"
)

type fakeEvalSetStore struct {
	sets       map[uuid.UUID]repository.EvalSet
	sessions   map[uuid.UUID][]uuid.UUID
	packs      map[uuid.UUID][]string
	results    map[uuid.UUID]repository.EvalSetResult
	orgID      uuid.UUID
	createErr  error
	transition []string
}

func newFakeEvalSetStore() *fakeEvalSetStore {
	return &fakeEvalSetStore{
		sets:     map[uuid.UUID]repository.EvalSet{},
		sessions: map[uuid.UUID][]uuid.UUID{},
		packs:    map[uuid.UUID][]string{},
		results:  map[uuid.UUID]repository.EvalSetResult{},
		orgID:    uuid.New(),
	}
}

func (f *fakeEvalSetStore) CreateEvalSet(_ context.Context, params repository.CreateEvalSetParams) (repository.EvalSet, error) {
	if f.createErr != nil {
		return repository.EvalSet{}, f.createErr
	}
	set := repository.EvalSet{
		ID:                uuid.New(),
		WorkspaceID:       params.WorkspaceID,
		OrganizationID:    params.OrganizationID,
		Name:              params.Name,
		Status:            domain.EvalSetStatusQueued,
		Manifest:          params.Manifest,
		Expansion:         params.Expansion,
		MaxConcurrentRuns: params.MaxConcurrentRuns,
		BudgetUSD:         params.BudgetUSD,
		CaseFanout:        params.CaseFanout,
		CombinationCount:  params.CombinationCount,
		CreatedByUserID:   params.CreatedByUserID,
	}
	f.sets[set.ID] = set
	return set, nil
}

func (f *fakeEvalSetStore) GetEvalSetByID(_ context.Context, id uuid.UUID) (repository.EvalSet, error) {
	set, ok := f.sets[id]
	if !ok {
		return repository.EvalSet{}, repository.ErrEvalSetNotFound
	}
	return set, nil
}

func (f *fakeEvalSetStore) ListEvalSetsByWorkspaceID(_ context.Context, workspaceID uuid.UUID, _, _ int32) ([]repository.EvalSet, int64, error) {
	out := make([]repository.EvalSet, 0)
	for _, set := range f.sets {
		if set.WorkspaceID == workspaceID {
			out = append(out, set)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeEvalSetStore) AttachEvalSessionToEvalSet(_ context.Context, evalSetID, evalSessionID uuid.UUID, packRef string) error {
	f.sessions[evalSetID] = append(f.sessions[evalSetID], evalSessionID)
	f.packs[evalSetID] = append(f.packs[evalSetID], packRef)
	return nil
}

func (f *fakeEvalSetStore) ListEvalSessionsByEvalSetID(_ context.Context, evalSetID uuid.UUID) ([]uuid.UUID, []string, error) {
	return f.sessions[evalSetID], f.packs[evalSetID], nil
}

func (f *fakeEvalSetStore) TransitionEvalSetStatus(_ context.Context, id uuid.UUID, from, to domain.EvalSetStatus, failureReason *string) (repository.EvalSet, error) {
	set, ok := f.sets[id]
	if !ok {
		return repository.EvalSet{}, repository.ErrEvalSetNotFound
	}
	if set.Status != from || !from.CanTransitionTo(to) {
		return repository.EvalSet{}, repository.ErrInvalidTransition
	}
	set.Status = to
	set.FailureReason = failureReason
	f.sets[id] = set
	f.transition = append(f.transition, string(from)+"→"+string(to))
	return set, nil
}

func (f *fakeEvalSetStore) GetOrganizationIDByWorkspaceID(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return f.orgID, nil
}

func (f *fakeEvalSetStore) GetEvalSetResultByEvalSetID(_ context.Context, evalSetID uuid.UUID) (repository.EvalSetResult, error) {
	result, ok := f.results[evalSetID]
	if !ok {
		return repository.EvalSetResult{}, repository.ErrEvalSetNotFound
	}
	return result, nil
}

type fakeEvalSessionCreator struct {
	calls []CreateEvalSessionInput
	err   error
}

func (f *fakeEvalSessionCreator) CreateEvalSession(_ context.Context, _ Caller, input CreateEvalSessionInput) (CreateEvalSessionResult, error) {
	f.calls = append(f.calls, input)
	if f.err != nil {
		return CreateEvalSessionResult{}, f.err
	}
	runIDs := make([]uuid.UUID, 0, len(input.EvalSession.RunMatrix))
	for range input.EvalSession.RunMatrix {
		runIDs = append(runIDs, uuid.New())
	}
	return CreateEvalSessionResult{
		Session: domain.EvalSession{ID: uuid.New(), Status: domain.EvalSessionStatusQueued},
		RunIDs:  runIDs,
	}, nil
}

type fakeEvalSetStarter struct {
	started  []uuid.UUID
	canceled []uuid.UUID
}

func (f *fakeEvalSetStarter) StartEvalSetWorkflow(_ context.Context, evalSetID uuid.UUID) error {
	f.started = append(f.started, evalSetID)
	return nil
}

func (f *fakeEvalSetStarter) CancelEvalSetWorkflow(_ context.Context, evalSetID uuid.UUID) error {
	f.canceled = append(f.canceled, evalSetID)
	return nil
}

func TestEvalSetManagerCreate_GroupsSessionsByPack(t *testing.T) {
	packA := uuid.New()
	packB := uuid.New()
	agent1 := uuid.New()
	agent2 := uuid.New()
	agent3 := uuid.New()
	workspaceID := uuid.New()

	store := newFakeEvalSetStore()
	sessions := &fakeEvalSessionCreator{}
	starter := &fakeEvalSetStarter{}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(store, sessions, starter)

	manifest := map[string]any{
		"schema": evalset.SchemaV1,
		"name":   "nightly",
		"packs":  []string{packA.String(), packB.String()},
		"agents": []map[string]string{
			{"deployment": agent1.String()},
			{"deployment": agent2.String()},
			{"deployment": agent3.String()},
		},
		"repeats": 5,
		"limits":  map[string]any{"max_concurrent_runs": 4},
	}
	raw, _ := json.Marshal(manifest)

	set, sessionIDs, err := manager.Create(context.Background(), Caller{UserID: uuid.New()}, workspaceID, raw, 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if set.CombinationCount != 30 {
		t.Fatalf("combination_count = %d, want 30", set.CombinationCount)
	}
	if len(sessionIDs) != 2 {
		t.Fatalf("session count = %d, want 2", len(sessionIDs))
	}
	if len(sessions.calls) != 2 {
		t.Fatalf("CreateEvalSession calls = %d, want 2", len(sessions.calls))
	}
	totalRuns := 0
	for _, call := range sessions.calls {
		if !call.SkipWorkflowStart {
			t.Fatal("expected SkipWorkflowStart=true so EvalSetWorkflow owns child launch")
		}
		if len(call.EvalSession.RunMatrix) != 15 {
			t.Fatalf("matrix len = %d, want 15", len(call.EvalSession.RunMatrix))
		}
		totalRuns += len(call.EvalSession.RunMatrix)
		if call.EvalSession.Aggregation.Method != "median" {
			t.Fatalf("aggregation method = %q", call.EvalSession.Aggregation.Method)
		}
	}
	if totalRuns != 30 {
		t.Fatalf("total runs = %d, want 30", totalRuns)
	}
	if len(starter.started) != 1 || starter.started[0] != set.ID {
		t.Fatalf("starter = %v, want [%s]", starter.started, set.ID)
	}
}

func TestEvalSetManagerCreate_RejectsNonUUIDRefs(t *testing.T) {
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(newFakeEvalSetStore(), &fakeEvalSessionCreator{}, &fakeEvalSetStarter{})
	manifest := map[string]any{
		"schema":  evalset.SchemaV1,
		"name":    "bad",
		"packs":   []string{"catalog/code-review"},
		"agents":  []map[string]string{{"deployment": "claude-default"}},
		"repeats": 1,
	}
	raw, _ := json.Marshal(manifest)
	_, _, err := manager.Create(context.Background(), Caller{UserID: uuid.New()}, uuid.New(), raw, 0)
	if err == nil {
		t.Fatal("expected error for non-UUID pack/agent refs")
	}
}

func TestCancelEvalSet_CancelsWorkflowAndSettles(t *testing.T) {
	store := newFakeEvalSetStore()
	starter := &fakeEvalSetStarter{}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(store, &fakeEvalSessionCreator{}, starter)

	id := uuid.New()
	store.sets[id] = repository.EvalSet{
		ID:          id,
		WorkspaceID: uuid.New(),
		Status:      domain.EvalSetStatusRunning,
	}
	set, err := manager.Cancel(context.Background(), Caller{UserID: uuid.New()}, id)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if set.Status != domain.EvalSetStatusCancelled {
		t.Fatalf("status = %s, want cancelled", set.Status)
	}
	if len(starter.canceled) != 1 || starter.canceled[0] != id {
		t.Fatalf("canceled workflows = %v", starter.canceled)
	}
}

func TestCancelEvalSet_IdempotentTerminal(t *testing.T) {
	store := newFakeEvalSetStore()
	starter := &fakeEvalSetStarter{}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(store, &fakeEvalSessionCreator{}, starter)
	id := uuid.New()
	store.sets[id] = repository.EvalSet{ID: id, WorkspaceID: uuid.New(), Status: domain.EvalSetStatusCompleted}
	set, err := manager.Cancel(context.Background(), Caller{UserID: uuid.New()}, id)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if set.Status != domain.EvalSetStatusCompleted {
		t.Fatalf("status = %s", set.Status)
	}
	if len(starter.canceled) != 0 {
		t.Fatalf("should not cancel terminal sets")
	}
}

func TestEvalSetManagerCreate_RequiresPersistence(t *testing.T) {
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{})
	_, _, err := manager.Create(context.Background(), Caller{UserID: uuid.New()}, uuid.New(), []byte(`{}`), 0)
	if err == nil {
		t.Fatal("expected not configured error")
	}
	if !errors.Is(err, err) && err.Error() == "" {
		t.Fatal("empty error")
	}
}
