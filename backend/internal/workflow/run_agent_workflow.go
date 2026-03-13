package workflow

import (
	"errors"
	"fmt"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

func RunAgentWorkflow(ctx sdkworkflow.Context, input RunAgentWorkflowInput) error {
	ctx = sdkworkflow.WithActivityOptions(ctx, defaultActivityOptions)

	err := runAgentWorkflow(ctx, input)
	if err == nil {
		return nil
	}
	if isWorkflowCanceled(err) || shouldSkipRunAgentFailureTransition(err) {
		return err
	}

	return markRunAgentFailed(ctx, input.RunAgentID, err)
}

func runAgentWorkflow(ctx sdkworkflow.Context, input RunAgentWorkflowInput) error {
	runAgent, err := loadRunAgent(ctx, input.RunAgentID)
	if err != nil {
		return err
	}
	if err := validateRunAgentQueued(runAgent, input.RunID); err != nil {
		return err
	}

	if err := transitionRunAgentStatus(ctx, input.RunAgentID, domain.RunAgentStatusReady, stringPtr("execution lane prepared"), nil); err != nil {
		return err
	}
	if err := sdkworkflow.ExecuteActivity(ctx, prepareLaneActivityName, input).Get(ctx, nil); err != nil {
		return err
	}
	if err := sdkworkflow.Sleep(ctx, fakeStageDelay); err != nil {
		return err
	}
	if err := transitionRunAgentStatus(ctx, input.RunAgentID, domain.RunAgentStatusExecuting, stringPtr("fake execution started"), nil); err != nil {
		return err
	}
	if err := sdkworkflow.ExecuteActivity(ctx, simulateExecutionActivityName, input).Get(ctx, nil); err != nil {
		return err
	}
	if err := sdkworkflow.Sleep(ctx, fakeStageDelay); err != nil {
		return err
	}
	if err := transitionRunAgentStatus(ctx, input.RunAgentID, domain.RunAgentStatusEvaluating, stringPtr("fake evaluation started"), nil); err != nil {
		return err
	}
	if err := sdkworkflow.ExecuteActivity(ctx, simulateEvaluationActivityName, input).Get(ctx, nil); err != nil {
		return err
	}
	if err := sdkworkflow.Sleep(ctx, fakeStageDelay); err != nil {
		return err
	}
	if err := transitionRunAgentStatus(ctx, input.RunAgentID, domain.RunAgentStatusCompleted, stringPtr("fake evaluation completed"), nil); err != nil {
		return err
	}

	return nil
}

func loadRunAgent(ctx sdkworkflow.Context, runAgentID uuid.UUID) (domain.RunAgent, error) {
	var runAgent domain.RunAgent
	err := sdkworkflow.ExecuteActivity(ctx, loadRunAgentActivityName, LoadRunAgentInput{
		RunAgentID: runAgentID,
	}).Get(ctx, &runAgent)
	return runAgent, err
}

func transitionRunAgentStatus(ctx sdkworkflow.Context, runAgentID uuid.UUID, toStatus domain.RunAgentStatus, reason *string, failureReason *string) error {
	var runAgent domain.RunAgent
	return sdkworkflow.ExecuteActivity(ctx, transitionRunAgentStatusName, TransitionRunAgentStatusInput{
		RunAgentID:    runAgentID,
		ToStatus:      toStatus,
		Reason:        reason,
		FailureReason: failureReason,
	}).Get(ctx, &runAgent)
}

func markRunAgentFailed(ctx sdkworkflow.Context, runAgentID uuid.UUID, workflowErr error) error {
	reason := workflowErr.Error()
	var runAgent domain.RunAgent
	activityErr := sdkworkflow.ExecuteActivity(ctx, transitionRunAgentStatusName, TransitionRunAgentStatusInput{
		RunAgentID:    runAgentID,
		ToStatus:      domain.RunAgentStatusFailed,
		Reason:        &reason,
		FailureReason: &reason,
	}).Get(ctx, &runAgent)
	if activityErr != nil {
		return fmt.Errorf("run-agent workflow failed: %v; additionally failed to mark run agent failed: %w", workflowErr, activityErr)
	}

	return workflowErr
}

func shouldSkipRunAgentFailureTransition(err error) bool {
	var canceledErr *temporal.CanceledError
	return errors.As(err, &canceledErr) ||
		errors.Is(err, repository.ErrRunAgentNotFound) ||
		errors.Is(err, ErrRunAgentMustBeQueued) ||
		errors.Is(err, ErrRunAgentRunMismatch) ||
		hasApplicationErrorType(err, repositoryRunAgentNotFoundErrorType)
}
