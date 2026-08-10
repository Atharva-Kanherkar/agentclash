package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/challengepack"
	"github.com/agentclash/agentclash/runtime/scoring"
)

// Mapping a completed agent tryout onto the visual pack builder's working
// document (challengepack.Composition). The tryout's own snapshots use a
// deliberately looser shorthand than a runnable pack — its validators name
// shapes like json_field / artifact_produced that the scoring engine does not
// implement, and its scorecard is a list of display labels. So the mapper
// re-derives the pack from the evidence the tryout genuinely carries:
//
//	template_snapshot.runtime.expected_artifacts -> post-execution file captures
//	                                                + file_exists validators
//	evaluation_spec_snapshot.llm_judges          -> judges (already the exact
//	                                                scoring.LLMJudgeDeclaration
//	                                                shape) + one dimension each
//	input_snapshot                               -> the single input-set case
//	tool_policy_snapshot                         -> version.tool_policy verbatim
//	cost_limit_usd / max_duration_seconds        -> runtime limits + latency and
//	                                                cost dimension normalization
//
// The result is a Bundle, which BundleToComposition turns into a Composition
// for free — including the Advanced passthrough for the fields the builder UI
// cannot render yet (post-execution checks, runtime limits).

const (
	// tryoutPackFamily groups promoted tryouts in the pack catalog. It is
	// deliberately not "security", which would demand a security policy.
	tryoutPackFamily = "tryout"
	// tryoutPackDifficulty is the neutral default; the builder lets the
	// author change it before publishing.
	tryoutPackDifficulty  = "medium"
	tryoutPackInputSetKey = "tryout-input"
	tryoutPackCaseKey     = "tryout-case"
	// tryoutPackSlugIDLength is how much of the source tryout id is folded
	// into the pack slug so two promotions of the same template do not
	// collide on publish.
	tryoutPackSlugIDLength = 8
)

// tryoutTemplateSnapshotView is the slice of template_snapshot the mapper reads.
type tryoutTemplateSnapshotView struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Runtime     struct {
		Instructions      string                   `json:"instructions"`
		ExpectedArtifacts []tryoutExpectedArtifact `json:"expected_artifacts"`
	} `json:"runtime"`
}

type tryoutExpectedArtifact struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	Path string `json:"path"`
}

// tryoutEvaluationSnapshotView is the slice of evaluation_spec_snapshot the
// mapper reads. Judge declarations are already stored in the exact shape
// scoring.LLMJudgeDeclaration decodes (see agent_tryout_judge.go), so they are
// carried over verbatim.
type tryoutEvaluationSnapshotView struct {
	LLMJudges []scoring.LLMJudgeDeclaration `json:"llm_judges"`
	Scorecard struct {
		Dimensions []scoring.DimensionDeclaration `json:"dimensions"`
	} `json:"scorecard"`
}

// agentTryoutPackDraft renders the builder inputs for a promoted tryout: the
// draft name and the composition JSON stored verbatim in
// challenge_pack_drafts.composition.
func agentTryoutPackDraft(source repository.AgentTryout) (string, json.RawMessage, error) {
	bundle := agentTryoutPackBundle(source)
	composition, err := challengepack.BundleToComposition(bundle)
	if err != nil {
		return "", nil, fmt.Errorf("build composition from tryout %s: %w", source.ID, err)
	}
	encoded, err := json.Marshal(composition)
	if err != nil {
		return "", nil, fmt.Errorf("marshal composition for tryout %s: %w", source.ID, err)
	}
	return bundle.Pack.Name, encoded, nil
}

// agentTryoutPackBundle assembles the runnable-pack shape a tryout implies. It
// is intentionally total — a sparse tryout yields a thin but still valid pack
// rather than an error, so promotion never dead-ends on a missing snapshot.
func agentTryoutPackBundle(source repository.AgentTryout) challengepack.Bundle {
	template := decodeTryoutTemplateSnapshot(source.TemplateSnapshot)
	evaluation := decodeTryoutEvaluationSnapshot(source.EvaluationSpecSnapshot)

	challengeKey := tryoutSlugify(firstNonEmpty(source.TemplateSlug, template.Slug), "tryout")
	title := firstNonEmpty(template.Name, source.TemplateSlug, "Agent tryout")

	bundle := challengepack.Bundle{
		Pack: challengepack.PackMetadata{
			Slug:        tryoutPackSlug(challengeKey, source),
			Name:        title,
			Family:      tryoutPackFamily,
			Description: optionalTrimmedString(template.Description),
		},
		Version: challengepack.VersionMetadata{
			Number:        1,
			ExecutionMode: challengepack.ExecutionModeNative,
			ToolPolicy:    decodeJSONObjectOrNil(source.ToolPolicySnapshot),
		},
		Challenges: []challengepack.ChallengeDefinition{{
			Key:          challengeKey,
			Title:        title,
			Category:     challengeKey,
			Difficulty:   tryoutPackDifficulty,
			Instructions: strings.TrimSpace(template.Runtime.Instructions),
		}},
		InputSets: []challengepack.InputSetDefinition{{
			Key:   tryoutPackInputSetKey,
			Name:  "Tryout input",
			Cases: []challengepack.CaseDefinition{{ChallengeKey: challengeKey, CaseKey: tryoutPackCaseKey, Payload: decodeJSONObjectOrNil(source.InputSnapshot)}},
		}},
	}

	checks, validators := tryoutArtifactEvidence(template.Runtime.ExpectedArtifacts)
	spec := scoring.EvaluationSpec{
		Name:                bundle.Pack.Slug,
		VersionNumber:       bundle.Version.Number,
		Validators:          validators,
		LLMJudges:           evaluation.LLMJudges,
		PostExecutionChecks: checks,
		RuntimeLimits:       tryoutRuntimeLimits(source),
		Scorecard:           scoring.ScorecardDeclaration{Strategy: scoring.ScoringStrategyWeighted},
	}
	// JudgeMode is left empty on purpose: ComposeBundle infers it from what
	// the spec actually declares when the draft is compiled.
	spec.Scorecard.Dimensions = tryoutDimensions(evaluation, source)
	bundle.Version.EvaluationSpec = spec

	return bundle
}

// tryoutArtifactEvidence turns the template's declared output files into the
// deterministic half of the scorecard: a file capture per artifact plus a
// file_exists validator that asserts the agent actually produced it. When the
// template declares no artifacts, it falls back to asserting a non-empty final
// output so the pack still has the one validator ValidateEvaluationSpec
// requires.
func tryoutArtifactEvidence(artifacts []tryoutExpectedArtifact) ([]scoring.PostExecutionCheck, []scoring.ValidatorDeclaration) {
	var checks []scoring.PostExecutionCheck
	var validators []scoring.ValidatorDeclaration
	seen := map[string]struct{}{}

	for _, artifact := range artifacts {
		key := tryoutSlugify(artifact.Key, "")
		path := strings.TrimSpace(artifact.Path)
		if key == "" || path == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		checks = append(checks, scoring.PostExecutionCheck{
			Key:  key,
			Type: scoring.PostExecutionCheckTypeFileCapture,
			Path: path,
		})
		validators = append(validators, scoring.ValidatorDeclaration{
			Key:    "produced_" + key,
			Type:   scoring.ValidatorTypeFileExists,
			Target: "file:" + key,
		})
	}

	if len(validators) == 0 {
		validators = append(validators, scoring.ValidatorDeclaration{
			Key:  "final_output_not_empty",
			Type: scoring.ValidatorTypeRegexMatch,
			// literal: supplies the pattern inline; ".+" matches any
			// non-empty final output.
			Target:       "final_output",
			ExpectedFrom: "literal:.+",
		})
	}
	return checks, validators
}

// tryoutRuntimeLimits carries the tryout's enforced budget into the pack so the
// promoted eval runs under the same ceiling the tryout did.
func tryoutRuntimeLimits(source repository.AgentTryout) scoring.RuntimeLimits {
	var limits scoring.RuntimeLimits
	if source.CostLimitUSD > 0 {
		cost := source.CostLimitUSD
		limits.MaxCostUSD = &cost
	}
	if source.MaxDurationSeconds > 0 {
		durationMS := int64(source.MaxDurationSeconds) * 1000
		limits.MaxDurationMS = &durationMS
	}
	return limits
}

// tryoutDimensions wires the scorecard. Only dimensions backed by evidence the
// promoted pack actually produces are emitted: correctness (always, since there
// is always at least one validator), the builtin dimensions the template asked
// for, and one llm_judge dimension per carried-over judge. The template's
// free-text labels ("accuracy", "visuals", ...) have no source in the scoring
// engine and are deliberately dropped — the builder's dimension editor is where
// an author re-adds them against a real source.
func tryoutDimensions(evaluation tryoutEvaluationSnapshotView, source repository.AgentTryout) []scoring.DimensionDeclaration {
	requested := map[string]struct{}{scoring.ScorecardDimensionCorrectness: {}}
	for _, dim := range evaluation.Scorecard.Dimensions {
		if key := strings.TrimSpace(dim.Key); key != "" {
			requested[key] = struct{}{}
		}
	}

	dimensions := make([]scoring.DimensionDeclaration, 0, len(requested))
	used := map[string]struct{}{}
	appendDim := func(dim scoring.DimensionDeclaration) {
		if _, exists := used[dim.Key]; exists {
			return
		}
		used[dim.Key] = struct{}{}
		dimensions = append(dimensions, dim)
	}

	// Emitted in a fixed order so the composition is deterministic.
	appendDim(scoring.DimensionDeclaration{
		Key:    scoring.ScorecardDimensionCorrectness,
		Source: scoring.DimensionSourceValidators,
	})
	if _, ok := requested[scoring.ScorecardDimensionReliability]; ok {
		appendDim(scoring.DimensionDeclaration{
			Key:    scoring.ScorecardDimensionReliability,
			Source: scoring.DimensionSourceReliability,
		})
	}
	if _, ok := requested[scoring.ScorecardDimensionLatency]; ok && source.MaxDurationSeconds > 0 {
		max := float64(source.MaxDurationSeconds) * 1000
		appendDim(scoring.DimensionDeclaration{
			Key:             scoring.ScorecardDimensionLatency,
			Source:          scoring.DimensionSourceLatency,
			BetterDirection: "lower",
			Normalization:   tryoutHalfwayNormalization(max),
		})
	}
	if _, ok := requested[scoring.ScorecardDimensionCost]; ok && source.CostLimitUSD > 0 {
		appendDim(scoring.DimensionDeclaration{
			Key:             scoring.ScorecardDimensionCost,
			Source:          scoring.DimensionSourceCost,
			BetterDirection: "lower",
			Normalization:   tryoutHalfwayNormalization(source.CostLimitUSD),
		})
	}

	for _, judge := range evaluation.LLMJudges {
		key := strings.TrimSpace(judge.Key)
		if key == "" {
			continue
		}
		appendDim(scoring.DimensionDeclaration{
			Key:      key,
			Source:   scoring.DimensionSourceLLMJudge,
			JudgeKey: key,
		})
	}

	return dimensions
}

// tryoutHalfwayNormalization scores "half the budget" as perfect and "the whole
// budget" as zero, which is the only defensible curve we can derive from a cap
// alone. Authors retune it in the builder.
func tryoutHalfwayNormalization(max float64) *scoring.DimensionNormalization {
	target := max / 2
	return &scoring.DimensionNormalization{Target: &target, Max: &max}
}

// tryoutPackSlug keeps promoted packs from colliding: two promotions of the
// same template would otherwise both publish version 1 of one pack.
func tryoutPackSlug(challengeKey string, source repository.AgentTryout) string {
	suffix := strings.ReplaceAll(source.ID.String(), "-", "")
	if len(suffix) > tryoutPackSlugIDLength {
		suffix = suffix[:tryoutPackSlugIDLength]
	}
	if suffix == "" {
		return "tryout-" + challengeKey
	}
	return "tryout-" + challengeKey + "-" + suffix
}

func decodeTryoutTemplateSnapshot(raw json.RawMessage) tryoutTemplateSnapshotView {
	var view tryoutTemplateSnapshotView
	if len(raw) == 0 {
		return view
	}
	// A malformed snapshot degrades to an empty view rather than failing the
	// promotion: the builder is the place to fill the gaps in.
	_ = json.Unmarshal(raw, &view)
	return view
}

func decodeTryoutEvaluationSnapshot(raw json.RawMessage) tryoutEvaluationSnapshotView {
	var view tryoutEvaluationSnapshotView
	if len(raw) == 0 {
		return view
	}
	_ = json.Unmarshal(raw, &view)
	return view
}

// decodeJSONObjectOrNil returns raw as a generic object, or nil when it is
// absent or is not a JSON object (a non-object payload has no place in a
// tool_policy / case payload map).
func decodeJSONObjectOrNil(raw json.RawMessage) map[string]any {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil
	}
	return object
}

// tryoutSlugify reduces a value to lowercase alphanumerics plus single dashes
// so it is safe as a pack slug, challenge key, or validator key.
func tryoutSlugify(value, fallback string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case r == '_':
			b.WriteRune('_')
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return fallback
	}
	return slug
}

func optionalTrimmedString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
