package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
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
	childCtxBase := sdkworkflow.WithChildOptions(ctx, sdkworkflow.ChildWorkflowOptions{
		ParentClosePolicy: enumspb.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
	})

	futures := make([]sdkworkflow.ChildWorkflowFuture, 0, len(runAgents))
	for _, runAgent := range runAgents {
		childCtx := sdkworkflow.WithChildOptions(childCtxBase, sdkworkflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s/%s/%s", RunAgentWorkflowName, runAgent.RunID, runAgent.ID),
		})
		future := sdkworkflow.ExecuteChildWorkflow(childCtx, RunAgentWorkflowName, RunAgentWorkflowInput{
			RunID:      runAgent.RunID,
			RunAgentID: runAgent.ID,
		})
		futures = append(futures, future)
	}

	var firstErr error
	for _, future := range futures {
		if err := future.Get(ctx, nil); err != nil && firstErr == nil {
			firstErr = err
		}
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
	return errors.Is(err, repository.ErrRunNotFound) ||
		errors.Is(err, repository.ErrTemporalIDConflict) ||
		errors.Is(err, ErrRunMustBeQueued) ||
		hasApplicationErrorType(err, repositoryRunNotFoundErrorType) ||
		hasApplicationErrorType(err, repositoryTemporalIDConflictType)
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
