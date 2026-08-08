package workflow

import (
	"errors"
	"fmt"

	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

const EvalSetWorkflowName = "EvalSetWorkflow"

const evalSetBoundedSessionFanoutVersionChangeID = "eval-set-bounded-session-fanout"

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

	if err := executeEvalSetSessions(ctx, sessionIDs, int(set.MaxConcurrentRuns)); err != nil {
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

	if err := sdkworkflow.ExecuteActivity(ctx, transitionEvalSetStatusActivityName, TransitionEvalSetStatusInput{
		EvalSetID: input.EvalSetID,
		From:      domain.EvalSetStatusAggregating,
		To:        domain.EvalSetStatusCompleted,
	}).Get(ctx, nil); err != nil {
		return fmt.Errorf("transition eval set to completed: %w", err)
	}
	return nil
}

// repositoryEvalSetView is the activity-safe subset used by EvalSetWorkflow.
type repositoryEvalSetView struct {
	ID                uuid.UUID `json:"id"`
	MaxConcurrentRuns int32     `json:"max_concurrent_runs"`
	Status            string    `json:"status"`
}

func executeEvalSetSessions(ctx sdkworkflow.Context, sessionIDs []uuid.UUID, maxConcurrent int) error {
	n := len(sessionIDs)
	if n == 0 {
		return nil
	}
	childErrors := make([]error, n)

	launch := func(index int) sdkworkflow.Future {
		sessionID := sessionIDs[index]
		childCtx := sdkworkflow.WithChildOptions(ctx, sdkworkflow.ChildWorkflowOptions{
			WorkflowID:        fmt.Sprintf("%s/%s", EvalSessionWorkflowName, sessionID),
			TaskQueue:         TaskQueueExecution,
			ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
		})
		return sdkworkflow.ExecuteChildWorkflow(childCtx, EvalSessionWorkflowName, EvalSessionWorkflowInput{
			EvalSessionID:     sessionID,
			MaxConcurrentRuns: maxConcurrent,
		})
	}
	onComplete := func(index int, future sdkworkflow.Future) error {
		if err := future.Get(ctx, nil); err != nil {
			childErrors[index] = err
			if isWorkflowCanceled(err) {
				return err
			}
		}
		return nil
	}

	// GetVersion pins the bounded-launch decision for future history changes.
	_ = boundedFanoutVersion(ctx, evalSetBoundedSessionFanoutVersionChangeID)
	cap := resolvePositiveCap(maxConcurrent, DefaultMaxConcurrentEvalSessionRuns)
	if err := launchBounded(ctx, cap, n, launch, onComplete); err != nil {
		return err
	}
	for _, err := range childErrors {
		if err != nil && !isWorkflowCanceled(err) {
			// Soft-fail individual sessions: aggregation still runs. Hard-fail
			// only if every child failed with a non-cancel error.
			continue
		}
		if isWorkflowCanceled(err) {
			return err
		}
	}
	failed := 0
	for _, err := range childErrors {
		if err != nil {
			failed++
		}
	}
	if failed == n && n > 0 {
		return errors.New("all child eval sessions failed")
	}
	return nil
}

func markEvalSetCancelled(ctx sdkworkflow.Context, evalSetID uuid.UUID, workflowErr error) error {
	disconnectedCtx, _ := sdkworkflow.NewDisconnectedContext(ctx)
	disconnectedCtx = sdkworkflow.WithActivityOptions(disconnectedCtx, defaultActivityOptions)
	var set repositoryEvalSetView
	if err := sdkworkflow.ExecuteActivity(disconnectedCtx, loadEvalSetActivityName, evalSetID).Get(disconnectedCtx, &set); err != nil {
		return fmt.Errorf("eval set workflow cancelled: %v; load failed: %w", workflowErr, err)
	}
	from := domain.EvalSetStatus(set.Status)
	if from == domain.EvalSetStatusCancelled || from == domain.EvalSetStatusCompleted || from == domain.EvalSetStatusFailed {
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
