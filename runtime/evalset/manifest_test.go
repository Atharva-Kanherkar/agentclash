package evalset_test

import (
	"strings"
	"testing"

	"github.com/agentclash/agentclash/runtime/evalset"
)

const sampleYAML = `
schema: evalset/v1
name: nightly-coding-sweep
packs:
  - catalog/code-review
  - my-workspace/refund-recovery@3
agents:
  - deployment: claude-opus-5-default
  - deployment: gpt-5-default
  - deployment: gemini-default
models: []
repeats: 5
seeds: {strategy: auto}
limits:
  max_concurrent_runs: 20
  budget_usd: 50
case_fanout: true
`

func TestExpand_TwoPackThreeAgentFiveRepeat(t *testing.T) {
	m, err := evalset.ParseManifest([]byte(sampleYAML))
	if err != nil {
		t.Fatal(err)
	}
	report, err := m.Expand(evalset.DefaultMaxCombos)
	if err != nil {
		t.Fatal(err)
	}
	if report.Count != 30 {
		t.Fatalf("count = %d, want 30", report.Count)
	}
	if report.Combinations[0].MatrixKey != "catalog/code-review/claude-opus-5-default/1" {
		t.Fatalf("first key = %q", report.Combinations[0].MatrixKey)
	}
	last := report.Combinations[len(report.Combinations)-1]
	if last.MatrixKey != "my-workspace/refund-recovery@3/gemini-default/5" {
		t.Fatalf("last key = %q", last.MatrixKey)
	}
	// Determinism
	report2, err := m.Expand(evalset.DefaultMaxCombos)
	if err != nil {
		t.Fatal(err)
	}
	for i := range report.Combinations {
		if report.Combinations[i].MatrixKey != report2.Combinations[i].MatrixKey {
			t.Fatalf("non-deterministic at %d", i)
		}
	}
}

func TestExpand_RejectsOverCap(t *testing.T) {
	m := evalset.Manifest{
		Schema:  evalset.SchemaV1,
		Name:    "big",
		Packs:   []string{"a", "b"},
		Agents:  []evalset.AgentEntry{{Deployment: "x"}, {Deployment: "y"}},
		Repeats: 1000,
	}
	_, err := m.Expand(100)
	if err == nil || !strings.Contains(err.Error(), "exceeds max") {
		t.Fatalf("err = %v", err)
	}
}

func TestExpand_RejectsDuplicateSeeds(t *testing.T) {
	m := evalset.Manifest{
		Schema:  evalset.SchemaV1,
		Name:    "seeds",
		Packs:   []string{"p"},
		Agents:  []evalset.AgentEntry{{Deployment: "a"}},
		Repeats: 2,
		Seeds:   &evalset.SeedConfig{Strategy: "explicit", Seeds: []int64{1, 1}},
	}
	_, err := m.Expand(evalset.DefaultMaxCombos)
	if err == nil || !strings.Contains(err.Error(), "duplicate seed") {
		t.Fatalf("err = %v", err)
	}
}

func TestExpand_WithModelsAxis(t *testing.T) {
	m := evalset.Manifest{
		Schema:  evalset.SchemaV1,
		Name:    "models",
		Packs:   []string{"p"},
		Agents:  []evalset.AgentEntry{{Deployment: "harness"}},
		Models:  []string{"m1", "m2"},
		Repeats: 2,
	}
	report, err := m.Expand(evalset.DefaultMaxCombos)
	if err != nil {
		t.Fatal(err)
	}
	if report.Count != 4 {
		t.Fatalf("count = %d", report.Count)
	}
	if report.Combinations[0].MatrixKey != "p/harness/m1/1" {
		t.Fatalf("key = %q", report.Combinations[0].MatrixKey)
	}
}

func TestExpand_RejectsMaxAllowedCombosCeiling(t *testing.T) {
	m := evalset.Manifest{
		Schema:  evalset.SchemaV1,
		Name:    "cap",
		Packs:   []string{"p"},
		Agents:  []evalset.AgentEntry{{Deployment: "a"}},
		Repeats: 1,
	}
	_, err := m.Expand(evalset.MaxAllowedCombos + 1)
	if err == nil || !strings.Contains(err.Error(), "server limit") {
		t.Fatalf("err = %v", err)
	}
}

func TestExpand_RejectsDuplicateMatrixKey(t *testing.T) {
	m := evalset.Manifest{
		Schema: evalset.SchemaV1,
		Name:   "dup-labels",
		Packs:  []string{"p"},
		Agents: []evalset.AgentEntry{
			{Deployment: "dep-a", Label: "my agent"},
			{Deployment: "dep-b", Label: "my-agent"},
		},
		Repeats: 1,
	}
	_, err := m.Expand(evalset.DefaultMaxCombos)
	if err == nil || !strings.Contains(err.Error(), "duplicate matrix_key") {
		t.Fatalf("err = %v", err)
	}
}
