package workflow

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
	sdkactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

type fakeEvalSetRepo struct {
	mu         sync.Mutex
	set        repository.EvalSet
	sessions   []uuid.UUID
	packs      []string
	runs       map[uuid.UUID][]domain.Run
	session    map[uuid.UUID]domain.EvalSession
	result     *repository.EvalSetResult
	statuses   []domain.EvalSetStatus
	caseCosts  float64
	activeRuns int
}

func (f *fakeEvalSetRepo) GetEvalSetByID(_ context.Context, id uuid.UUID) (repository.EvalSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.set.ID != id {
		return repository.EvalSet{}, repository.ErrEvalSetNotFound
	}
	return f.set, nil
}

func (f *fakeEvalSetRepo) TransitionEvalSetStatus(_ context.Context, id uuid.UUID, from, to domain.EvalSetStatus, failureReason *string) (repository.EvalSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.set.ID != id || f.set.Status != from || !from.CanTransitionTo(to) {
		return repository.EvalSet{}, repository.ErrInvalidTransition
	}
	f.set.Status = to
	f.set.FailureReason = failureReason
	f.statuses = append(f.statuses, to)
	return f.set, nil
}

func (f *fakeEvalSetRepo) ListEvalSessionsByEvalSetID(_ context.Context, _ uuid.UUID) ([]uuid.UUID, []string, error) {
	return append([]uuid.UUID(nil), f.sessions...), append([]string(nil), f.packs...), nil
}

func (f *fakeEvalSetRepo) UpsertEvalSetResult(_ context.Context, evalSetID uuid.UUID, aggregate, evidence json.RawMessage, sessionCount, runCount int32) (repository.EvalSetResult, error) {
	result := repository.EvalSetResult{
		EvalSetID:    evalSetID,
		Aggregate:    aggregate,
		Evidence:     evidence,
		SessionCount: sessionCount,
		RunCount:     runCount,
	}
	f.result = &result
	return result, nil
}

func (f *fakeEvalSetRepo) GetEvalSessionByID(_ context.Context, id uuid.UUID) (domain.EvalSession, error) {
	if s, ok := f.session[id]; ok {
		return s, nil
	}
	return domain.EvalSession{ID: id, Status: domain.EvalSessionStatusCompleted}, nil
}

func (f *fakeEvalSetRepo) ListRunsByEvalSessionID(_ context.Context, evalSessionID uuid.UUID) ([]domain.Run, error) {
	return f.runs[evalSessionID], nil
}

func (f *fakeEvalSetRepo) ListRunAgentsByRunID(_ context.Context, runID uuid.UUID) ([]domain.RunAgent, error) {
	return []domain.RunAgent{{ID: uuid.New(), RunID: runID, Status: domain.RunAgentStatusCompleted}}, nil
}

func (f *fakeEvalSetRepo) GetRunAgentScorecardByRunAgentID(_ context.Context, _ uuid.UUID) (repository.RunAgentScorecard, error) {
	return repository.RunAgentScorecard{}, repository.ErrRunAgentScorecardNotFound
}

func (f *fakeEvalSetRepo) UpsertCaseResult(_ context.Context, _ repository.UpsertCaseResultParams) (repository.CaseResult, error) {
	return repository.CaseResult{}, nil
}

func (f *fakeEvalSetRepo) UpdateEvalSetSpentUSD(_ context.Context, id uuid.UUID, spent float64) (repository.EvalSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.set.ID != id {
		return repository.EvalSet{}, repository.ErrEvalSetNotFound
	}
	f.set.SpentUSD = spent
	return f.set, nil
}

func (f *fakeEvalSetRepo) SumCaseResultCostByEvalSetID(_ context.Context, _ uuid.UUID) (float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.caseCosts, nil
}

func (f *fakeEvalSetRepo) CountActiveWorkspaceRuns(_ context.Context, _ uuid.UUID) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.activeRuns, nil
}

func newEvalSetWorkflowTestEnvironment(
	repo *fakeEvalSetRepo,
	childSession func(ctx sdkworkflow.Context, input EvalSessionWorkflowInput) error,
) *testsuite.TestWorkflowEnvironment {
	var suite testsuite.WorkflowTestSuite
	suite.SetDisableRegistrationAliasing(true)
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{
		ID:        "test-eval-set-workflow",
		TaskQueue: TaskQueueExecution,
	})

	activities := (&Activities{}).
		WithEvalSetRepository(repo).
		WithEvalSetBudgetRepository(repo).
		WithWorkspaceRunCounter(repo)
	env.RegisterWorkflowWithOptions(EvalSetWorkflow, sdkworkflow.RegisterOptions{Name: EvalSetWorkflowName})
	env.RegisterWorkflowWithOptions(ScanEvalSetWorkflow, sdkworkflow.RegisterOptions{Name: ScanEvalSetWorkflowName})
	env.RegisterActivityWithOptions(activities.TransitionEvalSetStatus, sdkactivity.RegisterOptions{Name: transitionEvalSetStatusActivityName})
	env.RegisterActivityWithOptions(activities.LoadEvalSet, sdkactivity.RegisterOptions{Name: loadEvalSetActivityName})
	env.RegisterActivityWithOptions(activities.ListEvalSetSessionIDs, sdkactivity.RegisterOptions{Name: listEvalSetSessionIDsActivityName})
	env.RegisterActivityWithOptions(activities.ListEvalSetManifestScanners, sdkactivity.RegisterOptions{Name: listEvalSetManifestScannersActivityName})
	env.RegisterActivityWithOptions(activities.AggregateEvalSet, sdkactivity.RegisterOptions{Name: aggregateEvalSetActivityName})
	env.RegisterActivityWithOptions(activities.CheckEvalSetBudget, sdkactivity.RegisterOptions{Name: checkEvalSetBudgetActivityName})
	env.RegisterActivityWithOptions(activities.RefreshEvalSetSpend, sdkactivity.RegisterOptions{Name: refreshEvalSetSpendActivityName})
	env.RegisterActivityWithOptions(activities.WaitWorkspaceRunCapacity, sdkactivity.RegisterOptions{Name: waitWorkspaceRunCapacityActivityName})
	env.RegisterActivityWithOptions(activities.RecordEvalSetSpendEvent, sdkactivity.RegisterOptions{Name: recordEvalSetSpendEventActivityName})

	if childSession == nil {
		childSession = func(ctx sdkworkflow.Context, input EvalSessionWorkflowInput) error {
			return nil
		}
	}
	env.RegisterWorkflowWithOptions(childSession, sdkworkflow.RegisterOptions{Name: EvalSessionWorkflowName})
	return env
}

func TestEvalSetWorkflowHappyPath_LaunchesChildrenAndAggregates(t *testing.T) {
	setID := uuid.New()
	sessionA := uuid.New()
	sessionB := uuid.New()
	repo := &fakeEvalSetRepo{
		set: repository.EvalSet{
			ID:                setID,
			Status:            domain.EvalSetStatusQueued,
			MaxConcurrentRuns: 2,
		},
		sessions: []uuid.UUID{sessionA, sessionB},
		packs:    []string{"pack-a", "pack-b"},
		session: map[uuid.UUID]domain.EvalSession{
			sessionA: {ID: sessionA, Status: domain.EvalSessionStatusCompleted},
			sessionB: {ID: sessionB, Status: domain.EvalSessionStatusCompleted},
		},
		runs: map[uuid.UUID][]domain.Run{
			sessionA: {
				{ID: uuid.New(), Status: domain.RunStatusCompleted, ExecutionPlan: json.RawMessage(`{"series":{"matrix_key":"p/a/1"}}`)},
				{ID: uuid.New(), Status: domain.RunStatusCompleted, ExecutionPlan: json.RawMessage(`{"series":{"matrix_key":"p/a/2"}}`)},
			},
			sessionB: {
				{ID: uuid.New(), Status: domain.RunStatusCompleted, ExecutionPlan: json.RawMessage(`{"series":{"matrix_key":"p/b/1"}}`)},
			},
		},
	}

	var launched []uuid.UUID
	var mu sync.Mutex
	env := newEvalSetWorkflowTestEnvironment(repo, func(ctx sdkworkflow.Context, input EvalSessionWorkflowInput) error {
		mu.Lock()
		launched = append(launched, input.EvalSessionID)
		mu.Unlock()
		return nil
	})
	env.ExecuteWorkflow(EvalSetWorkflow, EvalSetWorkflowInput{EvalSetID: setID})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("EvalSetWorkflow: %v", err)
	}
	if repo.set.Status != domain.EvalSetStatusCompleted {
		t.Fatalf("status = %s, want completed", repo.set.Status)
	}
	if repo.result == nil || repo.result.RunCount != 3 || repo.result.SessionCount != 2 {
		t.Fatalf("result = %+v", repo.result)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(launched) != 2 {
		t.Fatalf("launched children = %d, want 2", len(launched))
	}
}

func TestEvalSetWorkflowBudgetExceeded_StopsMidFlight(t *testing.T) {
	setID := uuid.New()
	budget := 1.0
	sessions := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	repo := &fakeEvalSetRepo{
		set: repository.EvalSet{
			ID:                setID,
			WorkspaceID:       uuid.New(),
			Status:            domain.EvalSetStatusQueued,
			MaxConcurrentRuns: 1,
			BudgetUSD:         &budget,
		},
		sessions: sessions,
		packs:    []string{"a", "b", "c"},
		session: map[uuid.UUID]domain.EvalSession{
			sessions[0]: {ID: sessions[0], Status: domain.EvalSessionStatusCompleted},
			sessions[1]: {ID: sessions[1], Status: domain.EvalSessionStatusCompleted},
			sessions[2]: {ID: sessions[2], Status: domain.EvalSessionStatusQueued},
		},
		runs: map[uuid.UUID][]domain.Run{
			sessions[0]: {{ID: uuid.New(), Status: domain.RunStatusCompleted}},
			sessions[1]: {{ID: uuid.New(), Status: domain.RunStatusCompleted}},
		},
	}

	var launched []uuid.UUID
	var mu sync.Mutex
	env := newEvalSetWorkflowTestEnvironment(repo, func(ctx sdkworkflow.Context, input EvalSessionWorkflowInput) error {
		mu.Lock()
		launched = append(launched, input.EvalSessionID)
		mu.Unlock()
		return nil
	})
	env.ExecuteWorkflow(EvalSetWorkflow, EvalSetWorkflowInput{EvalSetID: setID})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("EvalSetWorkflow: %v", err)
	}
	if repo.set.Status != domain.EvalSetStatusBudgetExceeded {
		t.Fatalf("status = %s, want budget_exceeded (transitions %v)", repo.set.Status, repo.statuses)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(launched) != 2 {
		t.Fatalf("launched = %d, want 2 (third blocked by budget)", len(launched))
	}
	if repo.result == nil {
		t.Fatal("expected partial aggregate result")
	}
	var agg map[string]any
	if err := json.Unmarshal(repo.result.Aggregate, &agg); err != nil {
		t.Fatal(err)
	}
	if agg["outcome"] != "budget_exceeded" && repo.set.SpentUSD < budget {
		t.Fatalf("aggregate=%v spent=%v", agg, repo.set.SpentUSD)
	}
}

func TestEvalSetWorkflowCancel_SettlesCancelled(t *testing.T) {
	setID := uuid.New()
	sessionA := uuid.New()
	repo := &fakeEvalSetRepo{
		set: repository.EvalSet{
			ID:                setID,
			Status:            domain.EvalSetStatusQueued,
			MaxConcurrentRuns: 1,
		},
		sessions: []uuid.UUID{sessionA},
		packs:    []string{"pack-a"},
		session:  map[uuid.UUID]domain.EvalSession{sessionA: {ID: sessionA, Status: domain.EvalSessionStatusRunning}},
		runs:     map[uuid.UUID][]domain.Run{},
	}

	env := newEvalSetWorkflowTestEnvironment(repo, func(ctx sdkworkflow.Context, input EvalSessionWorkflowInput) error {
		return sdkworkflow.Sleep(ctx, 0) // allow cancel injection
	})
	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, 0)
	env.ExecuteWorkflow(EvalSetWorkflow, EvalSetWorkflowInput{EvalSetID: setID})
	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if repo.set.Status != domain.EvalSetStatusCancelled {
		t.Fatalf("status = %s, want cancelled (got transitions %v)", repo.set.Status, repo.statuses)
	}
}
