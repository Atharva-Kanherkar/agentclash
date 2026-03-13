package workflow

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestRunWorkflowHappyPath(t *testing.T) {
	runID := uuid.New()
	runAgentID := uuid.New()
	repo := newFakeRunRepository(
		fixtureRun(runID, domain.RunStatusQueued),
		fixtureRunAgent(runID, runAgentID, 0),
	)

	env := newTestWorkflowEnvironment(repo, FakeWorkHooks{})
	env.ExecuteWorkflow(RunWorkflow, RunWorkflowInput{RunID: runID})

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("RunWorkflow returned error: %v", err)
	}
	if !env.IsWorkflowCompleted() {
		t.Fatalf("workflow did not complete")
	}

	run := repo.currentRun()
	if run.Status != domain.RunStatusCompleted {
		t.Fatalf("run status = %s, want %s", run.Status, domain.RunStatusCompleted)
	}
	if run.TemporalWorkflowID == nil || *run.TemporalWorkflowID != "test-run-workflow" {
		t.Fatalf("temporal workflow id = %v, want %q", run.TemporalWorkflowID, "test-run-workflow")
	}
	if run.TemporalRunID == nil || *run.TemporalRunID == "" {
		t.Fatalf("temporal run id was not stored")
	}

	runStatuses := repo.runStatusSequence()
	wantRunStatuses := []domain.RunStatus{
		domain.RunStatusProvisioning,
		domain.RunStatusRunning,
		domain.RunStatusScoring,
		domain.RunStatusCompleted,
	}
	if !equalRunStatuses(runStatuses, wantRunStatuses) {
		t.Fatalf("run statuses = %v, want %v", runStatuses, wantRunStatuses)
	}

	runAgent := repo.currentRunAgent(runAgentID)
	if runAgent.Status != domain.RunAgentStatusCompleted {
		t.Fatalf("run agent status = %s, want %s", runAgent.Status, domain.RunAgentStatusCompleted)
	}
	wantRunAgentStatuses := []domain.RunAgentStatus{
		domain.RunAgentStatusReady,
		domain.RunAgentStatusExecuting,
		domain.RunAgentStatusEvaluating,
		domain.RunAgentStatusCompleted,
	}
	if got := repo.runAgentStatusSequence(runAgentID); !equalRunAgentStatuses(got, wantRunAgentStatuses) {
		t.Fatalf("run-agent statuses = %v, want %v", got, wantRunAgentStatuses)
	}

	if repo.setTemporalIDsCount() != 1 {
		t.Fatalf("set temporal ids call count = %d, want 1", repo.setTemporalIDsCount())
	}
	if !repo.hasCallPrefix("TransitionRunStatus") {
		t.Fatalf("expected repository TransitionRunStatus to be used")
	}
	if !repo.hasCallPrefix("TransitionRunAgentStatus") {
		t.Fatalf("expected repository TransitionRunAgentStatus to be used")
	}
}

func TestRunWorkflowStartsOneChildPerRunAgent(t *testing.T) {
	runID := uuid.New()
	firstRunAgentID := uuid.New()
	secondRunAgentID := uuid.New()
	repo := newFakeRunRepository(
		fixtureRun(runID, domain.RunStatusQueued),
		fixtureRunAgent(runID, firstRunAgentID, 0),
		fixtureRunAgent(runID, secondRunAgentID, 1),
	)

	env := newTestWorkflowEnvironment(repo, FakeWorkHooks{})
	env.ExecuteWorkflow(RunWorkflow, RunWorkflowInput{RunID: runID})

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("RunWorkflow returned error: %v", err)
	}
	if repo.callCountWithPrefix("GetRunAgentByID:") != 2 {
		t.Fatalf("GetRunAgentByID call count = %d, want 2", repo.callCountWithPrefix("GetRunAgentByID:"))
	}
	if repo.currentRunAgent(firstRunAgentID).Status != domain.RunAgentStatusCompleted {
		t.Fatalf("first run agent did not complete")
	}
	if repo.currentRunAgent(secondRunAgentID).Status != domain.RunAgentStatusCompleted {
		t.Fatalf("second run agent did not complete")
	}
}

func TestRunAgentWorkflowHappyPath(t *testing.T) {
	runID := uuid.New()
	runAgentID := uuid.New()
	repo := newFakeRunRepository(
		fixtureRun(runID, domain.RunStatusRunning),
		fixtureRunAgent(runID, runAgentID, 0),
	)

	env := newTestWorkflowEnvironment(repo, FakeWorkHooks{})
	env.ExecuteWorkflow(RunAgentWorkflow, RunAgentWorkflowInput{
		RunID:      runID,
		RunAgentID: runAgentID,
	})

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("RunAgentWorkflow returned error: %v", err)
	}

	runAgent := repo.currentRunAgent(runAgentID)
	if runAgent.Status != domain.RunAgentStatusCompleted {
		t.Fatalf("run agent status = %s, want %s", runAgent.Status, domain.RunAgentStatusCompleted)
	}
}

func TestRunWorkflowCancellationMarksRunCancelled(t *testing.T) {
	runID := uuid.New()
	runAgentID := uuid.New()
	repo := newFakeRunRepository(
		fixtureRun(runID, domain.RunStatusQueued),
		fixtureRunAgent(runID, runAgentID, 0),
	)

	env := newTestWorkflowEnvironment(repo, FakeWorkHooks{})
	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, fakeStageDelay/2)
	env.ExecuteWorkflow(RunWorkflow, RunWorkflowInput{RunID: runID})

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if !temporal.IsCanceledError(err) {
		t.Fatalf("workflow error = %v, want canceled error", err)
	}

	run := repo.currentRun()
	if run.Status != domain.RunStatusCancelled {
		t.Fatalf("run status = %s, want %s", run.Status, domain.RunStatusCancelled)
	}

	runAgent := repo.currentRunAgent(runAgentID)
	if runAgent.Status != domain.RunAgentStatusReady {
		t.Fatalf("run agent status after cancellation = %s, want %s", runAgent.Status, domain.RunAgentStatusReady)
	}
}

func TestRunWorkflowChildFailureMarksRunAndRunAgentFailed(t *testing.T) {
	runID := uuid.New()
	runAgentID := uuid.New()
	repo := newFakeRunRepository(
		fixtureRun(runID, domain.RunStatusQueued),
		fixtureRunAgent(runID, runAgentID, 0),
	)

	env := newTestWorkflowEnvironment(repo, FakeWorkHooks{
		SimulateExecution: func(ctx context.Context, input RunAgentWorkflowInput) error {
			return errors.New("simulated execution failure")
		},
	})
	env.ExecuteWorkflow(RunWorkflow, RunWorkflowInput{RunID: runID})

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected workflow failure")
	}

	run := repo.currentRun()
	if run.Status != domain.RunStatusFailed {
		t.Fatalf("run status = %s, want %s", run.Status, domain.RunStatusFailed)
	}

	runAgent := repo.currentRunAgent(runAgentID)
	if runAgent.Status != domain.RunAgentStatusFailed {
		t.Fatalf("run agent status = %s, want %s", runAgent.Status, domain.RunAgentStatusFailed)
	}
	if runAgent.FailureReason == nil || !strings.Contains(*runAgent.FailureReason, "simulated execution failure") {
		t.Fatalf("run agent failure reason = %v, want simulated execution failure", runAgent.FailureReason)
	}
}

func TestRunWorkflowTemporalIDConflictDoesNotRebindOrAdvanceStatus(t *testing.T) {
	runID := uuid.New()
	runAgentID := uuid.New()
	existingWorkflowID := "existing-workflow"
	existingRunID := "existing-run-id"
	run := fixtureRun(runID, domain.RunStatusQueued)
	run.TemporalWorkflowID = &existingWorkflowID
	run.TemporalRunID = &existingRunID
	repo := newFakeRunRepository(
		run,
		fixtureRunAgent(runID, runAgentID, 0),
	)

	env := newTestWorkflowEnvironment(repo, FakeWorkHooks{})
	env.ExecuteWorkflow(RunWorkflow, RunWorkflowInput{RunID: runID})

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatalf("expected workflow error")
	}
	if !hasApplicationErrorType(err, repositoryTemporalIDConflictType) {
		t.Fatalf("workflow error = %v, want temporal id conflict application error", err)
	}

	persistedRun := repo.currentRun()
	if persistedRun.Status != domain.RunStatusQueued {
		t.Fatalf("run status = %s, want %s", persistedRun.Status, domain.RunStatusQueued)
	}
	if persistedRun.TemporalWorkflowID == nil || *persistedRun.TemporalWorkflowID != existingWorkflowID {
		t.Fatalf("temporal workflow id = %v, want %q", persistedRun.TemporalWorkflowID, existingWorkflowID)
	}
	if repo.runStatusTransitionCount() != 0 {
		t.Fatalf("run status transition count = %d, want 0", repo.runStatusTransitionCount())
	}
}

func newTestWorkflowEnvironment(repo *fakeRunRepository, hooks FakeWorkHooks) *testsuite.TestWorkflowEnvironment {
	var suite testsuite.WorkflowTestSuite
	suite.SetDisableRegistrationAliasing(true)

	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{
		ID:        "test-run-workflow",
		TaskQueue: "workflow-test",
	})
	Register(env, NewActivities(repo, hooks))

	return env
}

type fakeRunRepository struct {
	mu                  sync.Mutex
	run                 domain.Run
	runAgents           map[uuid.UUID]domain.RunAgent
	callLog             []string
	runStatusCalls      []repository.TransitionRunStatusParams
	runAgentStatusCalls []repository.TransitionRunAgentStatusParams
	setTemporalIDsCalls []repository.SetRunTemporalIDsParams
}

func newFakeRunRepository(run domain.Run, runAgents ...domain.RunAgent) *fakeRunRepository {
	runAgentMap := make(map[uuid.UUID]domain.RunAgent, len(runAgents))
	for _, runAgent := range runAgents {
		runAgentMap[runAgent.ID] = cloneRunAgent(runAgent)
	}

	return &fakeRunRepository{
		run:       cloneRun(run),
		runAgents: runAgentMap,
	}
}

func (r *fakeRunRepository) GetRunByID(_ context.Context, id uuid.UUID) (domain.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.run.ID != id {
		return domain.Run{}, repository.ErrRunNotFound
	}
	r.callLog = append(r.callLog, "GetRunByID")

	return cloneRun(r.run), nil
}

func (r *fakeRunRepository) ListRunAgentsByRunID(_ context.Context, runID uuid.UUID) ([]domain.RunAgent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callLog = append(r.callLog, "ListRunAgentsByRunID")

	runAgents := make([]domain.RunAgent, 0, len(r.runAgents))
	for _, runAgent := range r.runAgents {
		if runAgent.RunID == runID {
			runAgents = append(runAgents, cloneRunAgent(runAgent))
		}
	}
	sort.Slice(runAgents, func(i, j int) bool {
		return runAgents[i].LaneIndex < runAgents[j].LaneIndex
	})

	return runAgents, nil
}

func (r *fakeRunRepository) GetRunAgentByID(_ context.Context, id uuid.UUID) (domain.RunAgent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callLog = append(r.callLog, fmt.Sprintf("GetRunAgentByID:%s", id))

	runAgent, ok := r.runAgents[id]
	if !ok {
		return domain.RunAgent{}, repository.ErrRunAgentNotFound
	}

	return cloneRunAgent(runAgent), nil
}

func (r *fakeRunRepository) SetRunTemporalIDs(_ context.Context, params repository.SetRunTemporalIDsParams) (domain.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callLog = append(r.callLog, fmt.Sprintf("SetRunTemporalIDs:%s", params.RunID))
	r.setTemporalIDsCalls = append(r.setTemporalIDsCalls, params)

	if r.run.ID != params.RunID {
		return domain.Run{}, repository.ErrRunNotFound
	}
	if r.run.TemporalWorkflowID != nil || r.run.TemporalRunID != nil {
		if equalStringPtrs(r.run.TemporalWorkflowID, &params.TemporalWorkflowID) &&
			equalStringPtrs(r.run.TemporalRunID, &params.TemporalRunID) {
			return cloneRun(r.run), nil
		}

		return domain.Run{}, repository.TemporalIDConflictError{
			RunID:                params.RunID,
			ExistingWorkflowID:   cloneStringPtr(r.run.TemporalWorkflowID),
			ExistingTemporalRun:  cloneStringPtr(r.run.TemporalRunID),
			RequestedWorkflowID:  params.TemporalWorkflowID,
			RequestedTemporalRun: params.TemporalRunID,
		}
	}

	r.run.TemporalWorkflowID = &params.TemporalWorkflowID
	r.run.TemporalRunID = &params.TemporalRunID

	return cloneRun(r.run), nil
}

func (r *fakeRunRepository) TransitionRunStatus(_ context.Context, params repository.TransitionRunStatusParams) (domain.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callLog = append(r.callLog, fmt.Sprintf("TransitionRunStatus:%s", params.ToStatus))

	if r.run.ID != params.RunID {
		return domain.Run{}, repository.ErrRunNotFound
	}
	if !r.run.Status.CanTransitionTo(params.ToStatus) {
		return domain.Run{}, repository.InvalidTransitionError{
			Entity: "run",
			From:   string(r.run.Status),
			To:     string(params.ToStatus),
		}
	}

	r.run.Status = params.ToStatus
	now := time.Now().UTC()
	switch params.ToStatus {
	case domain.RunStatusProvisioning:
		if r.run.StartedAt == nil {
			r.run.StartedAt = &now
		}
	case domain.RunStatusCompleted:
		if r.run.FinishedAt == nil {
			r.run.FinishedAt = &now
		}
	case domain.RunStatusFailed:
		if r.run.FailedAt == nil {
			r.run.FailedAt = &now
		}
		if r.run.FinishedAt == nil {
			r.run.FinishedAt = &now
		}
	case domain.RunStatusCancelled:
		if r.run.CancelledAt == nil {
			r.run.CancelledAt = &now
		}
		if r.run.FinishedAt == nil {
			r.run.FinishedAt = &now
		}
	}
	r.runStatusCalls = append(r.runStatusCalls, params)

	return cloneRun(r.run), nil
}

func (r *fakeRunRepository) TransitionRunAgentStatus(_ context.Context, params repository.TransitionRunAgentStatusParams) (domain.RunAgent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callLog = append(r.callLog, fmt.Sprintf("TransitionRunAgentStatus:%s:%s", params.RunAgentID, params.ToStatus))

	runAgent, ok := r.runAgents[params.RunAgentID]
	if !ok {
		return domain.RunAgent{}, repository.ErrRunAgentNotFound
	}
	if !runAgent.Status.CanTransitionTo(params.ToStatus) {
		return domain.RunAgent{}, repository.InvalidTransitionError{
			Entity: "run_agent",
			From:   string(runAgent.Status),
			To:     string(params.ToStatus),
		}
	}

	runAgent.Status = params.ToStatus
	if params.ToStatus == domain.RunAgentStatusFailed && params.FailureReason != nil {
		reason := *params.FailureReason
		runAgent.FailureReason = &reason
	}
	r.runAgents[params.RunAgentID] = runAgent
	r.runAgentStatusCalls = append(r.runAgentStatusCalls, params)

	return cloneRunAgent(runAgent), nil
}

func (r *fakeRunRepository) currentRun() domain.Run {
	r.mu.Lock()
	defer r.mu.Unlock()

	return cloneRun(r.run)
}

func (r *fakeRunRepository) currentRunAgent(id uuid.UUID) domain.RunAgent {
	r.mu.Lock()
	defer r.mu.Unlock()

	return cloneRunAgent(r.runAgents[id])
}

func (r *fakeRunRepository) runStatusSequence() []domain.RunStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	statuses := make([]domain.RunStatus, 0, len(r.runStatusCalls))
	for _, call := range r.runStatusCalls {
		statuses = append(statuses, call.ToStatus)
	}

	return statuses
}

func (r *fakeRunRepository) runAgentStatusSequence(runAgentID uuid.UUID) []domain.RunAgentStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	statuses := make([]domain.RunAgentStatus, 0, len(r.runAgentStatusCalls))
	for _, call := range r.runAgentStatusCalls {
		if call.RunAgentID == runAgentID {
			statuses = append(statuses, call.ToStatus)
		}
	}

	return statuses
}

func (r *fakeRunRepository) runStatusTransitionCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.runStatusCalls)
}

func (r *fakeRunRepository) setTemporalIDsCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.setTemporalIDsCalls)
}

func (r *fakeRunRepository) hasCallPrefix(prefix string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, call := range r.callLog {
		if strings.HasPrefix(call, prefix) {
			return true
		}
	}

	return false
}

func (r *fakeRunRepository) callCountWithPrefix(prefix string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count int
	for _, call := range r.callLog {
		if strings.HasPrefix(call, prefix) {
			count++
		}
	}

	return count
}

func fixtureRun(runID uuid.UUID, status domain.RunStatus) domain.Run {
	createdAt := time.Now().UTC()

	return domain.Run{
		ID:            runID,
		Status:        status,
		Name:          "fixture-run",
		ExecutionMode: "comparison",
		ExecutionPlan: []byte(`{}`),
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
}

func fixtureRunAgent(runID uuid.UUID, runAgentID uuid.UUID, laneIndex int32) domain.RunAgent {
	createdAt := time.Now().UTC()

	return domain.RunAgent{
		ID:        runAgentID,
		RunID:     runID,
		LaneIndex: laneIndex,
		Label:     fmt.Sprintf("lane-%d", laneIndex),
		Status:    domain.RunAgentStatusQueued,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func cloneRun(run domain.Run) domain.Run {
	cloned := run
	cloned.TemporalWorkflowID = cloneStringPtr(run.TemporalWorkflowID)
	cloned.TemporalRunID = cloneStringPtr(run.TemporalRunID)
	cloned.ExecutionPlan = append([]byte(nil), run.ExecutionPlan...)

	return cloned
}

func cloneRunAgent(runAgent domain.RunAgent) domain.RunAgent {
	cloned := runAgent
	cloned.FailureReason = cloneStringPtr(runAgent.FailureReason)

	return cloned
}

func equalStringPtrs(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func equalRunStatuses(left []domain.RunStatus, right []domain.RunStatus) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

func equalRunAgentStatuses(left []domain.RunAgentStatus, right []domain.RunAgentStatus) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}
