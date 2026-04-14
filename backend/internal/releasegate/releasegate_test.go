package releasegate

import (
	"encoding/json"
	"testing"
)

func TestEvaluatePassesWhenThresholdsHold(t *testing.T) {
	evaluation, err := Evaluate(testSummary(t, `{
		"status":"comparable",
		"dimension_deltas":{
			"correctness":{"delta":-0.01,"better_direction":"higher","state":"available"},
			"reliability":{"delta":-0.01,"better_direction":"higher","state":"available"},
			"latency":{"delta":0.02,"better_direction":"lower","state":"available"},
			"cost":{"delta":0.03,"better_direction":"lower","state":"available"}
		},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`), DefaultPolicy())
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.Verdict != VerdictPass {
		t.Fatalf("verdict = %q, want %q", evaluation.Verdict, VerdictPass)
	}
	if evaluation.ReasonCode != "within_thresholds" {
		t.Fatalf("reason code = %q, want within_thresholds", evaluation.ReasonCode)
	}
}

func TestEvaluateWarnsWhenWarnThresholdCrossed(t *testing.T) {
	evaluation, err := Evaluate(testSummary(t, `{
		"status":"comparable",
		"dimension_deltas":{
			"correctness":{"delta":-0.03,"better_direction":"higher","state":"available"},
			"reliability":{"delta":0,"better_direction":"higher","state":"available"},
			"latency":{"delta":0,"better_direction":"lower","state":"available"},
			"cost":{"delta":0,"better_direction":"lower","state":"available"}
		},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`), DefaultPolicy())
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.Verdict != VerdictWarn {
		t.Fatalf("verdict = %q, want %q", evaluation.Verdict, VerdictWarn)
	}
	if evaluation.ReasonCode != "threshold_warn_correctness" {
		t.Fatalf("reason code = %q, want threshold_warn_correctness", evaluation.ReasonCode)
	}
}

func TestEvaluateFailsWhenFailThresholdCrossed(t *testing.T) {
	evaluation, err := Evaluate(testSummary(t, `{
		"status":"comparable",
		"dimension_deltas":{
			"correctness":{"delta":-0.06,"better_direction":"higher","state":"available"},
			"reliability":{"delta":0,"better_direction":"higher","state":"available"},
			"latency":{"delta":0,"better_direction":"lower","state":"available"},
			"cost":{"delta":0,"better_direction":"lower","state":"available"}
		},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`), DefaultPolicy())
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.Verdict != VerdictFail {
		t.Fatalf("verdict = %q, want %q", evaluation.Verdict, VerdictFail)
	}
	if evaluation.ReasonCode != "threshold_fail_correctness" {
		t.Fatalf("reason code = %q, want threshold_fail_correctness", evaluation.ReasonCode)
	}
}

func TestEvaluateReturnsInsufficientEvidenceWhenMissingFieldsPresent(t *testing.T) {
	evaluation, err := Evaluate(testSummary(t, `{
		"status":"comparable",
		"dimension_deltas":{
			"correctness":{"delta":0,"better_direction":"higher","state":"available"},
			"reliability":{"delta":0,"better_direction":"higher","state":"available"},
			"latency":{"delta":0,"better_direction":"lower","state":"available"},
			"cost":{"delta":0,"better_direction":"lower","state":"available"}
		},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"unavailable"},
		"evidence_quality":{"missing_fields":["replay_summary_divergence"]}
	}`), DefaultPolicy())
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.Verdict != VerdictInsufficientEvidence {
		t.Fatalf("verdict = %q, want %q", evaluation.Verdict, VerdictInsufficientEvidence)
	}
	if evaluation.ReasonCode != "comparison_evidence_missing" {
		t.Fatalf("reason code = %q, want comparison_evidence_missing", evaluation.ReasonCode)
	}
}

func TestEvaluateReturnsInsufficientEvidenceWhenComparisonNotComparable(t *testing.T) {
	evaluation, err := Evaluate(testSummary(t, `{
		"status":"not_comparable",
		"reason_code":"missing_scorecard",
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"unavailable"},
		"evidence_quality":{}
	}`), DefaultPolicy())
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.Verdict != VerdictInsufficientEvidence {
		t.Fatalf("verdict = %q, want %q", evaluation.Verdict, VerdictInsufficientEvidence)
	}
	if evaluation.ReasonCode != "comparison_not_comparable" {
		t.Fatalf("reason code = %q, want comparison_not_comparable", evaluation.ReasonCode)
	}
}

func TestPolicySnapshotFingerprintStable(t *testing.T) {
	policy := DefaultPolicy()
	firstJSON, firstFingerprint, err := PolicySnapshot(policy)
	if err != nil {
		t.Fatalf("first PolicySnapshot returned error: %v", err)
	}
	secondJSON, secondFingerprint, err := PolicySnapshot(policy)
	if err != nil {
		t.Fatalf("second PolicySnapshot returned error: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("snapshot mismatch:\n%s\n%s", firstJSON, secondJSON)
	}
	if firstFingerprint != secondFingerprint {
		t.Fatalf("fingerprint = %q, want %q", firstFingerprint, secondFingerprint)
	}
}

// Phase 4: the evaluator must treat user-declared dims exactly like built-ins.
// A policy that requires a custom "safety" dim with a threshold should pass
// when the candidate improves against the declared direction.
func TestEvaluateAppliesCustomDimensionThreshold(t *testing.T) {
	warn := 0.05
	fail := 0.10
	policy := Policy{
		PolicyKey:          "safety-gate",
		PolicyVersion:      1,
		RequireComparable:  true,
		RequiredDimensions: []string{"safety"},
		Dimensions: map[string]DimensionThreshold{
			"safety": {WarnDelta: &warn, FailDelta: &fail},
		},
	}
	evaluation, err := Evaluate(testSummary(t, `{
		"status":"comparable",
		"dimension_deltas":{
			"safety":{"delta":0.02,"better_direction":"higher","state":"available"}
		},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`), policy)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.Verdict != VerdictPass {
		t.Fatalf("verdict = %q, want %q", evaluation.Verdict, VerdictPass)
	}
	result, ok := evaluation.Details.DimensionResults["safety"]
	if !ok {
		t.Fatalf("dimension_results missing safety key")
	}
	if result.BetterDirection != "higher" {
		t.Fatalf("safety better_direction = %q, want higher", result.BetterDirection)
	}
}

// Phase 4: when multiple required dims are unavailable, Evaluate must
// accumulate ALL of them into triggered_conditions instead of short-circuiting
// on the first miss. Operators reading the gate result should see every
// missing dim so they can fix the spec in one pass.
func TestEvaluateAccumulatesAllMissingRequiredDimensions(t *testing.T) {
	warn := 0.05
	fail := 0.10
	threshold := DimensionThreshold{WarnDelta: &warn, FailDelta: &fail}
	policy := Policy{
		PolicyKey:          "multi-required",
		PolicyVersion:      1,
		RequireComparable:  true,
		RequiredDimensions: []string{"alpha", "beta", "gamma"},
		Dimensions: map[string]DimensionThreshold{
			"alpha": threshold,
			"beta":  threshold,
			"gamma": threshold,
		},
	}
	evaluation, err := Evaluate(testSummary(t, `{
		"status":"comparable",
		"dimension_deltas":{
			"gamma":{"delta":0.01,"better_direction":"higher","state":"available"}
		},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`), policy)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.Verdict != VerdictInsufficientEvidence {
		t.Fatalf("verdict = %q, want %q", evaluation.Verdict, VerdictInsufficientEvidence)
	}
	if evaluation.ReasonCode != "required_dimension_unavailable" {
		t.Fatalf("reason code = %q", evaluation.ReasonCode)
	}
	triggers := evaluation.Details.TriggeredConditions
	if len(triggers) != 2 {
		t.Fatalf("triggered_conditions = %v, want 2 entries (alpha + beta)", triggers)
	}
	want := map[string]bool{
		"required_dimension_unavailable:alpha": true,
		"required_dimension_unavailable:beta":  true,
	}
	for _, trigger := range triggers {
		if !want[trigger] {
			t.Fatalf("unexpected trigger %q; got %v", trigger, triggers)
		}
	}
}

// Phase 5: when the policy requires a scorecard pass, Evaluate must route
// on the candidate's verdict regardless of dimension thresholds. These
// three cases cover the happy path (candidate passed → existing threshold
// logic still runs), explicit fail (candidate_failed → fail fast), and
// unknown verdict (legacy row → insufficient_evidence).
func TestEvaluateRequiresScorecardPass(t *testing.T) {
	warn := 0.05
	fail := 0.10
	threshold := DimensionThreshold{WarnDelta: &warn, FailDelta: &fail}
	policyWithRequire := func() Policy {
		return Policy{
			PolicyKey:            "scorecard-gate",
			PolicyVersion:        1,
			RequireComparable:    true,
			RequireScorecardPass: true,
			RequiredDimensions:   []string{"correctness"},
			Dimensions: map[string]DimensionThreshold{
				"correctness": threshold,
			},
		}
	}

	passPayload := `{
		"status":"comparable",
		"dimension_deltas":{
			"correctness":{"delta":0,"better_direction":"higher","state":"available"}
		},
		"scorecard_pass":{"baseline":true,"candidate":true},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`
	failPayload := `{
		"status":"comparable",
		"dimension_deltas":{
			"correctness":{"delta":0,"better_direction":"higher","state":"available"}
		},
		"scorecard_pass":{"baseline":true,"candidate":false},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`
	unknownPayload := `{
		"status":"comparable",
		"dimension_deltas":{
			"correctness":{"delta":0,"better_direction":"higher","state":"available"}
		},
		"scorecard_pass":{"baseline":true},
		"failure_divergence":{"candidate_failed_baseline_succeeded":false,"both_failed_differently":false},
		"replay_summary_divergence":{"state":"available"},
		"evidence_quality":{}
	}`

	tests := []struct {
		name       string
		payload    string
		wantVerdict Verdict
		wantReason string
	}{
		{"candidate_passed", passPayload, VerdictPass, "within_thresholds"},
		{"candidate_failed", failPayload, VerdictFail, "scorecard_not_passed"},
		{"candidate_unknown", unknownPayload, VerdictInsufficientEvidence, "scorecard_pass_unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluation, err := Evaluate(testSummary(t, tt.payload), policyWithRequire())
			if err != nil {
				t.Fatalf("Evaluate returned error: %v", err)
			}
			if evaluation.Verdict != tt.wantVerdict {
				t.Fatalf("verdict = %q, want %q", evaluation.Verdict, tt.wantVerdict)
			}
			if evaluation.ReasonCode != tt.wantReason {
				t.Fatalf("reason code = %q, want %q", evaluation.ReasonCode, tt.wantReason)
			}
		})
	}
}

// Phase 5: DefaultPolicy must NOT require scorecard pass — turning the flag
// on by default would break every existing release gate whose summary was
// written before Phase 5 (no scorecard_pass object). The flag is opt-in.
func TestDefaultPolicyDoesNotRequireScorecardPass(t *testing.T) {
	if DefaultPolicy().RequireScorecardPass {
		t.Fatalf("DefaultPolicy.RequireScorecardPass = true, want false")
	}
}

func testSummary(t *testing.T, payload string) ComparisonSummary {
	t.Helper()

	var summary ComparisonSummary
	if err := json.Unmarshal([]byte(payload), &summary); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	return summary
}
