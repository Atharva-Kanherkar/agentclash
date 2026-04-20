package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/provider"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/google/uuid"
)

const rankingInsightsTimeout = 45 * time.Second

type GenerateRunRankingInsightsInput struct {
	ProviderAccountID uuid.UUID
	ModelAliasID      uuid.UUID
}

type GenerateRunRankingInsightsResult struct {
	Run      domain.Run
	Insights runRankingInsightsResponse
}

type runRankingInsightsResponse struct {
	GeneratedAt         time.Time                        `json:"generated_at"`
	GroundingScope      string                           `json:"grounding_scope"`
	ProviderKey         string                           `json:"provider_key"`
	ProviderModelID     string                           `json:"provider_model_id"`
	RecommendedWinner   runRankingInsightCandidate       `json:"recommended_winner"`
	WhyItWon            string                           `json:"why_it_won"`
	Tradeoffs           []string                         `json:"tradeoffs"`
	BestForReliability  *runRankingInsightRecommendation `json:"best_for_reliability,omitempty"`
	BestForCost         *runRankingInsightRecommendation `json:"best_for_cost,omitempty"`
	BestForLatency      *runRankingInsightRecommendation `json:"best_for_latency,omitempty"`
	ModelSummaries      []runRankingModelInsight         `json:"model_summaries"`
	RecommendedNextStep string                           `json:"recommended_next_step"`
	ConfidenceNotes     string                           `json:"confidence_notes"`
}

type runRankingInsightCandidate struct {
	RunAgentID uuid.UUID `json:"run_agent_id"`
	Label      string    `json:"label"`
}

type runRankingInsightRecommendation struct {
	RunAgentID uuid.UUID `json:"run_agent_id"`
	Label      string    `json:"label"`
	Reason     string    `json:"reason"`
}

type runRankingModelInsight struct {
	RunAgentID         uuid.UUID `json:"run_agent_id"`
	Label              string    `json:"label"`
	StrongestDimension string    `json:"strongest_dimension"`
	WeakestDimension   string    `json:"weakest_dimension"`
	Summary            string    `json:"summary"`
}

type createRunRankingInsightsRequest struct {
	ProviderAccountID string `json:"provider_account_id"`
	ModelAliasID      string `json:"model_alias_id"`
}

type RunRankingInsightsValidationError struct {
	Code    string
	Message string
}

func (e RunRankingInsightsValidationError) Error() string {
	return e.Message
}

func (m *RunReadManager) GenerateRunRankingInsights(ctx context.Context, caller Caller, runID uuid.UUID, input GenerateRunRankingInsightsInput) (GenerateRunRankingInsightsResult, error) {
	if m.insightsClient == nil {
		return GenerateRunRankingInsightsResult{}, fmt.Errorf("ranking insights provider client is not configured")
	}

	run, err := m.repo.GetRunByID(ctx, runID)
	if err != nil {
		return GenerateRunRankingInsightsResult{}, err
	}
	if err := m.authorizer.AuthorizeWorkspace(ctx, caller, run.WorkspaceID); err != nil {
		return GenerateRunRankingInsightsResult{}, err
	}
	if run.Status != domain.RunStatusCompleted {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_run_status",
			Message: "ranking insights are only available for completed runs",
		}
	}

	runAgents, err := m.repo.ListRunAgentsByRunID(ctx, runID)
	if err != nil {
		return GenerateRunRankingInsightsResult{}, fmt.Errorf("list run agents: %w", err)
	}
	if len(runAgents) < 2 || run.ExecutionMode != "comparison" {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_run_for_insights",
			Message: "ranking insights require a completed multi-agent run",
		}
	}

	rankingResult, err := m.GetRunRanking(ctx, caller, runID, GetRunRankingInput{})
	if err != nil {
		return GenerateRunRankingInsightsResult{}, err
	}
	if rankingResult.State != RankingReadStateReady || rankingResult.Ranking == nil {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "ranking_unavailable",
			Message: "ranking insights require an available run ranking",
		}
	}

	providerAccount, err := m.repo.GetProviderAccountByID(ctx, input.ProviderAccountID)
	if err != nil {
		return GenerateRunRankingInsightsResult{}, err
	}
	if providerAccount.WorkspaceID == nil || *providerAccount.WorkspaceID != run.WorkspaceID {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_provider_account_id",
			Message: "provider_account_id must belong to the run workspace",
		}
	}
	if providerAccount.Status != "active" {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_provider_account_id",
			Message: "provider_account_id must reference an active provider account",
		}
	}

	modelAlias, err := m.repo.GetModelAliasByID(ctx, input.ModelAliasID)
	if err != nil {
		return GenerateRunRankingInsightsResult{}, err
	}
	if modelAlias.WorkspaceID == nil || *modelAlias.WorkspaceID != run.WorkspaceID {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_model_alias_id",
			Message: "model_alias_id must belong to the run workspace",
		}
	}
	if modelAlias.Status != "active" {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_model_alias_id",
			Message: "model_alias_id must reference an active model alias",
		}
	}
	if modelAlias.ProviderAccountID != nil && *modelAlias.ProviderAccountID != providerAccount.ID {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_model_alias_id",
			Message: "model_alias_id must be compatible with the selected provider account",
		}
	}

	modelCatalogEntry, err := m.repo.GetModelCatalogEntryByID(ctx, modelAlias.ModelCatalogEntryID)
	if err != nil {
		return GenerateRunRankingInsightsResult{}, err
	}
	if modelCatalogEntry.ProviderKey != providerAccount.ProviderKey {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_model_alias_id",
			Message: "model_alias_id provider does not match the selected provider account",
		}
	}

	invokeCtx := ctx
	if strings.HasPrefix(providerAccount.CredentialReference, "workspace-secret://") {
		secrets, loadErr := m.repo.LoadWorkspaceSecrets(ctx, run.WorkspaceID)
		if loadErr != nil {
			return GenerateRunRankingInsightsResult{}, fmt.Errorf("load workspace secrets: %w", loadErr)
		}
		invokeCtx = provider.WithWorkspaceSecrets(invokeCtx, secrets)
	}

	promptPayload, err := buildRunRankingInsightsPrompt(run, rankingResult.Ranking, rankingResult.Scorecard)
	if err != nil {
		return GenerateRunRankingInsightsResult{}, fmt.Errorf("build ranking insights prompt: %w", err)
	}

	response, err := m.insightsClient.InvokeModel(invokeCtx, provider.Request{
		ProviderKey:         providerAccount.ProviderKey,
		ProviderAccountID:   providerAccount.ID.String(),
		CredentialReference: providerAccount.CredentialReference,
		Model:               modelCatalogEntry.ProviderModelID,
		StepTimeout:         rankingInsightsTimeout,
		Messages: []provider.Message{
			{
				Role: "system",
				Content: strings.TrimSpace(`
You are an evaluation analyst for AgentClash.

Use only the run ranking data provided by the user. Do not invent missing metrics,
external model knowledge, or web results. Keep the analysis concise, concrete,
and grounded in the supplied run evidence.

Return JSON only. Do not wrap the JSON in markdown fences.
`),
			},
			{
				Role:    "user",
				Content: promptPayload,
			},
		},
		Metadata: mustMarshalRunRankingInsightsJSON(map[string]any{
			"run_id":              run.ID,
			"workspace_id":        run.WorkspaceID,
			"provider_account_id": providerAccount.ID,
			"model_alias_id":      modelAlias.ID,
			"feature":             "run_ranking_insights",
			"grounding_scope":     "current_run_only",
		}),
	})
	if err != nil {
		return GenerateRunRankingInsightsResult{}, err
	}

	insights, err := parseRunRankingInsights(response.OutputText, rankingResult.Ranking.Items)
	if err != nil {
		return GenerateRunRankingInsightsResult{}, RunRankingInsightsValidationError{
			Code:    "invalid_insights_output",
			Message: fmt.Sprintf("ranking insights model returned invalid output: %v", err),
		}
	}
	insights.GeneratedAt = m.now().UTC()
	insights.GroundingScope = "current_run_only"
	insights.ProviderKey = providerAccount.ProviderKey
	insights.ProviderModelID = response.ProviderModelID
	if strings.TrimSpace(insights.ProviderModelID) == "" {
		insights.ProviderModelID = modelCatalogEntry.ProviderModelID
	}

	return GenerateRunRankingInsightsResult{
		Run:      run,
		Insights: insights,
	}, nil
}

func mustMarshalRunRankingInsightsJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func buildRunRankingInsightsPrompt(run domain.Run, ranking *runRankingPayload, scorecard *repository.RunScorecard) (string, error) {
	payload := map[string]any{
		"task": "Analyze this completed multi-agent run and recommend the best model for this run only. Explain the winner, major tradeoffs, and the next experiment. Return JSON with this shape exactly: {\"recommended_winner\":{\"run_agent_id\":\"<uuid>\",\"label\":\"<label>\"},\"why_it_won\":\"...\",\"tradeoffs\":[\"...\"],\"best_for_reliability\":{\"run_agent_id\":\"<uuid>\",\"label\":\"<label>\",\"reason\":\"...\"},\"best_for_cost\":{\"run_agent_id\":\"<uuid>\",\"label\":\"<label>\",\"reason\":\"...\"},\"best_for_latency\":{\"run_agent_id\":\"<uuid>\",\"label\":\"<label>\",\"reason\":\"...\"},\"model_summaries\":[{\"run_agent_id\":\"<uuid>\",\"label\":\"<label>\",\"strongest_dimension\":\"...\",\"weakest_dimension\":\"...\",\"summary\":\"...\"}],\"recommended_next_step\":\"...\",\"confidence_notes\":\"...\"}.",
		"constraints": []string{
			"Only use the run data supplied below.",
			"Treat the result as advisory and grounded in current-run evidence only.",
			"If the signal is mixed or close, say so in confidence_notes.",
			"Do not mention web research or external models.",
		},
		"run": map[string]any{
			"id":             run.ID,
			"name":           run.Name,
			"status":         run.Status,
			"execution_mode": run.ExecutionMode,
		},
		"ranking":   ranking,
		"scorecard": scorecard,
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func parseRunRankingInsights(raw string, items []runRankingItemResponse) (runRankingInsightsResponse, error) {
	jsonPayload, err := extractJSONObject(raw)
	if err != nil {
		return runRankingInsightsResponse{}, err
	}

	var insights runRankingInsightsResponse
	if err := json.Unmarshal([]byte(jsonPayload), &insights); err != nil {
		return runRankingInsightsResponse{}, err
	}

	return validateRunRankingInsights(insights, items)
}

func extractJSONObject(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end < start {
		return "", errors.New("response did not contain a JSON object")
	}
	return trimmed[start : end+1], nil
}

func validateRunRankingInsights(insights runRankingInsightsResponse, items []runRankingItemResponse) (runRankingInsightsResponse, error) {
	byID := make(map[uuid.UUID]runRankingItemResponse, len(items))
	for _, item := range items {
		byID[item.RunAgentID] = item
	}

	winner, ok := byID[insights.RecommendedWinner.RunAgentID]
	if !ok {
		return runRankingInsightsResponse{}, errors.New("recommended_winner.run_agent_id is not part of this run")
	}
	if strings.TrimSpace(insights.RecommendedWinner.Label) == "" {
		insights.RecommendedWinner.Label = winner.Label
	}
	if strings.TrimSpace(insights.WhyItWon) == "" {
		return runRankingInsightsResponse{}, errors.New("why_it_won is required")
	}
	if len(insights.Tradeoffs) == 0 {
		return runRankingInsightsResponse{}, errors.New("tradeoffs must contain at least one item")
	}
	if strings.TrimSpace(insights.RecommendedNextStep) == "" {
		return runRankingInsightsResponse{}, errors.New("recommended_next_step is required")
	}
	if strings.TrimSpace(insights.ConfidenceNotes) == "" {
		return runRankingInsightsResponse{}, errors.New("confidence_notes is required")
	}

	for idx, summary := range insights.ModelSummaries {
		item, ok := byID[summary.RunAgentID]
		if !ok {
			return runRankingInsightsResponse{}, fmt.Errorf("model_summaries[%d].run_agent_id is not part of this run", idx)
		}
		if strings.TrimSpace(summary.Label) == "" {
			insights.ModelSummaries[idx].Label = item.Label
		}
		if strings.TrimSpace(summary.Summary) == "" {
			return runRankingInsightsResponse{}, fmt.Errorf("model_summaries[%d].summary is required", idx)
		}
	}

	for _, rec := range []*runRankingInsightRecommendation{
		insights.BestForReliability,
		insights.BestForCost,
		insights.BestForLatency,
	} {
		if rec == nil {
			continue
		}
		item, ok := byID[rec.RunAgentID]
		if !ok {
			return runRankingInsightsResponse{}, errors.New("best_for_* recommendation references a run agent outside this run")
		}
		if strings.TrimSpace(rec.Label) == "" {
			rec.Label = item.Label
		}
	}

	return insights, nil
}

func createRunRankingInsightsHandler(logger *slog.Logger, service RunReadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}

		runID, err := runIDFromURLParam("runID")(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_run_id", err.Error())
			return
		}
		if err := requireJSONContentType(r); err != nil {
			writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", err.Error())
			return
		}

		var body createRunRankingInsightsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
			return
		}

		providerAccountID, err := uuid.Parse(strings.TrimSpace(body.ProviderAccountID))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_provider_account_id", "provider_account_id must be a valid UUID")
			return
		}
		modelAliasID, err := uuid.Parse(strings.TrimSpace(body.ModelAliasID))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_model_alias_id", "model_alias_id must be a valid UUID")
			return
		}

		result, err := service.GenerateRunRankingInsights(r.Context(), caller, runID, GenerateRunRankingInsightsInput{
			ProviderAccountID: providerAccountID,
			ModelAliasID:      modelAliasID,
		})
		if err != nil {
			var validationErr RunRankingInsightsValidationError
			switch {
			case errors.As(err, &validationErr):
				writeError(w, http.StatusBadRequest, validationErr.Code, validationErr.Message)
			case errors.Is(err, repository.ErrRunNotFound):
				writeError(w, http.StatusNotFound, "run_not_found", "run not found")
			case errors.Is(err, repository.ErrProviderAccountNotFound):
				writeError(w, http.StatusBadRequest, "invalid_provider_account_id", "provider_account_id must reference an active provider account")
			case errors.Is(err, repository.ErrModelAliasNotFound):
				writeError(w, http.StatusBadRequest, "invalid_model_alias_id", "model_alias_id must reference an active model alias")
			case errors.Is(err, repository.ErrModelCatalogNotFound):
				writeError(w, http.StatusBadRequest, "invalid_model_alias_id", "model_alias_id must reference a valid model catalog entry")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				var providerFailure provider.Failure
				if errors.As(err, &providerFailure) {
					writeError(w, http.StatusBadGateway, "ranking_insights_provider_error", providerFailure.Error())
					return
				}
				logger.Error("create run ranking insights request failed",
					"method", r.Method,
					"path", r.URL.Path,
					"run_id", runID,
					"error", err,
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusOK, result.Insights)
	}
}
