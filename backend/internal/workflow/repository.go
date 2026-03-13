package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrRunMustBeQueued      = errors.New("run must already be queued")
	ErrRunHasNoAgents       = errors.New("run must have at least one run agent")
	ErrRunAgentMustBeQueued = errors.New("run agent must already be queued")
	ErrRunAgentRunMismatch  = errors.New("run agent does not belong to run")
)

type RunRepository interface {
	GetRunByID(ctx context.Context, id uuid.UUID) (domain.Run, error)
	ListRunAgentsByRunID(ctx context.Context, runID uuid.UUID) ([]domain.RunAgent, error)
	GetRunAgentByID(ctx context.Context, id uuid.UUID) (domain.RunAgent, error)
	SetRunTemporalIDs(ctx context.Context, params repository.SetRunTemporalIDsParams) (domain.Run, error)
	TransitionRunStatus(ctx context.Context, params repository.TransitionRunStatusParams) (domain.Run, error)
	TransitionRunAgentStatus(ctx context.Context, params repository.TransitionRunAgentStatusParams) (domain.RunAgent, error)
}

func validateRunQueued(run domain.Run) error {
	if run.Status == domain.RunStatusQueued {
		return nil
	}

	return fmt.Errorf("%w: run %s is %s", ErrRunMustBeQueued, run.ID, run.Status)
}

func validateRunAgentQueued(runAgent domain.RunAgent, runID uuid.UUID) error {
	if runAgent.RunID != runID {
		return fmt.Errorf("%w: run_agent=%s run=%s expected_run=%s", ErrRunAgentRunMismatch, runAgent.ID, runAgent.RunID, runID)
	}
	if runAgent.Status == domain.RunAgentStatusQueued {
		return nil
	}

	return fmt.Errorf("%w: run_agent %s is %s", ErrRunAgentMustBeQueued, runAgent.ID, runAgent.Status)
}
