package domain

import "fmt"

// EvalSetStatus is the lifecycle of a multi-pack eval set (Fleet 7 / Fleet 13).
type EvalSetStatus string

const (
	EvalSetStatusQueued         EvalSetStatus = "queued"
	EvalSetStatusExpanding      EvalSetStatus = "expanding"
	EvalSetStatusRunning        EvalSetStatus = "running"
	EvalSetStatusAggregating    EvalSetStatus = "aggregating"
	EvalSetStatusCompleted      EvalSetStatus = "completed"
	EvalSetStatusFailed         EvalSetStatus = "failed"
	EvalSetStatusCancelled      EvalSetStatus = "cancelled"
	EvalSetStatusBudgetExceeded EvalSetStatus = "budget_exceeded"
)

var evalSetTransitions = map[EvalSetStatus]map[EvalSetStatus]struct{}{
	EvalSetStatusQueued: {
		EvalSetStatusExpanding: {},
		EvalSetStatusFailed:    {},
		EvalSetStatusCancelled: {},
	},
	EvalSetStatusExpanding: {
		EvalSetStatusRunning:   {},
		EvalSetStatusFailed:    {},
		EvalSetStatusCancelled: {},
	},
	EvalSetStatusRunning: {
		EvalSetStatusAggregating:    {},
		EvalSetStatusFailed:         {},
		EvalSetStatusCancelled:      {},
		EvalSetStatusBudgetExceeded: {},
	},
	EvalSetStatusAggregating: {
		EvalSetStatusCompleted:      {},
		EvalSetStatusFailed:         {},
		EvalSetStatusCancelled:      {},
		EvalSetStatusBudgetExceeded: {},
	},
}

// CanTransitionTo reports whether from→to is a legal eval-set transition.
func (s EvalSetStatus) CanTransitionTo(to EvalSetStatus) bool {
	allowed, ok := evalSetTransitions[s]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

// ValidateEvalSetStatus ensures the status string is known.
func ValidateEvalSetStatus(s EvalSetStatus) error {
	switch s {
	case EvalSetStatusQueued, EvalSetStatusExpanding, EvalSetStatusRunning,
		EvalSetStatusAggregating, EvalSetStatusCompleted, EvalSetStatusFailed,
		EvalSetStatusCancelled, EvalSetStatusBudgetExceeded:
		return nil
	default:
		return fmt.Errorf("invalid eval set status %q", s)
	}
}

// IsEvalSetTerminal reports whether the set will not launch further work.
func IsEvalSetTerminal(s EvalSetStatus) bool {
	switch s {
	case EvalSetStatusCompleted, EvalSetStatusFailed, EvalSetStatusCancelled, EvalSetStatusBudgetExceeded:
		return true
	default:
		return false
	}
}
