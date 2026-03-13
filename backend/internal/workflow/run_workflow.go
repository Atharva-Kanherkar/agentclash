package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

const (
	defaultActivityTimeout = 5 * time.Second
	fakeStageDelay         = 1 * time.Second
)

var defaultActivityOptions = sdkworkflow.ActivityOptions{
	StartToCloseTimeout: defaultActivityTimeout,
	RetryPolicy: &temporal.RetryPolicy{
		MaximumAttempts: 1,
	},
}

func RunWorkflow(ctx sdkworkflow.Context, input RunWorkflowInput) error {
	ctx = sdkworkflow.WithActivityOptions(ctx, defaultActivityOptions)

	err := runWorkflow(ctx, input)
	if err == nil {
		return nil
	}

	if isWorkflowCanceled(err) {
		return markRunCancelled(ctx, input.RunID, err)
	}
	if shouldSkipRunFailureTransition(err) {
		return err
	}

	return markRunFailed(ctx, input.RunID, err)
}

func runWorkflow(ctx sdkworkflow.Context, input RunWorkflowInput) error {
	run, err := loadRun(ctx, input.RunID)
	if err != nil {
		return err
	}
	if err := validateRunQueued(run); err != nil {
		return err
	}

	runAgents, err := listRunAgents(ctx, input.RunID)
	if err != nil {
		return err
	}
	if len(runAgents) == 0 {
		return fmt.Errorf("%w: run %s", ErrRunHasNoAgents, input.RunID)
	}

	info := sdkworkflow.GetInfo(ctx)
	if err := attachRunTemporalIDs(ctx, input.RunID, info.WorkflowExecution.ID, info.WorkflowExecution.RunID); err != nil {
		return err
	}
	if err := transitionRunStatus(ctx, input.RunID, domain.RunStatusProvisioning, stringPtr("run workflow provisioning started")); err != nil {
		return err
	}
	if err := transitionRunStatus(ctx, input.RunID, domain.RunStatusRunning, stringPtr("run workflow launched run-agent children")); err != nil {
		return err
	}

	if err := executeRunAgents(ctx, runAgents); err != nil {
		return err
	}

	if err := transitionRunStatus(ctx, input.RunID, domain.RunStatusScoring, stringPtr("all run-agent workflows completed")); err != nil {
		return err
	}
	if err := transitionRunStatus(ctx, input.RunID, domain.RunStatusCompleted, stringPtr("fake scoring completed")); err != nil {
		return err
	}

	return nil
}

func executeRunAgents(ctx sdkworkflow.Context, runAgents []domain.RunAgent) error {
	selector := sdkworkflow.NewSelector(ctx)
	childCancels := make([]sdkworkflow.CancelFunc, 0, len(runAgents))
	var firstErr error
	completedChildren := 0

	for _, runAgent := range runAgents {
		childCtx, cancel := sdkworkflow.WithCancel(ctx)
		childCtx = sdkworkflow.WithChildOptions(childCtx, sdkworkflow.ChildWorkflowOptions{
			WorkflowID:        fmt.Sprintf("%s/%s/%s", RunAgentWorkflowName, runAgent.RunID, runAgent.ID),
			ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
		})
		childCancels = append(childCancels, cancel)

		future := sdkworkflow.ExecuteChildWorkflow(childCtx, RunAgentWorkflowName, RunAgentWorkflowInput{
			RunID:      runAgent.RunID,
			RunAgentID: runAgent.ID,
		})
		selector.AddFuture(future, func(f sdkworkflow.Future) {
			completedChildren++

			if firstErr != nil {
				return
			}

			if err := f.Get(ctx, nil); err != nil {
				firstErr = err
				for _, childCancel := range childCancels {
					childCancel()
				}
			}
		})
	}

	for completedChildren < len(runAgents) && firstErr == nil {
		selector.Select(ctx)
	}

	return firstErr
}

func loadRun(ctx sdkworkflow.Context, runID uuid.UUID) (domain.Run, error) {
	var run domain.Run
	err := sdkworkflow.ExecuteActivity(ctx, loadRunActivityName, LoadRunInput{RunID: runID}).Get(ctx, &run)
	return run, err
}

func listRunAgents(ctx sdkworkflow.Context, runID uuid.UUID) ([]domain.RunAgent, error) {
	var runAgents []domain.RunAgent
	err := sdkworkflow.ExecuteActivity(ctx, listRunAgentsActivityName, ListRunAgentsInput{RunID: runID}).Get(ctx, &runAgents)
	return runAgents, err
}

func attachRunTemporalIDs(ctx sdkworkflow.Context, runID uuid.UUID, workflowID string, temporalRunID string) error {
	var run domain.Run
	return sdkworkflow.ExecuteActivity(ctx, attachTemporalIDsActivityName, AttachRunTemporalIDsInput{
		RunID:              runID,
		TemporalWorkflowID: workflowID,
		TemporalRunID:      temporalRunID,
	}).Get(ctx, &run)
}

func transitionRunStatus(ctx sdkworkflow.Context, runID uuid.UUID, toStatus domain.RunStatus, reason *string) error {
	var run domain.Run
	return sdkworkflow.ExecuteActivity(ctx, transitionRunStatusActivityName, TransitionRunStatusInput{
		RunID:    runID,
		ToStatus: toStatus,
		Reason:   reason,
	}).Get(ctx, &run)
}

func markRunFailed(ctx sdkworkflow.Context, runID uuid.UUID, workflowErr error) error {
	reason := workflowErr.Error()
	var run domain.Run
	activityErr := sdkworkflow.ExecuteActivity(ctx, transitionRunStatusActivityName, TransitionRunStatusInput{
		RunID:    runID,
		ToStatus: domain.RunStatusFailed,
		Reason:   &reason,
	}).Get(ctx, &run)
	if activityErr != nil {
		return fmt.Errorf("run workflow failed: %v; additionally failed to mark run failed: %w", workflowErr, activityErr)
	}

	return workflowErr
}

func markRunCancelled(ctx sdkworkflow.Context, runID uuid.UUID, workflowErr error) error {
	disconnectedCtx, _ := sdkworkflow.NewDisconnectedContext(ctx)
	disconnectedCtx = sdkworkflow.WithActivityOptions(disconnectedCtx, defaultActivityOptions)

	reason := "run workflow cancelled"
	var run domain.Run
	activityErr := sdkworkflow.ExecuteActivity(disconnectedCtx, transitionRunStatusActivityName, TransitionRunStatusInput{
		RunID:    runID,
		ToStatus: domain.RunStatusCancelled,
		Reason:   &reason,
	}).Get(disconnectedCtx, &run)
	if activityErr != nil {
		return fmt.Errorf("run workflow cancelled: %v; additionally failed to mark run cancelled: %w", workflowErr, activityErr)
	}

	return workflowErr
}

func shouldSkipRunFailureTransition(err error) bool {
	return errors.Is(err, ErrRunMustBeQueued) ||
		hasApplicationErrorType(err, repositoryRunNotFoundErrorType) ||
		hasApplicationErrorType(err, repositoryTemporalIDConflictType) ||
		hasApplicationErrorType(err, repositoryInvalidTransitionType) ||
		hasApplicationErrorType(err, repositoryTransitionConflictType)
}

func isWorkflowCanceled(err error) bool {
	var canceledErr *temporal.CanceledError
	return errors.As(err, &canceledErr)
}

func hasApplicationErrorType(err error, wantType string) bool {
	var applicationErr *temporal.ApplicationError
	if !errors.As(err, &applicationErr) {
		return false
	}

	return applicationErr.Type() == wantType
}

func stringPtr(value string) *string {
	return &value
}
