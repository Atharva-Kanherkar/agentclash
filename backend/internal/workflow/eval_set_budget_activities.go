package workflow

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

// EvalSetBudgetRepository extends eval-set persistence for Fleet 13 spend.
type EvalSetBudgetRepository interface {
	EvalSetRepository
	UpdateEvalSetSpentUSD(ctx context.Context, id uuid.UUID, spent float64) (repository.EvalSet, error)
	SumCaseResultCostByEvalSetID(ctx context.Context, evalSetID uuid.UUID) (float64, error)
}

// WorkspaceRunCounter counts in-flight runs for scheduler-level quotas.
type WorkspaceRunCounter interface {
	CountActiveWorkspaceRuns(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

func (a *Activities) WithEvalSetBudgetRepository(repo EvalSetBudgetRepository) *Activities {
	a.evalSetRepo = repo
	a.evalSetBudgetRepo = repo
	return a
}

func (a *Activities) WithWorkspaceRunCounter(counter WorkspaceRunCounter) *Activities {
	a.workspaceRunCounter = counter
	return a
}

func (a *Activities) CheckEvalSetBudget(ctx context.Context, input CheckEvalSetBudgetInput) (CheckEvalSetBudgetResult, error) {
	if a.evalSetRepo == nil {
		return CheckEvalSetBudgetResult{}, errors.New("eval set repository is not configured")
	}
	set, err := a.evalSetRepo.GetEvalSetByID(ctx, input.EvalSetID)
	if err != nil {
		return CheckEvalSetBudgetResult{}, wrapActivityError(err)
	}
	result := CheckEvalSetBudgetResult{
		Allowed:   true,
		SpentUSD:  set.SpentUSD,
		BudgetUSD: set.BudgetUSD,
	}
	if set.BudgetUSD != nil && *set.BudgetUSD >= 0 && set.SpentUSD >= *set.BudgetUSD {
		result.Allowed = false
	}
	return result, nil
}

func (a *Activities) RefreshEvalSetSpend(ctx context.Context, input RefreshEvalSetSpendInput) (CheckEvalSetBudgetResult, error) {
	repo := a.evalSetBudgetRepo
	if repo == nil {
		return CheckEvalSetBudgetResult{}, errors.New("eval set budget repository is not configured")
	}
	set, err := repo.GetEvalSetByID(ctx, input.EvalSetID)
	if err != nil {
		return CheckEvalSetBudgetResult{}, wrapActivityError(err)
	}
	sum, err := repo.SumCaseResultCostByEvalSetID(ctx, input.EvalSetID)
	if err != nil {
		return CheckEvalSetBudgetResult{}, wrapActivityError(err)
	}
	spent := sum
	switch {
	case spent <= 0 && input.ChargeUSD > 0:
		// Projection empty (pre-aggregate / fake provider): apply incremental charge.
		spent = set.SpentUSD + input.ChargeUSD
	case spent < set.SpentUSD:
		// Never decrease a previously recorded total.
		spent = set.SpentUSD
	}
	updated, err := repo.UpdateEvalSetSpentUSD(ctx, input.EvalSetID, spent)
	if err != nil {
		return CheckEvalSetBudgetResult{}, wrapActivityError(err)
	}
	delta := spent - set.SpentUSD
	if delta > 0 {
		_ = a.RecordEvalSetSpendEvent(ctx, RecordEvalSetSpendEventInput{
			EvalSetID: input.EvalSetID,
			SpentUSD:  spent,
			DeltaUSD:  delta,
		})
	}
	result := CheckEvalSetBudgetResult{
		Allowed:   true,
		SpentUSD:  updated.SpentUSD,
		BudgetUSD: updated.BudgetUSD,
	}
	if updated.BudgetUSD != nil && *updated.BudgetUSD >= 0 && updated.SpentUSD >= *updated.BudgetUSD {
		result.Allowed = false
	}
	return result, nil
}

func (a *Activities) WaitWorkspaceRunCapacity(ctx context.Context, input WaitWorkspaceRunCapacityInput) error {
	if a.workspaceRunCounter == nil {
		return nil
	}
	cap := int(input.MaxConcurrent)
	if cap <= 0 {
		cap = workspaceConcurrentRunsFromEnv()
	}
	if cap <= 0 {
		return nil
	}
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		active, err := a.workspaceRunCounter.CountActiveWorkspaceRuns(ctx, input.WorkspaceID)
		if err != nil {
			return wrapActivityError(err)
		}
		if active < cap {
			return nil
		}
		if time.Now().After(deadline) {
			return wrapActivityError(errors.New("timed out waiting for workspace run capacity"))
		}
		activity.RecordHeartbeat(ctx, active)
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (a *Activities) RecordEvalSetSpendEvent(_ context.Context, input RecordEvalSetSpendEventInput) error {
	slog.Info("eval_set.spend_increment",
		"eval_set_id", input.EvalSetID.String(),
		"spent_usd", input.SpentUSD,
		"delta_usd", input.DeltaUSD,
	)
	return nil
}

func workspaceConcurrentRunsFromEnv() int {
	raw := os.Getenv("FLEET_WORKSPACE_MAX_CONCURRENT_RUNS")
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
