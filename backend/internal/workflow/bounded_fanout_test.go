package workflow

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
	sdkactivity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

func TestExpandTaskQueues_DefaultsAndLegacy(t *testing.T) {
	if got := ExpandTaskQueues(nil); len(got) != 3 {
		t.Fatalf("nil → %v, want 3 queues", got)
	}
	got := ExpandTaskQueues([]string{LegacyTaskQueue})
	if len(got) != 3 {
		t.Fatalf("legacy → %v", got)
	}
	got = ExpandTaskQueues([]string{TaskQueueExecution, TaskQueueScoring})
	if len(got) != 2 || got[0] != TaskQueueExecution || got[1] != TaskQueueScoring {
		t.Fatalf("subset → %v", got)
	}
}

func TestLaunchBounded_RespectsMaxInFlight(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	suite.SetDisableRegistrationAliasing(true)
	env := suite.NewTestWorkflowEnvironment()

	var (
		mu          sync.Mutex
		inFlight    int
		maxInFlight int
	)

	sleepActivity := func(d time.Duration) error {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()
		time.Sleep(d)
		mu.Lock()
		inFlight--
		mu.Unlock()
		return nil
	}
	env.RegisterActivityWithOptions(sleepActivity, sdkactivity.RegisterOptions{Name: "workflow.test_sleep"})

	boundedWorkflow := func(ctx sdkworkflow.Context, n int, cap int) error {
		launch := func(index int) sdkworkflow.Future {
			return sdkworkflow.ExecuteActivity(
				sdkworkflow.WithActivityOptions(ctx, defaultActivityOptions),
				"workflow.test_sleep",
				30*time.Millisecond,
			)
		}
		onComplete := func(index int, future sdkworkflow.Future) error {
			return future.Get(ctx, nil)
		}
		return launchBounded(ctx, cap, n, adaptLaunch(launch), onComplete, nil)
	}
	env.RegisterWorkflow(boundedWorkflow)
	env.ExecuteWorkflow(boundedWorkflow, 12, 3)

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	mu.Lock()
	peak := maxInFlight
	mu.Unlock()
	if peak > 3 {
		t.Fatalf("peak in-flight = %d, want ≤3", peak)
	}
	if peak < 2 {
		t.Fatalf("peak in-flight = %d, want ≥2", peak)
	}
}

func TestEvalSessionWorkflowBoundedFanoutCap(t *testing.T) {
	sessionID := uuid.New()
	repo := newFakeRunRepository(fixtureRun(uuid.New(), domain.RunStatusQueued))
	childRuns := make([]domain.Run, 0, 50)
	for i := 0; i < 50; i++ {
		childRuns = append(childRuns, fixtureChildRun(uuid.New(), sessionID))
	}
	repo.setEvalSession(fixtureEvalSession(sessionID, domain.EvalSessionStatusQueued), childRuns...)

	var (
		mu          sync.Mutex
		inFlight    int
		maxInFlight int
	)
	env := newEvalSessionWorkflowTestEnvironment(repo, nil, func(ctx sdkworkflow.Context, input RunWorkflowInput) error {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()
		_ = sdkworkflow.Sleep(ctx, 15*time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
		return nil
	})

	env.ExecuteWorkflow(EvalSessionWorkflow, EvalSessionWorkflowInput{
		EvalSessionID:     sessionID,
		MaxConcurrentRuns: 16,
	})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	mu.Lock()
	peak := maxInFlight
	mu.Unlock()
	if peak > 16 {
		t.Fatalf("peak in-flight child runs = %d, want ≤16", peak)
	}
	if peak < 2 {
		t.Fatalf("peak in-flight child runs = %d, want ≥2", peak)
	}
}

func TestEvalSessionWorkflowDefaultVersionUnboundedReplay(t *testing.T) {
	sessionID := uuid.New()
	repo := newFakeRunRepository(fixtureRun(uuid.New(), domain.RunStatusQueued))
	repo.setEvalSession(
		fixtureEvalSession(sessionID, domain.EvalSessionStatusQueued),
		fixtureChildRun(uuid.New(), sessionID),
		fixtureChildRun(uuid.New(), sessionID),
	)

	var started atomic.Int64
	env := newEvalSessionWorkflowTestEnvironment(repo, nil, func(ctx sdkworkflow.Context, input RunWorkflowInput) error {
		started.Add(1)
		return nil
	})
	env.OnGetVersion(evalSessionBoundedFanoutVersionChangeID, sdkworkflow.DefaultVersion, sdkworkflow.Version(1)).
		Return(sdkworkflow.DefaultVersion)
	env.OnGetVersion(evalSessionRefreshChildRunsVersionChangeID, sdkworkflow.DefaultVersion, sdkworkflow.Version(1)).
		Return(sdkworkflow.DefaultVersion)
	env.OnGetVersion(taskQueuePartitionVersionChangeID, sdkworkflow.DefaultVersion, sdkworkflow.Version(1)).
		Return(sdkworkflow.DefaultVersion)

	env.ExecuteWorkflow(EvalSessionWorkflow, EvalSessionWorkflowInput{EvalSessionID: sessionID})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if started.Load() != 2 {
		t.Fatalf("started = %d, want 2", started.Load())
	}
}
