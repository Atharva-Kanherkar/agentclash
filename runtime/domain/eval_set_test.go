package domain_test

import (
	"testing"

	"github.com/agentclash/agentclash/runtime/domain"
)

func TestEvalSetStatusTransitions(t *testing.T) {
	cases := []struct {
		from domain.EvalSetStatus
		to   domain.EvalSetStatus
		ok   bool
	}{
		{domain.EvalSetStatusQueued, domain.EvalSetStatusExpanding, true},
		{domain.EvalSetStatusQueued, domain.EvalSetStatusFailed, true},
		{domain.EvalSetStatusQueued, domain.EvalSetStatusCancelled, true},
		{domain.EvalSetStatusQueued, domain.EvalSetStatusRunning, false},
		{domain.EvalSetStatusExpanding, domain.EvalSetStatusRunning, true},
		{domain.EvalSetStatusRunning, domain.EvalSetStatusAggregating, true},
		{domain.EvalSetStatusAggregating, domain.EvalSetStatusCompleted, true},
		{domain.EvalSetStatusCompleted, domain.EvalSetStatusRunning, false},
		{domain.EvalSetStatusFailed, domain.EvalSetStatusQueued, false},
		{domain.EvalSetStatusCancelled, domain.EvalSetStatusRunning, false},
	}
	for _, tc := range cases {
		if got := tc.from.CanTransitionTo(tc.to); got != tc.ok {
			t.Fatalf("%s → %s = %v, want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}
