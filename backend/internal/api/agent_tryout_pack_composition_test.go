package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/challengepack"
	"github.com/agentclash/agentclash/runtime/scoring"
	"github.com/google/uuid"
)

// tryoutFixture builds a completed tryout with the snapshot shapes the real
// templates produce, then lets each case vary one dimension.
func tryoutFixture(mutate func(*repository.AgentTryout)) repository.AgentTryout {
	orgID, workspaceID := uuid.New(), uuid.New()
	tryout := repository.AgentTryout{
		ID:              uuid.MustParse("11112222-3333-4444-5555-666677778888"),
		OrganizationID:  &orgID,
		WorkspaceID:     &workspaceID,
		TemplateSlug:    "meeting-minutes",
		Status:          repository.AgentTryoutStatusCompleted,
		RedactionStatus: repository.AgentTryoutRedactionPassed,
		InputSnapshot:   json.RawMessage(`{"notes":"standup notes","audience":"execs"}`),
		TemplateSnapshot: json.RawMessage(`{
			"slug":"meeting-minutes",
			"name":"Meeting Minutes to Action Plan",
			"description":"Turn notes into minutes.",
			"runtime":{
				"instructions":"Write action-plan.md and minutes.json.",
				"expected_artifacts":[
					{"key":"action_plan","type":"markdown","path":"action-plan.md"},
					{"key":"structured_minutes","type":"json","path":"minutes.json"}
				]
			}
		}`),
		ToolPolicySnapshot: json.RawMessage(`{"tools":["file_writer"],"sandbox":{"filesystem":"workspace","shell":"disabled"}}`),
		EvaluationSpecSnapshot: json.RawMessage(`{
			"validators":[{"key":"has_summary","type":"json_field","field":"summary"}],
			"scorecard":{"dimensions":["correctness","reliability","latency","cost"]},
			"judge_mode":"hybrid",
			"llm_judges":[{
				"key":"overall_quality","mode":"rubric","model":"gpt-5-mini",
				"rubric":"Score 1-5.","context_from":["final_output"],
				"samples":1,"timeout_ms":45000,"score_scale":{"min":1,"max":5}
			}],
			"judge_meta":{"model":"gpt-5-mini","strictness":"standard"}
		}`),
		CostLimitUSD:       10,
		MaxDurationSeconds: 120,
	}
	if mutate != nil {
		mutate(&tryout)
	}
	return tryout
}

func dimensionKeys(spec scoring.EvaluationSpec) []string {
	keys := make([]string, 0, len(spec.Scorecard.Dimensions))
	for _, dim := range spec.Scorecard.Dimensions {
		keys = append(keys, dim.Key)
	}
	return keys
}

func validatorKeys(spec scoring.EvaluationSpec) []string {
	keys := make([]string, 0, len(spec.Validators))
	for _, validator := range spec.Validators {
		keys = append(keys, validator.Key)
	}
	return keys
}

func TestAgentTryoutPackBundleMapping(t *testing.T) {
	tests := []struct {
		name   string
		tryout repository.AgentTryout
		assert func(t *testing.T, bundle challengepack.Bundle)
	}{
		{
			name:   "expected artifacts become file captures and file_exists validators",
			tryout: tryoutFixture(nil),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				spec := bundle.Version.EvaluationSpec
				wantValidators := []string{"produced_action_plan", "produced_structured_minutes"}
				if got := validatorKeys(spec); !equalStrings(got, wantValidators) {
					t.Fatalf("validators = %v, want %v", got, wantValidators)
				}
				for _, validator := range spec.Validators {
					if validator.Type != scoring.ValidatorTypeFileExists {
						t.Fatalf("validator %s type = %q, want file_exists", validator.Key, validator.Type)
					}
					if !strings.HasPrefix(validator.Target, "file:") {
						t.Fatalf("validator %s target = %q, want a file: reference", validator.Key, validator.Target)
					}
				}
				if len(spec.PostExecutionChecks) != 2 {
					t.Fatalf("post execution checks = %d, want 2", len(spec.PostExecutionChecks))
				}
				if spec.PostExecutionChecks[0].Path != "action-plan.md" {
					t.Fatalf("first check path = %q, want action-plan.md", spec.PostExecutionChecks[0].Path)
				}
			},
		},
		{
			name:   "judges carry over verbatim and each gets a dimension",
			tryout: tryoutFixture(nil),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				spec := bundle.Version.EvaluationSpec
				if len(spec.LLMJudges) != 1 || spec.LLMJudges[0].Key != "overall_quality" {
					t.Fatalf("llm judges = %+v, want the tryout's overall_quality judge", spec.LLMJudges)
				}
				if spec.LLMJudges[0].Rubric != "Score 1-5." || spec.LLMJudges[0].Model != "gpt-5-mini" {
					t.Fatalf("judge was not carried verbatim: %+v", spec.LLMJudges[0])
				}
				want := []string{"correctness", "reliability", "latency", "cost", "overall_quality"}
				if got := dimensionKeys(spec); !equalStrings(got, want) {
					t.Fatalf("dimensions = %v, want %v", got, want)
				}
			},
		},
		{
			name: "unmappable template dimension labels are dropped",
			tryout: tryoutFixture(func(tryout *repository.AgentTryout) {
				tryout.EvaluationSpecSnapshot = json.RawMessage(`{"scorecard":{"dimensions":["accuracy","visuals","cost"]}}`)
			}),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				want := []string{"correctness", "cost"}
				if got := dimensionKeys(bundle.Version.EvaluationSpec); !equalStrings(got, want) {
					t.Fatalf("dimensions = %v, want %v (accuracy/visuals have no scoring source)", got, want)
				}
			},
		},
		{
			name: "a template with no expected artifacts still declares a validator",
			tryout: tryoutFixture(func(tryout *repository.AgentTryout) {
				tryout.TemplateSnapshot = json.RawMessage(`{"slug":"prompt-only","name":"Prompt only"}`)
			}),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				spec := bundle.Version.EvaluationSpec
				if got := validatorKeys(spec); !equalStrings(got, []string{"final_output_not_empty"}) {
					t.Fatalf("validators = %v, want the non-empty-output fallback", got)
				}
				if len(spec.PostExecutionChecks) != 0 {
					t.Fatalf("post execution checks = %d, want 0", len(spec.PostExecutionChecks))
				}
			},
		},
		{
			name: "a zero budget drops the latency and cost dimensions",
			tryout: tryoutFixture(func(tryout *repository.AgentTryout) {
				tryout.CostLimitUSD = 0
				tryout.MaxDurationSeconds = 0
			}),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				want := []string{"correctness", "reliability", "overall_quality"}
				if got := dimensionKeys(bundle.Version.EvaluationSpec); !equalStrings(got, want) {
					t.Fatalf("dimensions = %v, want %v", got, want)
				}
				if bundle.Version.EvaluationSpec.RuntimeLimits.MaxCostUSD != nil {
					t.Fatalf("max_cost_usd must stay unset when the tryout had no cost cap")
				}
			},
		},
		{
			name:   "the tryout input becomes the single case payload",
			tryout: tryoutFixture(nil),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				if len(bundle.InputSets) != 1 || len(bundle.InputSets[0].Cases) != 1 {
					t.Fatalf("input sets = %+v, want exactly one case", bundle.InputSets)
				}
				payload := bundle.InputSets[0].Cases[0].Payload
				if payload["notes"] != "standup notes" {
					t.Fatalf("case payload = %+v, want the tryout input snapshot", payload)
				}
				if bundle.InputSets[0].Cases[0].ChallengeKey != bundle.Challenges[0].Key {
					t.Fatalf("case references %q but the challenge key is %q", bundle.InputSets[0].Cases[0].ChallengeKey, bundle.Challenges[0].Key)
				}
			},
		},
		{
			name:   "the tool policy snapshot is carried verbatim",
			tryout: tryoutFixture(nil),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				tools, ok := bundle.Version.ToolPolicy["tools"].([]any)
				if !ok || len(tools) != 1 || tools[0] != "file_writer" {
					t.Fatalf("tool policy = %+v, want the tryout's snapshot", bundle.Version.ToolPolicy)
				}
			},
		},
		{
			name: "empty snapshots degrade to a thin but complete pack",
			tryout: tryoutFixture(func(tryout *repository.AgentTryout) {
				tryout.TemplateSnapshot = nil
				tryout.EvaluationSpecSnapshot = nil
				tryout.ToolPolicySnapshot = nil
				tryout.InputSnapshot = nil
				tryout.TemplateSlug = ""
			}),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				if bundle.Challenges[0].Key != "tryout" {
					t.Fatalf("challenge key = %q, want the tryout fallback", bundle.Challenges[0].Key)
				}
				if bundle.Pack.Name == "" || bundle.Pack.Family == "" || bundle.Pack.Slug == "" {
					t.Fatalf("pack metadata is incomplete: %+v", bundle.Pack)
				}
			},
		},
		{
			name: "malformed snapshots do not abort the mapping",
			tryout: tryoutFixture(func(tryout *repository.AgentTryout) {
				tryout.TemplateSnapshot = json.RawMessage(`not json`)
				tryout.EvaluationSpecSnapshot = json.RawMessage(`[1,2,3]`)
				tryout.ToolPolicySnapshot = json.RawMessage(`"a string"`)
				tryout.InputSnapshot = json.RawMessage(`[]`)
			}),
			assert: func(t *testing.T, bundle challengepack.Bundle) {
				if bundle.Version.ToolPolicy != nil {
					t.Fatalf("tool policy = %+v, want nil for a non-object snapshot", bundle.Version.ToolPolicy)
				}
				if bundle.InputSets[0].Cases[0].Payload != nil {
					t.Fatalf("case payload = %+v, want nil for a non-object snapshot", bundle.InputSets[0].Cases[0].Payload)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundle := agentTryoutPackBundle(tc.tryout)
			tc.assert(t, bundle)
		})
	}
}

// TestAgentTryoutPackBundleAlwaysCompiles is the contract that makes promotion
// safe: whatever the tryout carried, the pack it produces must survive
// ComposeBundle -> ValidateBundle -> BundleYAML -> ParseYAML, which is exactly
// the builder's compile and publish path.
func TestAgentTryoutPackBundleAlwaysCompiles(t *testing.T) {
	tests := []struct {
		name   string
		tryout repository.AgentTryout
	}{
		{"full snapshots", tryoutFixture(nil)},
		{"no artifacts", tryoutFixture(func(tr *repository.AgentTryout) {
			tr.TemplateSnapshot = json.RawMessage(`{"slug":"prompt-only","name":"Prompt only"}`)
		})},
		{"no judges", tryoutFixture(func(tr *repository.AgentTryout) {
			tr.EvaluationSpecSnapshot = json.RawMessage(`{"validators":[],"scorecard":{"dimensions":["correctness"]}}`)
		})},
		{"no budget", tryoutFixture(func(tr *repository.AgentTryout) {
			tr.CostLimitUSD = 0
			tr.MaxDurationSeconds = 0
		})},
		{"everything empty", tryoutFixture(func(tr *repository.AgentTryout) {
			*tr = repository.AgentTryout{ID: tr.ID}
		})},
		{"free-text dimensions only", tryoutFixture(func(tr *repository.AgentTryout) {
			tr.EvaluationSpecSnapshot = json.RawMessage(`{"scorecard":{"dimensions":["accuracy","tone"]}}`)
		})},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, composition, err := agentTryoutPackDraft(tc.tryout)
			if err != nil {
				t.Fatalf("agentTryoutPackDraft: %v", err)
			}
			if strings.TrimSpace(name) == "" {
				t.Fatal("draft name must never be empty — CreateDraft rejects it")
			}

			var decoded challengepack.Composition
			if err := json.Unmarshal(composition, &decoded); err != nil {
				t.Fatalf("composition does not decode: %v", err)
			}
			bundle, err := challengepack.ComposeBundle(decoded, challengepack.ResolvedPieces{})
			if err != nil {
				t.Fatalf("ComposeBundle: %v", err)
			}
			if err := challengepack.ValidateBundle(bundle); err != nil {
				t.Fatalf("ValidateBundle: %v", err)
			}
			yamlBytes, err := challengepack.BundleYAML(bundle)
			if err != nil {
				t.Fatalf("BundleYAML: %v", err)
			}
			if _, err := challengepack.ParseYAML(yamlBytes); err != nil {
				t.Fatalf("ParseYAML (publish path): %v", err)
			}
		})
	}
}

func TestAgentTryoutPackSlugIsUniquePerTryout(t *testing.T) {
	first := tryoutFixture(nil)
	second := tryoutFixture(func(tryout *repository.AgentTryout) { tryout.ID = uuid.New() })

	firstSlug := agentTryoutPackBundle(first).Pack.Slug
	secondSlug := agentTryoutPackBundle(second).Pack.Slug
	if firstSlug == secondSlug {
		t.Fatalf("two promotions of the same template share slug %q; publishing the second would collide", firstSlug)
	}
	if !strings.HasPrefix(firstSlug, "tryout-meeting-minutes-") {
		t.Fatalf("slug = %q, want a tryout-<template>-<id> slug", firstSlug)
	}
}

func TestTryoutSlugify(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		fallback string
		want     string
	}{
		{"already a slug", "meeting-minutes", "x", "meeting-minutes"},
		{"underscores are kept for validator keys", "action_plan", "x", "action_plan"},
		{"spaces and case collapse", "  Slide Deck  ", "x", "slide-deck"},
		{"punctuation collapses to one dash", "a//b??c", "x", "a-b-c"},
		{"empty falls back", "  ", "tryout", "tryout"},
		{"punctuation only falls back", "!!!", "tryout", "tryout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tryoutSlugify(tc.in, tc.fallback); got != tc.want {
				t.Fatalf("tryoutSlugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
