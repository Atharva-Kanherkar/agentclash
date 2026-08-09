package workflow

import (
	"errors"

	"github.com/google/uuid"
)

const (
	evalSetBudgetEnforcementVersionChangeID = "eval-set-budget-enforcement"
	checkEvalSetBudgetActivityName          = "workflow.check_eval_set_budget"
	refreshEvalSetSpendActivityName         = "workflow.refresh_eval_set_spend"
	waitWorkspaceRunCapacityActivityName    = "workflow.wait_workspace_run_capacity"
	recordEvalSetSpendEventActivityName     = "workflow.record_eval_set_spend_event"
)

// ErrEvalSetBudgetExceeded is returned from the bounded launch gate when
// spent_usd meets or exceeds budget_usd. Partial results are kept.
var ErrEvalSetBudgetExceeded = errors.New("eval set budget exceeded")

type CheckEvalSetBudgetInput struct {
	EvalSetID uuid.UUID `json:"eval_set_id"`
}

type CheckEvalSetBudgetResult struct {
	Allowed   bool     `json:"allowed"`
	SpentUSD  float64  `json:"spent_usd"`
	BudgetUSD *float64 `json:"budget_usd,omitempty"`
}

type RefreshEvalSetSpendInput struct {
	EvalSetID uuid.UUID `json:"eval_set_id"`
	// ChargeUSD is an optional incremental charge applied when projection
	// costs are still zero (test harness / early runs before scoring).
	ChargeUSD float64 `json:"charge_usd,omitempty"`
}

type WaitWorkspaceRunCapacityInput struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	// MaxConcurrent is the workspace launch cap (0 → env/default).
	MaxConcurrent int32 `json:"max_concurrent"`
}

type RecordEvalSetSpendEventInput struct {
	EvalSetID uuid.UUID `json:"eval_set_id"`
	SpentUSD  float64   `json:"spent_usd"`
	DeltaUSD  float64   `json:"delta_usd"`
}
