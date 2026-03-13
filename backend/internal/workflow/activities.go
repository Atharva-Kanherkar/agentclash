package workflow

import (
	"context"
	"errors"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
)

const (
	loadRunActivityName                  = "workflow.load_run"
	listRunAgentsActivityName            = "workflow.list_run_agents"
	loadRunAgentActivityName             = "workflow.load_run_agent"
	attachTemporalIDsActivityName        = "workflow.attach_run_temporal_ids"
	transitionRunStatusActivityName      = "workflow.transition_run_status"
	transitionRunAgentStatusActivityName = "workflow.transition_run_agent_status"
	prepareLaneActivityName              = "workflow.prepare_execution_lane"
	simulateExecutionActivityName        = "workflow.simulate_execution"
	simulateEvaluationActivityName       = "workflow.simulate_evaluation"
)

const (
	repositoryRunNotFoundErrorType      = "repository.ErrRunNotFound"
	repositoryRunAgentNotFoundErrorType = "repository.ErrRunAgentNotFound"
	repositoryTemporalIDConflictType    = "repository.ErrTemporalIDConflict"
	repositoryInvalidTransitionType     = "repository.ErrInvalidTransition"
	repositoryTransitionConflictType    = "repository.ErrTransitionConflict"
)

type FakeWorkHooks struct {
	PrepareExecutionLane func(ctx context.Context, input RunAgentWorkflowInput) error
	SimulateExecution    func(ctx context.Context, input RunAgentWorkflowInput) error
	SimulateEvaluation   func(ctx context.Context, input RunAgentWorkflowInput) error
}

type Activities struct {
	repo  RunRepository
	hooks FakeWorkHooks
}

type LoadRunInput struct {
	RunID uuid.UUID `json:"run_id"`
}

type ListRunAgentsInput struct {
	RunID uuid.UUID `json:"run_id"`
}

type LoadRunAgentInput struct {
	RunAgentID uuid.UUID `json:"run_agent_id"`
}

type AttachRunTemporalIDsInput struct {
	RunID              uuid.UUID `json:"run_id"`
	TemporalWorkflowID string    `json:"temporal_workflow_id"`
	TemporalRunID      string    `json:"temporal_run_id"`
}

type TransitionRunStatusInput struct {
	RunID    uuid.UUID        `json:"run_id"`
	ToStatus domain.RunStatus `json:"to_status"`
	Reason   *string          `json:"reason,omitempty"`
}

type TransitionRunAgentStatusInput struct {
	RunAgentID    uuid.UUID             `json:"run_agent_id"`
	ToStatus      domain.RunAgentStatus `json:"to_status"`
	Reason        *string               `json:"reason,omitempty"`
	FailureReason *string               `json:"failure_reason,omitempty"`
}

func NewActivities(repo RunRepository, hooks FakeWorkHooks) *Activities {
	return &Activities{
		repo:  repo,
		hooks: hooks,
	}
}

func (a *Activities) LoadRun(ctx context.Context, input LoadRunInput) (domain.Run, error) {
	run, err := a.repo.GetRunByID(ctx, input.RunID)
	return run, wrapActivityError(err)
}

func (a *Activities) ListRunAgents(ctx context.Context, input ListRunAgentsInput) ([]domain.RunAgent, error) {
	runAgents, err := a.repo.ListRunAgentsByRunID(ctx, input.RunID)
	return runAgents, wrapActivityError(err)
}

func (a *Activities) LoadRunAgent(ctx context.Context, input LoadRunAgentInput) (domain.RunAgent, error) {
	runAgent, err := a.repo.GetRunAgentByID(ctx, input.RunAgentID)
	return runAgent, wrapActivityError(err)
}

func (a *Activities) AttachRunTemporalIDs(ctx context.Context, input AttachRunTemporalIDsInput) (domain.Run, error) {
	run, err := a.repo.SetRunTemporalIDs(ctx, repository.SetRunTemporalIDsParams{
		RunID:              input.RunID,
		TemporalWorkflowID: input.TemporalWorkflowID,
		TemporalRunID:      input.TemporalRunID,
	})
	return run, wrapActivityError(err)
}

func (a *Activities) TransitionRunStatus(ctx context.Context, input TransitionRunStatusInput) (domain.Run, error) {
	run, err := a.repo.TransitionRunStatus(ctx, repository.TransitionRunStatusParams{
		RunID:    input.RunID,
		ToStatus: input.ToStatus,
		Reason:   cloneStringPtr(input.Reason),
	})
	return run, wrapActivityError(err)
}

func (a *Activities) TransitionRunAgentStatus(ctx context.Context, input TransitionRunAgentStatusInput) (domain.RunAgent, error) {
	runAgent, err := a.repo.TransitionRunAgentStatus(ctx, repository.TransitionRunAgentStatusParams{
		RunAgentID:    input.RunAgentID,
		ToStatus:      input.ToStatus,
		Reason:        cloneStringPtr(input.Reason),
		FailureReason: cloneStringPtr(input.FailureReason),
	})
	return runAgent, wrapActivityError(err)
}

func (a *Activities) PrepareExecutionLane(ctx context.Context, input RunAgentWorkflowInput) error {
	return invokeHook(ctx, input, a.hooks.PrepareExecutionLane)
}

func (a *Activities) SimulateExecution(ctx context.Context, input RunAgentWorkflowInput) error {
	return invokeHook(ctx, input, a.hooks.SimulateExecution)
}

func (a *Activities) SimulateEvaluation(ctx context.Context, input RunAgentWorkflowInput) error {
	return invokeHook(ctx, input, a.hooks.SimulateEvaluation)
}

func invokeHook(ctx context.Context, input RunAgentWorkflowInput, hook func(context.Context, RunAgentWorkflowInput) error) error {
	if hook == nil {
		return nil
	}

	return hook(ctx, input)
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func wrapActivityError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, repository.ErrRunNotFound):
		return temporal.NewNonRetryableApplicationError(err.Error(), repositoryRunNotFoundErrorType, err)
	case errors.Is(err, repository.ErrRunAgentNotFound):
		return temporal.NewNonRetryableApplicationError(err.Error(), repositoryRunAgentNotFoundErrorType, err)
	case errors.Is(err, repository.ErrTemporalIDConflict):
		return temporal.NewNonRetryableApplicationError(err.Error(), repositoryTemporalIDConflictType, err)
	case errors.Is(err, repository.ErrInvalidTransition):
		return temporal.NewNonRetryableApplicationError(err.Error(), repositoryInvalidTransitionType, err)
	case errors.Is(err, repository.ErrTransitionConflict):
		return temporal.NewNonRetryableApplicationError(err.Error(), repositoryTransitionConflictType, err)
	default:
		return err
	}
}
