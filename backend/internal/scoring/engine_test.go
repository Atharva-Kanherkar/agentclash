package scoring

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEvaluateRunAgentCompletesWithDeterministicEvidence(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "exact",
				Type:         ValidatorTypeExactMatch,
				Target:       "final_output",
				ExpectedFrom: "challenge_input",
			},
		},
		Metrics: []MetricDeclaration{
			{Key: "completed", Type: MetricTypeBoolean, Collector: "run_completed_successfully"},
			{Key: "failures", Type: MetricTypeNumeric, Collector: "run_failure_count"},
			{Key: "tokens", Type: MetricTypeNumeric, Collector: "run_total_tokens"},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness, ScorecardDimensionReliability},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		ChallengeInputs: []EvidenceInput{
			{
				ChallengeIdentityID: uuid.New(),
				ItemKey:             "expected.txt",
				Payload:             []byte(`"done"`),
			},
		},
		Events: []Event{
			{Type: "system.run.started", OccurredAt: time.Date(2026, 3, 16, 9, 0, 0, 0, time.UTC), Payload: []byte(`{}`)},
			{Type: "system.output.finalized", OccurredAt: time.Date(2026, 3, 16, 9, 0, 1, 0, time.UTC), Payload: []byte(`{"output":"done"}`)},
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"done","total_tokens":12}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}

	if evaluation.Status != EvaluationStatusComplete {
		t.Fatalf("evaluation status = %s, want %s", evaluation.Status, EvaluationStatusComplete)
	}
	if got := evaluation.ValidatorResults[0].Verdict; got != "pass" {
		t.Fatalf("validator verdict = %q, want pass", got)
	}
	if evaluation.ValidatorResults[0].State != OutputStateAvailable {
		t.Fatalf("validator state = %s, want available", evaluation.ValidatorResults[0].State)
	}
	if evaluation.MetricResults[2].NumericValue == nil || *evaluation.MetricResults[2].NumericValue != 12 {
		t.Fatalf("total token metric = %v, want 12", evaluation.MetricResults[2].NumericValue)
	}
	if evaluation.DimensionScores[string(ScorecardDimensionCorrectness)] == nil || *evaluation.DimensionScores[string(ScorecardDimensionCorrectness)] != 1 {
		t.Fatalf("correctness score = %v, want 1", evaluation.DimensionScores[string(ScorecardDimensionCorrectness)])
	}
	if evaluation.DimensionScores[string(ScorecardDimensionReliability)] == nil || *evaluation.DimensionScores[string(ScorecardDimensionReliability)] != 1 {
		t.Fatalf("reliability score = %v, want 1", evaluation.DimensionScores[string(ScorecardDimensionReliability)])
	}
}

func TestEvaluateRunAgentReturnsPartialWhenEvidenceIsMissing(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "exact",
				Type:         ValidatorTypeExactMatch,
				Target:       "final_output",
				ExpectedFrom: "challenge_input",
			},
		},
		Metrics: []MetricDeclaration{
			{Key: "latency", Type: MetricTypeNumeric, Collector: "run_total_latency_ms", Unit: "ms"},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness, ScorecardDimensionLatency},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.started", OccurredAt: time.Date(2026, 3, 16, 9, 0, 0, 0, time.UTC), Payload: []byte(`{}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}

	if evaluation.Status != EvaluationStatusPartial {
		t.Fatalf("evaluation status = %s, want partial", evaluation.Status)
	}
	if evaluation.ValidatorResults[0].State != OutputStateUnavailable {
		t.Fatalf("validator state = %s, want unavailable", evaluation.ValidatorResults[0].State)
	}
	if evaluation.MetricResults[0].State != OutputStateUnavailable {
		t.Fatalf("metric state = %s, want unavailable", evaluation.MetricResults[0].State)
	}
	if evaluation.DimensionScores[string(ScorecardDimensionCorrectness)] != nil {
		t.Fatalf("correctness score = %v, want nil", evaluation.DimensionScores[string(ScorecardDimensionCorrectness)])
	}
	if evaluation.DimensionScores[string(ScorecardDimensionLatency)] != nil {
		t.Fatalf("latency score = %v, want nil", evaluation.DimensionScores[string(ScorecardDimensionLatency)])
	}
}

func TestEvaluateRunAgentMarksInvalidRegexAsValidatorError(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "regex",
				Type:         ValidatorTypeRegexMatch,
				Target:       "final_output",
				ExpectedFrom: "literal:[",
			},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"done"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}

	if evaluation.ValidatorResults[0].State != OutputStateError {
		t.Fatalf("validator state = %s, want error", evaluation.ValidatorResults[0].State)
	}
	if evaluation.ValidatorResults[0].Verdict != "error" {
		t.Fatalf("validator verdict = %q, want error", evaluation.ValidatorResults[0].Verdict)
	}
	if evaluation.DimensionScores[string(ScorecardDimensionCorrectness)] != nil {
		t.Fatalf("correctness score = %v, want nil", evaluation.DimensionScores[string(ScorecardDimensionCorrectness)])
	}
}

func TestEvaluateRunAgentComputesValidatorPassRateAfterValidators(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "exact",
				Type:         ValidatorTypeExactMatch,
				Target:       "final_output",
				ExpectedFrom: "challenge_input",
			},
		},
		Metrics: []MetricDeclaration{
			{Key: "validator_pass_rate", Type: MetricTypeNumeric, Collector: "validator_pass_rate"},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		ChallengeInputs: []EvidenceInput{
			{
				ChallengeIdentityID: uuid.New(),
				ItemKey:             "expected.txt",
				Payload:             []byte(`"done"`),
			},
		},
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"done"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}

	if len(evaluation.MetricResults) != 1 {
		t.Fatalf("metric result count = %d, want 1", len(evaluation.MetricResults))
	}
	if evaluation.MetricResults[0].State != OutputStateAvailable {
		t.Fatalf("metric state = %s, want available", evaluation.MetricResults[0].State)
	}
	if evaluation.MetricResults[0].NumericValue == nil || *evaluation.MetricResults[0].NumericValue != 1 {
		t.Fatalf("validator_pass_rate = %v, want 1", evaluation.MetricResults[0].NumericValue)
	}
}

func TestEvaluateRunAgentValidatesJSONSchema(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "schema",
				Type:         ValidatorTypeJSONSchema,
				Target:       "final_output",
				ExpectedFrom: `literal:{"type":"object","properties":{"answer":{"type":"string"}}}`,
			},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"{\"answer\":\"done\"}"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}
	if evaluation.ValidatorResults[0].Verdict != "pass" {
		t.Fatalf("validator verdict = %q, want pass", evaluation.ValidatorResults[0].Verdict)
	}
	if evaluation.ValidatorResults[0].State != OutputStateAvailable {
		t.Fatalf("validator state = %s, want available", evaluation.ValidatorResults[0].State)
	}
	if !strings.Contains(string(evaluation.ValidatorResults[0].RawOutput), `"schema_draft":"https://json-schema.org/draft/2020-12/schema"`) {
		t.Fatalf("raw_output = %s, want schema draft evidence", evaluation.ValidatorResults[0].RawOutput)
	}
}

func TestEvaluateRunAgentReturnsFailureForJSONSchemaMismatch(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "schema",
				Type:         ValidatorTypeJSONSchema,
				Target:       "final_output",
				ExpectedFrom: `literal:{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","required":["answer","score"],"properties":{"answer":{"type":"string"},"score":{"type":"number"}}}`,
			},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"{\"answer\":42}"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}
	if evaluation.ValidatorResults[0].Verdict != "fail" {
		t.Fatalf("validator verdict = %q, want fail", evaluation.ValidatorResults[0].Verdict)
	}
	if evaluation.ValidatorResults[0].Reason != "json schema validation failed" {
		t.Fatalf("validator reason = %q, want json schema validation failed", evaluation.ValidatorResults[0].Reason)
	}
	if !strings.Contains(string(evaluation.ValidatorResults[0].RawOutput), `"validation_error"`) {
		t.Fatalf("raw_output = %s, want validation error evidence", evaluation.ValidatorResults[0].RawOutput)
	}
}

func TestEvaluateRunAgentReturnsErrorForMalformedJSONValidatorInput(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "schema",
				Type:         ValidatorTypeJSONSchema,
				Target:       "final_output",
				ExpectedFrom: `literal:{"type":"object"`,
			},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"{\"answer\":\"done\"}"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}
	if evaluation.ValidatorResults[0].State != OutputStateError {
		t.Fatalf("validator state = %s, want error", evaluation.ValidatorResults[0].State)
	}
	if evaluation.ValidatorResults[0].Verdict != "error" {
		t.Fatalf("validator verdict = %q, want error", evaluation.ValidatorResults[0].Verdict)
	}
	if !strings.Contains(evaluation.ValidatorResults[0].Reason, "parse JSON schema") {
		t.Fatalf("validator reason = %q, want parse JSON schema error", evaluation.ValidatorResults[0].Reason)
	}
}

func TestEvaluateRunAgentMatchesJSONPathComparators(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "status",
				Type:         ValidatorTypeJSONPathMatch,
				Target:       "final_output",
				ExpectedFrom: `literal:{"path":"$.status","comparator":"equals","value":"done"}`,
			},
			{
				Key:          "score",
				Type:         ValidatorTypeJSONPathMatch,
				Target:       "final_output",
				ExpectedFrom: `literal:{"path":"$.score","comparator":"greater_than","value":10}`,
			},
			{
				Key:          "summary",
				Type:         ValidatorTypeJSONPathMatch,
				Target:       "final_output",
				ExpectedFrom: `literal:{"path":"$.summary","comparator":"contains","value":"success"}`,
			},
			{
				Key:          "exists",
				Type:         ValidatorTypeJSONPathMatch,
				Target:       "final_output",
				ExpectedFrom: `literal:$.details.items[0].id`,
			},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"{\"status\":\"done\",\"score\":11,\"summary\":\"operation success\",\"details\":{\"items\":[{\"id\":\"abc\"}]}}"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}

	for i, result := range evaluation.ValidatorResults {
		if result.Verdict != "pass" {
			t.Fatalf("validator[%d] verdict = %q, want pass", i, result.Verdict)
		}
		if !strings.Contains(string(result.RawOutput), `"path":`) {
			t.Fatalf("validator[%d] raw_output = %s, want path evidence", i, result.RawOutput)
		}
	}
}

func TestEvaluateRunAgentReturnsFailureForJSONPathMismatch(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "score",
				Type:         ValidatorTypeJSONPathMatch,
				Target:       "final_output",
				ExpectedFrom: `literal:{"path":"$.score","comparator":"less_than","value":5}`,
			},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"{\"score\":11}"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}
	if evaluation.ValidatorResults[0].Verdict != "fail" {
		t.Fatalf("validator verdict = %q, want fail", evaluation.ValidatorResults[0].Verdict)
	}
	if evaluation.ValidatorResults[0].Reason != "json path value was not less than expected value" {
		t.Fatalf("validator reason = %q, want less-than failure", evaluation.ValidatorResults[0].Reason)
	}
}

func TestEvaluateRunAgentReturnsErrorForMalformedJSONPathExpectation(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "path",
				Type:         ValidatorTypeJSONPathMatch,
				Target:       "final_output",
				ExpectedFrom: `literal:{"path":"$.score","comparator":"between","value":10}`,
			},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"{\"score\":11}"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}
	if evaluation.ValidatorResults[0].State != OutputStateError {
		t.Fatalf("validator state = %s, want error", evaluation.ValidatorResults[0].State)
	}
	if !strings.Contains(evaluation.ValidatorResults[0].Reason, "unsupported comparator") {
		t.Fatalf("validator reason = %q, want unsupported comparator error", evaluation.ValidatorResults[0].Reason)
	}
}

func TestEvaluateRunAgentWarnsWhenChallengeInputIsAmbiguousAcrossMultipleItems(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "exact",
				Type:         ValidatorTypeExactMatch,
				Target:       "final_output",
				ExpectedFrom: "challenge_input",
			},
		},
		Metrics: []MetricDeclaration{
			{Key: "total_tokens", Type: MetricTypeNumeric, Collector: "run_total_tokens"},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		ChallengeInputs: []EvidenceInput{
			{
				ChallengeIdentityID: uuid.New(),
				ItemKey:             "first.txt",
				Payload:             []byte(`"done"`),
			},
			{
				ChallengeIdentityID: uuid.New(),
				ItemKey:             "second.txt",
				Payload:             []byte(`"other"`),
			},
		},
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"done","total_tokens":12}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}

	if evaluation.Status != EvaluationStatusPartial {
		t.Fatalf("evaluation status = %s, want partial", evaluation.Status)
	}
	if len(evaluation.ValidatorResults) != 1 {
		t.Fatalf("validator result count = %d, want 1", len(evaluation.ValidatorResults))
	}
	if evaluation.ValidatorResults[0].State != OutputStateUnavailable {
		t.Fatalf("validator state = %s, want unavailable", evaluation.ValidatorResults[0].State)
	}
	if evaluation.ValidatorResults[0].Reason != "challenge input evidence is unavailable" {
		t.Fatalf("validator reason = %q, want challenge input evidence is unavailable", evaluation.ValidatorResults[0].Reason)
	}
	if len(evaluation.MetricResults) != 1 || evaluation.MetricResults[0].NumericValue == nil || *evaluation.MetricResults[0].NumericValue != 12 {
		t.Fatalf("total token metric = %#v, want numeric value 12", evaluation.MetricResults)
	}
	if !containsString(evaluation.Warnings, "challenge input is ambiguous across multiple items") {
		t.Fatalf("warnings = %v, want ambiguity warning", evaluation.Warnings)
	}
	if evaluation.DimensionScores[string(ScorecardDimensionCorrectness)] != nil {
		t.Fatalf("correctness score = %v, want nil", evaluation.DimensionScores[string(ScorecardDimensionCorrectness)])
	}
}

func TestEvaluateRunAgentSurfacesStubDimensionReasonsAsWarnings(t *testing.T) {
	spec := EvaluationSpec{
		Name:          "fixture",
		VersionNumber: 1,
		JudgeMode:     JudgeModeDeterministic,
		Validators: []ValidatorDeclaration{
			{
				Key:          "exact",
				Type:         ValidatorTypeExactMatch,
				Target:       "final_output",
				ExpectedFrom: "challenge_input",
			},
		},
		Metrics: []MetricDeclaration{
			{Key: "completed", Type: MetricTypeBoolean, Collector: "run_completed_successfully"},
			{Key: "failures", Type: MetricTypeNumeric, Collector: "run_failure_count"},
		},
		Scorecard: ScorecardDeclaration{
			Dimensions: []ScorecardDimension{ScorecardDimensionCorrectness, ScorecardDimensionLatency, ScorecardDimensionCost},
		},
	}

	evaluation, err := EvaluateRunAgent(EvaluationInput{
		RunAgentID:       uuid.New(),
		EvaluationSpecID: uuid.New(),
		ChallengeInputs: []EvidenceInput{
			{
				ChallengeIdentityID: uuid.New(),
				ItemKey:             "expected.txt",
				Payload:             []byte(`"done"`),
			},
		},
		Events: []Event{
			{Type: "system.run.completed", OccurredAt: time.Date(2026, 3, 16, 9, 0, 2, 0, time.UTC), Payload: []byte(`{"final_output":"done"}`)},
		},
	}, spec)
	if err != nil {
		t.Fatalf("EvaluateRunAgent returned error: %v", err)
	}

	if !containsString(evaluation.Warnings, "latency dimension normalization is not defined yet") {
		t.Fatalf("warnings = %v, want latency stub warning", evaluation.Warnings)
	}
	if !containsString(evaluation.Warnings, "cost dimension normalization is not defined yet") {
		t.Fatalf("warnings = %v, want cost stub warning", evaluation.Warnings)
	}
}

func TestExtractLooseStringPrefersValueThenContentThenTextThenAnswer(t *testing.T) {
	value, ok := extractLooseString(map[string]any{
		"answer":  "answer-value",
		"text":    "text-value",
		"content": "content-value",
		"value":   "value-choice",
	})
	if !ok {
		t.Fatal("extractLooseString returned not ok")
	}
	if value != "value-choice" {
		t.Fatalf("value = %q, want value-choice", value)
	}

	value, ok = extractLooseString(map[string]any{
		"answer":  "answer-value",
		"text":    "text-value",
		"content": "content-choice",
	})
	if !ok {
		t.Fatal("extractLooseString returned not ok for content-choice")
	}
	if value != "content-choice" {
		t.Fatalf("value = %q, want content-choice", value)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
