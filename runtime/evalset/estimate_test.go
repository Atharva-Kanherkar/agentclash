package evalset_test

import (
	"testing"

	"github.com/agentclash/agentclash/runtime/evalset"
)

func TestEstimateCostExceedsBudget(t *testing.T) {
	report := evalset.ExpansionReport{Count: 20}
	budget := 0.01
	est := evalset.EstimateCost(report, &budget, []string{"anthropic/claude-haiku-4"})
	if est.EstimatedUSD <= 0 {
		t.Fatalf("expected positive estimate, got %#v", est)
	}
	if !est.ExceedsBudget {
		t.Fatalf("expected exceeds_budget for tiny budget, got %#v", est)
	}
	if est.TolerancePct != 50 {
		t.Fatalf("tolerance = %d", est.TolerancePct)
	}
}

func TestEstimateCostWithinBudget(t *testing.T) {
	report := evalset.ExpansionReport{Count: 1}
	budget := 1000.0
	est := evalset.EstimateCost(report, &budget, nil)
	if est.ExceedsBudget {
		t.Fatalf("unexpected exceeds: %#v", est)
	}
}
