package vibe

import (
	"context"
	"errors"
	"github.com/agentclash/agentclash/runtime/scoring"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"testing"
)

func TestPaidTemporalActivityNeverRetries(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	paid, finalized := 0, 0
	env.RegisterActivityWithOptions(func(context.Context, string) error {
		paid++
		return errors.New("worker lost acknowledgement after paid response")
	}, activity.RegisterOptions{Name: "vibe.execute"})
	env.RegisterActivityWithOptions(func(context.Context, string, string) error { finalized++; return nil }, activity.RegisterOptions{Name: "vibe.finalize"})
	env.ExecuteWorkflow(OperationWorkflow, "durable-operation-id")
	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil || paid != 1 || finalized != 1 {
		t.Fatalf("paid=%d finalize=%d error=%v", paid, finalized, env.GetWorkflowError())
	}
}
func TestInvalidJudgeIsUnknown(t *testing.T) {
	for _, b := range []string{`{}`, `{"reasoning":"fine"}`, `{"pass":"yes","reasoning":"x"}`, `{"pass":true,"reasoning":"x","tool":"spend"}`} {
		r, err := ParseJudge(scoring.LLMJudgeDeclaration{Mode: scoring.JudgeMethodAssertion}, []byte(b), LimitsFor(true))
		if err == nil || r.Verdict != Unknown {
			t.Fatalf("%s became %s", b, r.Verdict)
		}
	}
}
