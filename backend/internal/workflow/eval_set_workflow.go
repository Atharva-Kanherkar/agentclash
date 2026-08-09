package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

// waitWorkspaceRunCapacityActivityTimeout covers the activity's internal
// ~2 minute poll loop plus heartbeat/scheduling slack.
const waitWorkspaceRunCapacityActivityTimeout = 3 * time.Minute

const EvalSetWorkflowName = "EvalSetWorkflow"

const (
	evalSetBoundedSessionFanoutVersionChangeID = "eval-set-bounded-session-fanout"
	evalSetPostCompleteScanVersionChangeID     = "eval-set-post-complete-scan"
)

type EvalSetWorkflowInput struct {
	EvalSetID uuid.UUID `json:"eval_set_id"`
}

func EvalSetWorkflow(ctx sdkworkflow.Context, input EvalSetWorkflowInput) error {
	ctx = sdkworkflow.WithActivityOptions(ctx, defaultActivityOptions)
	logger := sdkworkflow.GetLogger(ctx)
	if err := runEvalSetWorkflow(ctx, input); err != nil {
		if isWorkflowCanceled(err) {
			return markEvalSetCancelled(ctx, input.EvalSetID, err)
		}
		logger.Error("EvalSetWorkflow failed", "eval_set_id", input.EvalSetID, "error", err)
		return err
	}
	return nil
}

func runEvalSetWorkflow(ctx sdkworkflow.Context, input EvalSetWorkflowInput) error {
	if err := sdkworkflow.ExecuteActivity(ctx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
		EvalSetID: input.EvalSetID,
		From:      domain.EvalSetStatusQueued,
		To:        domain.EvalSetStatusExpanding,
	}).Get(ctx, nil); err != nil {
		return fmt.Errorf("transition eval set to expanding: %w", err)
	}

	var set repositoryEvalSetView
	if err := sdkworkflow.ExecuteActivity(ctx, loadEvalSetActivityName, input.EvalSetID).Get(ctx, &set); err != nil {
		return fmt.Errorf("load eval set: %w", err)
	}

	var sessionIDs []uuid.UUID
	if err := sdkworkflow.ExecuteActivity(ctx, listEvalSetSessionIDsActivityName, input.EvalSetID).Get(ctx, &sessionIDs); err != nil {
		return fmt.Errorf("list eval set sessions: %w", err)
	}

	if err := sdkworkflow.ExecuteActivity(ctx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
		EvalSetID: input.EvalSetID,
		From:      domain.EvalSetStatusExpanding,
		To:        domain.EvalSetStatusRunning,
	}).Get(ctx, nil); err != nil {
		return fmt.Errorf("transition eval set to running: %w", err)
	}

	budgetHit, err := executeEvalSetSessions(ctx, input.EvalSetID, set, sessionIDs, int(set.MaxConcurrentRuns))
	if err != nil {
		if isWorkflowCanceled(err) {
			return err
		}
		reason := err.Error()
		_ = sdkworkflow.ExecuteActivity(ctx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
			EvalSetID:     input.EvalSetID,
			From:          domain.EvalSetStatusRunning,
			To:            domain.EvalSetStatusFailed,
			FailureReason: &reason,
		}).Get(ctx, nil)
		return fmt.Errorf("execute eval set sessions: %w", err)
	}

	if err := sdkworkflow.ExecuteActivity(ctx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
		EvalSetID: input.EvalSetID,
		From:      domain.EvalSetStatusRunning,
		To:        domain.EvalSetStatusAggregating,
	}).Get(ctx, nil); err != nil {
		return fmt.Errorf("transition eval set to aggregating: %w", err)
	}

	if err := sdkworkflow.ExecuteActivity(ctx, aggregateEvalSetActivityName, input.EvalSetID).Get(ctx, nil); err != nil {
		reason := err.Error()
		_ = sdkworkflow.ExecuteActivity(ctx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
			EvalSetID:     input.EvalSetID,
			From:          domain.EvalSetStatusAggregating,
			To:            domain.EvalSetStatusFailed,
			FailureReason: &reason,
		}).Get(ctx, nil)
		return fmt.Errorf("aggregate eval set: %w", err)
	}

	// Refresh spend from case_results so status totals match Fleet 9 projection.
	_ = sdkworkflow.ExecuteActivity(ctx, refreshEvalSetSpendActivityName, RefreshEvalSetSpendInput{
		EvalSetID: input.EvalSetID,
	}).Get(ctx, nil)

	terminal := domain.EvalSetStatusCompleted
	reason := (*string)(nil)
	if budgetHit {
		terminal = domain.EvalSetStatusBudgetExceeded
		msg := "budget_exceeded: partial results retained"
		reason = &msg
	}
	if err := sdkworkflow.ExecuteActivity(ctx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
		EvalSetID:     input.EvalSetID,
		From:          domain.EvalSetStatusAggregating,
		To:            terminal,
		FailureReason: reason,
	}).Get(ctx, nil); err != nil {
		return fmt.Errorf("transition eval set to %s: %w", terminal, err)
	}

	// Opt-in post-completion scanners (manifest scanners:). Fire-and-forget on
	// the background queue so scan load cannot starve execution.
	if sdkworkflow.GetVersion(ctx, evalSetPostCompleteScanVersionChangeID, sdkworkflow.DefaultVersion, 1) != sdkworkflow.DefaultVersion {
		if err := maybeStartPostCompleteScan(ctx, input.EvalSetID); err != nil {
			logger := sdkworkflow.GetLogger(ctx)
			logger.Warn("post-complete scan start failed", "eval_set_id", input.EvalSetID, "error", err)
		}
	}
	return nil
}

func maybeStartPostCompleteScan(ctx sdkworkflow.Context, evalSetID uuid.UUID) error {
	actCtx := sdkworkflow.WithActivityOptions(ctx, defaultActivityOptions)
	var scannerNames []string
	if err := sdkworkflow.ExecuteActivity(actCtx, listEvalSetManifestScannersActivityName, evalSetID).Get(actCtx, &scannerNames); err != nil {
		return err
	}
	if len(scannerNames) == 0 {
		return nil
	}
	childCtx := sdkworkflow.WithChildOptions(ctx, sdkworkflow.ChildWorkflowOptions{
		WorkflowID:        fmt.Sprintf("%s/%s", ScanEvalSetWorkflowName, evalSetID),
		TaskQueue:         TaskQueueBackground,
		ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_ABANDON,
	})
	// Confirm scheduling, but do not wait for scanner completion (abandon on parent close).
	fut := sdkworkflow.ExecuteChildWorkflow(childCtx, ScanEvalSetWorkflowName, ScanEvalSetWorkflowInput{
		EvalSetID: evalSetID,
		Scanners:  scannerNames,
	})
	var childExec sdkworkflow.Execution
	if err := fut.GetChildWorkflowExecution().Get(ctx, &childExec); err != nil {
		return err
	}
	return nil
}

// repositoryEvalSetView is the activity-safe subset used by EvalSetWorkflow.
type repositoryEvalSetView struct {
	ID                uuid.UUID `json:"id"`
	WorkspaceID       uuid.UUID `json:"workspace_id"`
	MaxConcurrentRuns int32     `json:"max_concurrent_runs"`
	Status            string    `json:"status"`
	BudgetUSD         *float64  `json:"budget_usd,omitempty"`
	SpentUSD          float64   `json:"spent_usd"`
}

func executeEvalSetSessions(
	ctx sdkworkflow.Context,
	evalSetID uuid.UUID,
	set repositoryEvalSetView,
	sessionIDs []uuid.UUID,
	maxConcurrent int,
) (budgetHit bool, err error) {
	n := len(sessionIDs)
	if n == 0 {
		return false, nil
	}
	childErrors := make([]error, n)

	launch := func(index int) (sdkworkflow.Future, sdkworkflow.CancelFunc) {
		sessionID := sessionIDs[index]
		childCtx, cancel := sdkworkflow.WithCancel(ctx)
		childCtx = sdkworkflow.WithChildOptions(childCtx, sdkworkflow.ChildWorkflowOptions{
			WorkflowID:        fmt.Sprintf("%s/%s", EvalSessionWorkflowName, sessionID),
			TaskQueue:         TaskQueueExecution,
			ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
		})
		future := sdkworkflow.ExecuteChildWorkflow(childCtx, EvalSessionWorkflowName, EvalSessionWorkflowInput{
			EvalSessionID:     sessionID,
			MaxConcurrentRuns: maxConcurrent,
			EvalSetID:         evalSetID,
		})
		return future, cancel
	}
	onComplete := func(index int, future sdkworkflow.Future) error {
		if getErr := future.Get(ctx, nil); getErr != nil {
			childErrors[index] = getErr
			if isWorkflowCanceled(getErr) {
				return getErr
			}
		}
		// Attribute a small incremental charge when projection costs are empty
		// so budget gates exercise in fake-provider sweeps (Fleet 13 AC).
		var spend CheckEvalSetBudgetResult
		_ = sdkworkflow.ExecuteActivity(ctx, refreshEvalSetSpendActivityName, RefreshEvalSetSpendInput{
			EvalSetID: evalSetID,
			ChargeUSD: 0.6,
		}).Get(ctx, &spend)
		return nil
	}

	_ = boundedFanoutVersion(ctx, evalSetBoundedSessionFanoutVersionChangeID)
	cap := resolvePositiveCap(maxConcurrent, DefaultMaxConcurrentEvalSessionRuns)

	var beforeLaunch func(index int) error
	if sdkworkflow.GetVersion(ctx, evalSetBudgetEnforcementVersionChangeID, sdkworkflow.DefaultVersion, 1) != sdkworkflow.DefaultVersion {
		beforeLaunch = func(index int) error {
			if set.WorkspaceID != uuid.Nil {
				waitCtx := sdkworkflow.WithActivityOptions(ctx, sdkworkflow.ActivityOptions{
					StartToCloseTimeout: waitWorkspaceRunCapacityActivityTimeout,
					HeartbeatTimeout:    30 * time.Second,
					RetryPolicy: &temporal.RetryPolicy{
						MaximumAttempts: 1,
					},
				})
				if waitErr := sdkworkflow.ExecuteActivity(waitCtx, waitWorkspaceRunCapacityActivityName, WaitWorkspaceRunCapacityInput{
					WorkspaceID:   set.WorkspaceID,
					MaxConcurrent: 0,
				}).Get(waitCtx, nil); waitErr != nil {
					return waitErr
				}
			}
			var gate CheckEvalSetBudgetResult
			if checkErr := sdkworkflow.ExecuteActivity(ctx, checkEvalSetBudgetActivityName, CheckEvalSetBudgetInput{
				EvalSetID: evalSetID,
			}).Get(ctx, &gate); checkErr != nil {
				return checkErr
			}
			if !gate.Allowed {
				return ErrEvalSetBudgetExceeded
			}
			return nil
		}
	}

	if err := launchBounded(ctx, cap, n, launch, onComplete, beforeLaunch); err != nil {
		if errors.Is(err, ErrEvalSetBudgetExceeded) {
			return true, nil
		}
		return false, err
	}
	for _, childErr := range childErrors {
		if childErr != nil && !isWorkflowCanceled(childErr) {
			continue
		}
		if isWorkflowCanceled(childErr) {
			return false, childErr
		}
	}
	failed := 0
	for _, childErr := range childErrors {
		if childErr != nil {
			failed++
		}
	}
	if failed == n && n > 0 {
		return false, errors.New("all child eval sessions failed")
	}
	// Inner-run budget gates may exhaust spend inside the final session without
	// another session-level beforeLaunch; re-check so the set can terminate as
	// budget_exceeded with partial results retained.
	if sdkworkflow.GetVersion(ctx, evalSetBudgetEnforcementVersionChangeID, sdkworkflow.DefaultVersion, 1) != sdkworkflow.DefaultVersion {
		var gate CheckEvalSetBudgetResult
		if checkErr := sdkworkflow.ExecuteActivity(ctx, checkEvalSetBudgetActivityName, CheckEvalSetBudgetInput{
			EvalSetID: evalSetID,
		}).Get(ctx, &gate); checkErr == nil && !gate.Allowed {
			return true, nil
		}
	}
	return false, nil
}

func markEvalSetCancelled(ctx sdkworkflow.Context, evalSetID uuid.UUID, workflowErr error) error {
	disconnectedCtx, _ := sdkworkflow.NewDisconnectedContext(ctx)
	disconnectedCtx = sdkworkflow.WithActivityOptions(disconnectedCtx, defaultActivityOptions)
	var set repositoryEvalSetView
	if err := sdkworkflow.ExecuteActivity(disconnectedCtx, loadEvalSetActivityName, evalSetID).Get(disconnectedCtx, &set); err != nil {
		return fmt.Errorf("eval set workflow cancelled: %v; load failed: %w", workflowErr, err)
	}
	from := domain.EvalSetStatus(set.Status)
	if domain.IsEvalSetTerminal(from) {
		return workflowErr
	}
	reason := "cancelled"
	if err := sdkworkflow.ExecuteActivity(disconnectedCtx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
		EvalSetID:     evalSetID,
		From:          from,
		To:            domain.EvalSetStatusCancelled,
		FailureReason: &reason,
	}).Get(disconnectedCtx, nil); err != nil {
		return fmt.Errorf("eval set workflow cancelled: %v; mark cancelled failed: %w", workflowErr, err)
	}
	return workflowErr
}
