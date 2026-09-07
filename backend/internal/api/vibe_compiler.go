package api

import (
	"encoding/json"
	"fmt"
	"github.com/agentclash/agentclash/backend/internal/vibe"
	"github.com/agentclash/agentclash/runtime/challengepack"
	"github.com/agentclash/agentclash/runtime/scoring"
	"github.com/google/uuid"
	"strings"
)

// VibePackCompiler reuses #1246's narrow blueprint and #1245's shared
// BundleToComposition conversion. The interactive builder's permissive fallback
// is deliberately not used: reduced coverage is an error, never a normal run.
type VibePackCompiler struct{}

func (VibePackCompiler) Instructions() string {
	// Reuse the merged blueprint and scoring enums, not the builder's nested
	// task/rules JSON prompt, which small models may echo as blueprint fields.
	return fmt.Sprintf(`Inside draft.blueprint use ONLY this JSON shape, replacing the example values with the user's task:
%s
The overall response is still {"reply":"...","requirements":[],"draft":null OR {"title":"...","agent_prompt":"...","blueprint":...}}. Do not copy prompt metadata or add other keys. Never duplicate keys. Keep reply under 80 words in everyday language; describe the examples and expected behavior, without compiler, validator or judge terminology.
Use one to three cases with distinct keys. All validators and judges run on EVERY case. Never require mutually exclusive phrases globally. Conversational policy correctness needs one conditional assertion about the correct response for the given case; allow equivalent wording. When using a judge, the ONLY allowed literal text check is the example's regex_match with expected_from literal:.+. Put policy criteria in the assertion or use case-specific expected-value references. Omit judges only for entirely mechanical checks.
Validator types: %v. Targets must be final_output. Expected values use literal:<text> or case.payload.<field>; referenced payload fields are withheld from the agent. Other payload fields are user-facing inputs, never answer keys.
Dimension sources: %v. A validators dimension lists its validator keys; an llm_judge dimension sets judge_key and no validators; reliability sets neither. All validator, judge and dimension keys must be distinct. Judge modes: %v; assertion supplies one true/false claim, rubric instead supplies scoring criteria for 1 through 5. At most one judge.
Use explicit purchase ages or both relevant dates for date-sensitive cases. Preserve supplied policies and distinguish proposed assumptions from accepted requirements.`, vibeBlueprintExample, generatedPackValidatorTypes, generatedPackDimensionSources, generatedPackJudgeModes)
}

const vibeBlueprintExample = `{"slug":"agent-check","name":"Agent check","description":"Check the requested behavior","difficulty":"easy","instructions":"The complete task instructions","cases":[{"key":"example","payload":{"question":"A concrete user request"}}],"validators":[{"key":"has_answer","type":"regex_match","target":"final_output","expected_from":"literal:.+"}],"judges":[{"key":"behavior","mode":"assertion","assertion":"The response answers this case correctly under the supplied policy."}],"dimensions":[{"key":"output_present","source":"validators","validators":["has_answer"]},{"key":"policy_correctness","source":"llm_judge","judge_key":"behavior"}]}`

// ValidateDraft constrains new AI-authored previews. Explicit imports and
// accepted evaluation contracts are compiled unchanged by Compile below.
func (VibePackCompiler) ValidateDraft(content json.RawMessage, l vibe.Limits) error {
	var p generatedPackBlueprint
	if err := vibe.Decode(content, l, &p); err != nil {
		return err
	}
	for _, v := range p.Validators {
		if len(p.Judges) > 0 && strings.HasPrefix(v.ExpectedFrom, "literal:") {
			switch v.Type {
			case scoring.ValidatorTypeContains, scoring.ValidatorTypeExactMatch, scoring.ValidatorTypeRegexMatch, scoring.ValidatorTypeNormalizedMatch, scoring.ValidatorTypeFuzzyMatch, scoring.ValidatorTypeTokenF1:
				if v.Type != scoring.ValidatorTypeRegexMatch || v.ExpectedFrom != "literal:.+" {
					return fmt.Errorf("a generated semantic preview cannot require a literal phrase on every case; use regex_match with literal:.+ for nonempty output and put conditional policy checks in the assertion, or use case-specific expected-value references; no checks were removed")
				}
			}
		}
	}
	return nil
}

func (VibePackCompiler) Compile(content json.RawMessage, evaluator string, id uuid.UUID, l vibe.Limits) (vibe.Compiled, error) {
	var out vibe.Compiled
	if err := vibe.ValidateJSON(content, l); err != nil {
		return out, err
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(content, &shape); err != nil {
		return out, err
	}
	var b challengepack.Bundle
	if _, ok := shape["pack"]; ok {
		if err := vibe.Decode(content, l, &b); err != nil {
			return out, err
		}
		if b.Version.ExecutionMode != "" && b.Version.ExecutionMode != challengepack.ExecutionModePromptEval && b.Version.ExecutionMode != challengepack.ExecutionModeNative {
			return out, fmt.Errorf("this pack needs an advanced execution mode; open it in the pack builder")
		}
	} else {
		var p generatedPackBlueprint
		if err := vibe.Decode(content, l, &p); err != nil {
			return out, err
		}
		if err := checkGeneratedPackChoices(p); err != nil {
			return out, err
		}
		if len(p.Cases) == 0 {
			return out, fmt.Errorf("the generated evaluation contains no cases")
		}
		seen := map[string]bool{}
		for _, c := range p.Cases {
			if c.Key == "" || seen[c.Key] {
				return out, fmt.Errorf("case keys must be nonempty and unique; no cases were removed")
			}
			seen[c.Key] = true
		}
		for _, j := range p.Judges {
			if strings.TrimSpace(j.Key) == "" {
				return out, fmt.Errorf("an evaluator has an empty key; no evaluators were removed")
			}
		}
		b = generatedPackBundle(p, evaluator, id, true)
		if len(b.Version.EvaluationSpec.LLMJudges) != len(p.Judges) || len(b.Version.EvaluationSpec.Scorecard.Dimensions) != len(p.Dimensions) {
			return out, fmt.Errorf("the requested evaluation has invalid evaluator references; no coverage was removed")
		}
	}
	if len(b.Tools) > 0 || len(b.Version.ToolPolicy) > 0 || len(b.Version.Filesystem) > 0 || b.Version.Sandbox != nil || len(b.Version.Assets) > 0 || b.Modality != "" || b.Security != nil {
		return out, fmt.Errorf("this pack requires capabilities unavailable in a text preview; open the original pack in the builder")
	}
	if len(b.Challenges) != 1 {
		return out, fmt.Errorf("a text preview supports one task; split multi-task packs explicitly")
	}
	if len(b.Challenges[0].Assets) > 0 || len(b.Challenges[0].ArtifactRefs) > 0 {
		return out, fmt.Errorf("file-backed challenges need the advanced runner")
	}
	spec := &b.Version.EvaluationSpec
	if len(spec.Validators) == 0 || len(spec.Validators) > l.Checks || len(spec.LLMJudges) > l.Evaluators {
		return out, fmt.Errorf("evaluation exceeds the validator or evaluator limit")
	}
	for _, v := range spec.Validators {
		if len(v.Key) > vibe.MaxKeyBytes {
			return out, fmt.Errorf("validator key exceeds 128 bytes")
		}
		if err := checkGeneratedPackChoice("validator", v.Key, v.Type, generatedPackValidatorTypes); err != nil {
			return out, err
		}
		if v.Target != "final_output" {
			return out, fmt.Errorf("preview validators must evaluate the agent output")
		}
		if v.Type == scoring.ValidatorTypeRegexMatch && len(v.ExpectedFrom) > 1024 {
			return out, fmt.Errorf("regex exceeds the 1 KiB preview limit")
		}
		if len(v.ExpectedFrom) > 2048 {
			return out, fmt.Errorf("validator reference exceeds its bounded text allowance")
		}
	}
	for i := range spec.LLMJudges {
		j := &spec.LLMJudges[i]
		if len(j.Key) > vibe.MaxKeyBytes {
			return out, fmt.Errorf("evaluator key exceeds 128 bytes")
		}
		if len(j.Models) > 0 || j.Samples > 1 {
			return out, fmt.Errorf("multi-model judges and repeated sampling require an advanced run")
		}
		if j.Mode != scoring.JudgeMethodAssertion && j.Mode != scoring.JudgeMethodRubric {
			return out, fmt.Errorf("this evaluator needs evidence unavailable to the text preview")
		}
		if len(j.ContextFrom) > 0 || len(j.OutputSchema) > 0 || j.Consensus != nil || j.ReferenceFrom != "" {
			return out, fmt.Errorf("this evaluator has context or schema requirements unavailable in a text preview; no evaluators were removed")
		}
		j.Model = evaluator // explicit role policy, never the authoring/target model
	}
	for _, set := range b.InputSets {
		for _, c := range set.Cases {
			if len(c.CaseKey) > vibe.MaxKeyBytes {
				return out, fmt.Errorf("case key exceeds 128 bytes")
			}
			if len(c.Artifacts) > 0 || len(c.Assets) > 0 || c.UserSimulator != nil {
				return out, fmt.Errorf("artifact-backed cases need the advanced runner")
			}
			for _, input := range c.Inputs {
				if input.ArtifactKey != "" || input.Path != "" {
					return out, fmt.Errorf("file inputs require an advanced runner")
				}
			}
			for _, expectation := range c.Expectations {
				if expectation.ArtifactKey != "" {
					return out, fmt.Errorf("file expectations require an advanced runner")
				}
			}
			if len(vibe.TargetInput(c, *spec)) == 0 {
				return out, fmt.Errorf("case has no target inputs after withholding expected answers")
			}
			// Resolve structured/regex operands through the existing evidence
			// resolver before execution. A short reference cannot hide a giant
			// regex or unavailable file-backed expectation.
			for _, validator := range spec.Validators {
				if !validator.Type.RequiresExpectedFrom() {
					continue
				}
				value, _, err := scoring.ResolveEvidenceValueForJudge(validator.ExpectedFrom, scoring.EvaluationInput{ChallengeInputs: []scoring.EvidenceInput{vibe.CaseEvidence(c)}})
				if err != nil || value == nil {
					return out, fmt.Errorf("validator %s has unavailable expected evidence in case %s", validator.Key, c.CaseKey)
				}
				if validator.Type == scoring.ValidatorTypeRegexMatch && len(*value) > 1024 {
					return out, fmt.Errorf("resolved regex exceeds the 1 KiB preview limit")
				}
			}
			out.Cases = append(out.Cases, c)
		}
	}
	if len(out.Cases) == 0 || len(out.Cases) > l.ImportCases {
		return out, fmt.Errorf("evaluation must contain between 1 and %d cases", l.ImportCases)
	}
	b.Version.ExecutionMode = challengepack.ExecutionModePromptEval
	composition, err := challengepack.BundleToComposition(b)
	if err != nil {
		return out, err
	}
	composed, err := challengepack.ComposeBundle(composition, challengepack.ResolvedPieces{})
	if err != nil {
		return out, err
	}
	if err = challengepack.ValidateBundle(composed); err != nil {
		return out, fmt.Errorf("evaluation did not compile without reducing coverage: %w", err)
	}
	// Execute the compiled bundle, including inferred judge mode and defaults.
	// The authoring bundle intentionally leaves these fields for the builder.
	out.Bundle = composed
	out.Composition, err = json.Marshal(composition)
	if err != nil {
		return out, err
	}
	return out, nil
}
