package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/challengepack"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/scoring"
	"github.com/google/uuid"
)

type EvaluateRunAgentParams struct {
	RunAgentID       uuid.UUID
	EvaluationSpecID uuid.UUID
}

func (r *Repository) EvaluateRunAgent(ctx context.Context, params EvaluateRunAgentParams) (scoring.RunAgentEvaluation, error) {
	executionContext, err := r.GetRunAgentExecutionContextByID(ctx, params.RunAgentID)
	if err != nil {
		return scoring.RunAgentEvaluation{}, fmt.Errorf("load run-agent execution context: %w", err)
	}

	evaluationSpec, err := r.GetEvaluationSpecByID(ctx, params.EvaluationSpecID)
	if err != nil {
		return scoring.RunAgentEvaluation{}, fmt.Errorf("load evaluation spec: %w", err)
	}
	if evaluationSpec.ChallengePackVersionID != executionContext.ChallengePackVersion.ID {
		return scoring.RunAgentEvaluation{}, fmt.Errorf(
			"evaluation spec %s belongs to challenge pack version %s, not run-agent challenge pack version %s",
			evaluationSpec.ID,
			evaluationSpec.ChallengePackVersionID,
			executionContext.ChallengePackVersion.ID,
		)
	}

	spec, err := scoring.DecodeDefinition(evaluationSpec.Definition)
	if err != nil {
		return scoring.RunAgentEvaluation{}, fmt.Errorf("decode evaluation spec definition: %w", err)
	}

	events, err := r.ListRunEventsByRunAgentID(ctx, params.RunAgentID)
	if err != nil {
		return scoring.RunAgentEvaluation{}, fmt.Errorf("list canonical run events: %w", err)
	}

	evaluation, err := scoring.EvaluateRunAgent(mapEvaluationInput(params.EvaluationSpecID, executionContext, events), spec)
	if err != nil {
		return scoring.RunAgentEvaluation{}, fmt.Errorf("evaluate run-agent: %w", err)
	}
	if err := r.StoreRunAgentEvaluationResults(ctx, evaluation); err != nil {
		return scoring.RunAgentEvaluation{}, fmt.Errorf("store run-agent evaluation results: %w", err)
	}

	return evaluation, nil
}

func mapEvaluationInput(evaluationSpecID uuid.UUID, executionContext RunAgentExecutionContext, events []RunEvent) scoring.EvaluationInput {
	convertedEvents := make([]scoring.Event, 0, len(events))
	for _, event := range events {
		convertedEvents = append(convertedEvents, scoring.Event{
			Type:       string(event.EventType),
			Source:     string(event.Source),
			OccurredAt: event.OccurredAt,
			Payload:    cloneJSON(event.Payload),
		})
	}

	challengeInputs := make([]scoring.EvidenceInput, 0)
	manifestAssets := evaluationManifestAssets(executionContext.ChallengePackVersion.Manifest)
	if executionContext.ChallengeInputSet != nil {
		for _, item := range executionContext.ChallengeInputSet.Cases {
			challengeInputs = append(challengeInputs, scoring.EvidenceInput{
				ChallengeIdentityID: item.ChallengeIdentityID,
				ChallengeKey:        item.ChallengeKey,
				CaseKey:             item.CaseKey,
				ItemKey:             item.ItemKey,
				Payload:             cloneJSON(item.Payload),
				Inputs:              evaluationCaseValues(item.Inputs),
				Expectations:        evaluationExpectationValues(item.Expectations),
				Artifacts:           evaluationCaseArtifacts(item, manifestAssets),
			})
		}
	}

	return scoring.EvaluationInput{
		RunAgentID:       executionContext.RunAgent.ID,
		EvaluationSpecID: evaluationSpecID,
		ChallengeInputs:  challengeInputs,
		Events:           convertedEvents,
	}
}

func evaluationManifestAssets(manifest json.RawMessage) map[string]scoring.EvidenceArtifact {
	var decoded struct {
		Version struct {
			Assets []challengepack.AssetReference `json:"assets"`
		} `json:"version"`
	}
	if err := json.Unmarshal(manifest, &decoded); err != nil {
		return map[string]scoring.EvidenceArtifact{}
	}

	assets := make(map[string]scoring.EvidenceArtifact, len(decoded.Version.Assets))
	for _, asset := range decoded.Version.Assets {
		if asset.Key == "" {
			continue
		}
		assets[asset.Key] = scoring.EvidenceArtifact{
			Key:       asset.Key,
			Kind:      asset.Kind,
			Path:      asset.Path,
			MediaType: asset.MediaType,
		}
	}
	return assets
}

func evaluationCaseArtifacts(item ChallengeCaseExecutionContext, manifestAssets map[string]scoring.EvidenceArtifact) map[string]scoring.EvidenceArtifact {
	artifacts := make(map[string]scoring.EvidenceArtifact)
	for _, asset := range item.Assets {
		if asset.Key == "" {
			continue
		}
		artifacts[asset.Key] = scoring.EvidenceArtifact{
			Key:       asset.Key,
			Kind:      asset.Kind,
			Path:      asset.Path,
			MediaType: asset.MediaType,
		}
	}
	for _, ref := range item.Artifacts {
		if artifact, ok := manifestAssets[ref.Key]; ok {
			artifacts[ref.Key] = artifact
		}
	}
	for _, input := range item.Inputs {
		if artifact, ok := manifestAssets[input.ArtifactKey]; ok {
			artifacts[input.ArtifactKey] = artifact
		}
	}
	for _, expectation := range item.Expectations {
		if artifact, ok := manifestAssets[expectation.ArtifactKey]; ok {
			artifacts[expectation.ArtifactKey] = artifact
		}
		if key, ok := artifactKeyFromEvidenceSource(expectation.Source); ok {
			if artifact, exists := manifestAssets[key]; exists {
				artifacts[key] = artifact
			}
		}
	}
	return artifacts
}

func evaluationCaseValues(inputs []challengepack.CaseInput) map[string]scoring.EvidenceValue {
	values := make(map[string]scoring.EvidenceValue, len(inputs))
	for _, input := range inputs {
		values[input.Key] = scoring.EvidenceValue{
			Kind:        input.Kind,
			Value:       evaluationMarshalEvidenceValue(input.Value),
			ArtifactKey: input.ArtifactKey,
			Path:        input.Path,
		}
	}
	return values
}

func evaluationExpectationValues(expectations []challengepack.CaseExpectation) map[string]scoring.EvidenceValue {
	values := make(map[string]scoring.EvidenceValue, len(expectations))
	for _, expectation := range expectations {
		values[expectation.Key] = scoring.EvidenceValue{
			Kind:        expectation.Kind,
			Value:       evaluationMarshalEvidenceValue(expectation.Value),
			ArtifactKey: expectation.ArtifactKey,
			Source:      expectation.Source,
		}
	}
	return values
}

func evaluationMarshalEvidenceValue(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return encoded
}

func artifactKeyFromEvidenceSource(source string) (string, bool) {
	trimmed := strings.TrimSpace(source)
	if !strings.HasPrefix(trimmed, "artifact:") {
		return "", false
	}
	key := strings.TrimSpace(strings.TrimPrefix(trimmed, "artifact:"))
	if key == "" {
		return "", false
	}
	return key, true
}
