package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/releasegate"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/google/uuid"
)

type regressionScorecardDocument struct {
	ValidatorDetails []regressionValidatorDetail `json:"validator_details"`
	MetricDetails    []regressionMetricDetail    `json:"metric_details"`
}

type regressionValidatorDetail struct {
	Key              string                 `json:"key"`
	State            string                 `json:"state"`
	RegressionCaseID *uuid.UUID             `json:"regression_case_id,omitempty"`
	Source           *regressionScoreSource `json:"source,omitempty"`
}

type regressionMetricDetail struct {
	Key              string     `json:"key"`
	State            string     `json:"state"`
	RegressionCaseID *uuid.UUID `json:"regression_case_id,omitempty"`
}

type regressionScoreSource struct {
	Kind      string `json:"kind"`
	Sequence  *int64 `json:"sequence,omitempty"`
	EventType string `json:"event_type,omitempty"`
}

type regressionDetailKey struct {
	CaseID uuid.UUID
	Key    string
}

func (m *ReleaseGateManager) evaluateRegressionRules(
	ctx context.Context,
	summary releasegate.ComparisonSummary,
	rules *releasegate.RegressionGateRules,
) (releasegate.RegressionGateOutcome, error) {
	if rules == nil {
		return releasegate.RegressionGateOutcome{Verdict: releasegate.VerdictPass}, nil
	}

	candidateCases, candidateWarning, err := m.loadRegressionCaseEvaluations(
		ctx,
		summary.CandidateRefs.RunAgentID,
		summary.CandidateRefs.EvaluationSpecID,
	)
	if err != nil {
		return releasegate.RegressionGateOutcome{}, err
	}
	if candidateWarning != "" {
		return releasegate.RegressionGateOutcome{
			Verdict:  releasegate.VerdictPass,
			Warnings: []string{candidateWarning},
		}, nil
	}

	baselineCases := []releasegate.RegressionCaseEvaluation(nil)
	baselineWarnings := make([]string, 0, 1)
	if rules.NoNewBlockingFailureVsBaseline {
		var baselineWarning string
		baselineCases, baselineWarning, err = m.loadRegressionCaseEvaluations(
			ctx,
			summary.BaselineRefs.RunAgentID,
			summary.BaselineRefs.EvaluationSpecID,
		)
		if err != nil {
			return releasegate.RegressionGateOutcome{}, err
		}
		if baselineWarning != "" {
			baselineWarnings = append(baselineWarnings, baselineWarning)
		}
	}

	outcome := releasegate.EvaluateRegressionGateRules(candidateCases, baselineCases, rules)
	outcome.Warnings = append(outcome.Warnings, baselineWarnings...)
	return outcome, nil
}

func (m *ReleaseGateManager) loadRegressionCaseEvaluations(
	ctx context.Context,
	runAgentID *uuid.UUID,
	evaluationSpecID *uuid.UUID,
) ([]releasegate.RegressionCaseEvaluation, string, error) {
	if runAgentID == nil || evaluationSpecID == nil {
		return nil, "regression scoring evidence unavailable for the selected comparison participant; skipped regression gate rules", nil
	}

	scorecard, err := m.repo.GetRunAgentScorecardByRunAgentID(ctx, *runAgentID)
	if err != nil {
		if errors.Is(err, repository.ErrRunAgentScorecardNotFound) {
			return nil, "regression scoring evidence unavailable for the selected comparison participant; skipped regression gate rules", nil
		}
		return nil, "", fmt.Errorf("load run-agent scorecard %s: %w", *runAgentID, err)
	}

	document, err := decodeRegressionScorecardDocument(scorecard.Scorecard)
	if err != nil {
		return nil, "", fmt.Errorf("decode run-agent scorecard %s: %w", *runAgentID, err)
	}

	judgeResults, err := m.repo.ListJudgeResultsByRunAgentAndEvaluationSpec(ctx, *runAgentID, *evaluationSpecID)
	if err != nil {
		return nil, "", fmt.Errorf("list judge results %s: %w", *runAgentID, err)
	}
	metricResults, err := m.repo.ListMetricResultsByRunAgentAndEvaluationSpec(ctx, *runAgentID, *evaluationSpecID)
	if err != nil {
		return nil, "", fmt.Errorf("list metric results %s: %w", *runAgentID, err)
	}

	validatorDetails := make(map[regressionDetailKey]regressionValidatorDetail, len(document.ValidatorDetails))
	for _, detail := range document.ValidatorDetails {
		if detail.RegressionCaseID == nil {
			continue
		}
		validatorDetails[regressionDetailKey{CaseID: *detail.RegressionCaseID, Key: detail.Key}] = detail
	}

	metricDetails := make(map[regressionDetailKey]regressionMetricDetail, len(document.MetricDetails))
	for _, detail := range document.MetricDetails {
		if detail.RegressionCaseID == nil {
			continue
		}
		metricDetails[regressionDetailKey{CaseID: *detail.RegressionCaseID, Key: detail.Key}] = detail
	}

	caseCache := make(map[uuid.UUID]repository.RegressionCase)
	evaluations := make(map[uuid.UUID]*releasegate.RegressionCaseEvaluation)
	fallbackRefs := make(map[uuid.UUID][]releasegate.RegressionReplayStepRef)

	ensureCase := func(caseID uuid.UUID) (*releasegate.RegressionCaseEvaluation, error) {
		if existing, ok := evaluations[caseID]; ok {
			return existing, nil
		}
		regressionCase, ok := caseCache[caseID]
		if !ok {
			var loadErr error
			regressionCase, loadErr = m.repo.GetRegressionCaseByID(ctx, caseID)
			if loadErr != nil {
				return nil, fmt.Errorf("load regression case %s: %w", caseID, loadErr)
			}
			caseCache[caseID] = regressionCase
			fallbackRefs[caseID] = promotionReplayRefs(regressionCase)
		}
		evaluation := &releasegate.RegressionCaseEvaluation{
			RegressionCaseID: caseID,
			SuiteID:          regressionCase.SuiteID,
			Severity:         string(regressionCase.Severity),
		}
		evaluations[caseID] = evaluation
		return evaluation, nil
	}

	for _, result := range judgeResults {
		if result.RegressionCaseID == nil {
			continue
		}
		evaluation, err := ensureCase(*result.RegressionCaseID)
		if err != nil {
			return nil, "", err
		}
		detail := validatorDetails[regressionDetailKey{CaseID: *result.RegressionCaseID, Key: result.JudgeKey}]
		if !judgeResultFailed(result, detail) {
			continue
		}
		evidence := releasegate.RegressionEvidenceRef{
			ScoringResultID:   result.ID,
			ScoringResultType: "judge_result",
			ReplayStepRefs:    replayRefsFromJudgeDetail(detail),
		}
		if len(evidence.ReplayStepRefs) == 0 {
			evidence.ReplayStepRefs = append([]releasegate.RegressionReplayStepRef(nil), fallbackRefs[*result.RegressionCaseID]...)
		}
		setRegressionFailure(evaluation, evidence)
	}

	for _, result := range metricResults {
		if result.RegressionCaseID == nil {
			continue
		}
		evaluation, err := ensureCase(*result.RegressionCaseID)
		if err != nil {
			return nil, "", err
		}
		detail := metricDetails[regressionDetailKey{CaseID: *result.RegressionCaseID, Key: result.MetricKey}]
		if !metricResultFailed(result, detail) {
			continue
		}
		evidence := releasegate.RegressionEvidenceRef{
			ScoringResultID:   result.ID,
			ScoringResultType: "metric_result",
			ReplayStepRefs:    append([]releasegate.RegressionReplayStepRef(nil), fallbackRefs[*result.RegressionCaseID]...),
		}
		setRegressionFailure(evaluation, evidence)
	}

	orderedCaseIDs := make([]string, 0, len(evaluations))
	for caseID := range evaluations {
		orderedCaseIDs = append(orderedCaseIDs, caseID.String())
	}
	sort.Strings(orderedCaseIDs)

	ordered := make([]releasegate.RegressionCaseEvaluation, 0, len(orderedCaseIDs))
	for _, rawID := range orderedCaseIDs {
		caseID := uuid.MustParse(rawID)
		ordered = append(ordered, *evaluations[caseID])
	}
	return ordered, "", nil
}

func decodeRegressionScorecardDocument(payload json.RawMessage) (regressionScorecardDocument, error) {
	document := regressionScorecardDocument{
		ValidatorDetails: []regressionValidatorDetail{},
		MetricDetails:    []regressionMetricDetail{},
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return document, nil
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		return regressionScorecardDocument{}, err
	}
	return document, nil
}

func judgeResultFailed(result repository.JudgeResultRecord, detail regressionValidatorDetail) bool {
	state := strings.TrimSpace(strings.ToLower(detail.State))
	if state == "error" || state == "unavailable" {
		return true
	}
	if result.Verdict == nil {
		return false
	}
	return strings.TrimSpace(strings.ToLower(*result.Verdict)) != "pass"
}

func metricResultFailed(result repository.MetricResultRecord, detail regressionMetricDetail) bool {
	state := strings.TrimSpace(strings.ToLower(detail.State))
	if state == "error" || state == "unavailable" || state == "fail" {
		return true
	}
	return result.BooleanValue != nil && !*result.BooleanValue
}

func replayRefsFromJudgeDetail(detail regressionValidatorDetail) []releasegate.RegressionReplayStepRef {
	if detail.Source == nil || detail.Source.Sequence == nil {
		return nil
	}
	return []releasegate.RegressionReplayStepRef{{
		SequenceNumber: *detail.Source.Sequence,
		EventType:      detail.Source.EventType,
		Kind:           detail.Source.Kind,
	}}
}

func promotionReplayRefs(regressionCase repository.RegressionCase) []releasegate.RegressionReplayStepRef {
	if regressionCase.LatestPromotion == nil || len(regressionCase.LatestPromotion.SourceEventRefs) == 0 {
		return nil
	}

	var refs []releasegate.RegressionReplayStepRef
	if err := json.Unmarshal(regressionCase.LatestPromotion.SourceEventRefs, &refs); err != nil {
		return nil
	}
	return refs
}

func setRegressionFailure(target *releasegate.RegressionCaseEvaluation, evidence releasegate.RegressionEvidenceRef) {
	target.Failed = true
	if target.Evidence == nil || (len(target.Evidence.ReplayStepRefs) == 0 && len(evidence.ReplayStepRefs) > 0) {
		copied := evidence
		if len(evidence.ReplayStepRefs) > 0 {
			copied.ReplayStepRefs = append([]releasegate.RegressionReplayStepRef(nil), evidence.ReplayStepRefs...)
		}
		target.Evidence = &copied
	}
}
