package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ReplayReadRepository interface {
	GetRunAgentByID(ctx context.Context, id uuid.UUID) (domain.RunAgent, error)
	GetRunAgentReplayByRunAgentID(ctx context.Context, runAgentID uuid.UUID) (repository.RunAgentReplay, error)
	GetRunAgentScorecardByRunAgentID(ctx context.Context, runAgentID uuid.UUID) (repository.RunAgentScorecard, error)
}

type ReplayReadService interface {
	GetRunAgentReplay(ctx context.Context, caller Caller, runAgentID uuid.UUID) (GetRunAgentReplayResult, error)
	GetRunAgentScorecard(ctx context.Context, caller Caller, runAgentID uuid.UUID) (GetRunAgentScorecardResult, error)
}

type GetRunAgentReplayResult struct {
	RunAgent domain.RunAgent
	Replay   repository.RunAgentReplay
}

type GetRunAgentScorecardResult struct {
	RunAgent  domain.RunAgent
	Scorecard repository.RunAgentScorecard
}

type ReplayReadManager struct {
	authorizer WorkspaceAuthorizer
	repo       ReplayReadRepository
}

func NewReplayReadManager(authorizer WorkspaceAuthorizer, repo ReplayReadRepository) *ReplayReadManager {
	return &ReplayReadManager{
		authorizer: authorizer,
		repo:       repo,
	}
}

func (m *ReplayReadManager) GetRunAgentReplay(ctx context.Context, caller Caller, runAgentID uuid.UUID) (GetRunAgentReplayResult, error) {
	runAgent, err := m.repo.GetRunAgentByID(ctx, runAgentID)
	if err != nil {
		return GetRunAgentReplayResult{}, err
	}
	if err := m.authorizer.AuthorizeWorkspace(ctx, caller, runAgent.WorkspaceID); err != nil {
		return GetRunAgentReplayResult{}, err
	}

	replay, err := m.repo.GetRunAgentReplayByRunAgentID(ctx, runAgentID)
	if err != nil {
		return GetRunAgentReplayResult{}, err
	}

	return GetRunAgentReplayResult{
		RunAgent: runAgent,
		Replay:   replay,
	}, nil
}

func (m *ReplayReadManager) GetRunAgentScorecard(ctx context.Context, caller Caller, runAgentID uuid.UUID) (GetRunAgentScorecardResult, error) {
	runAgent, err := m.repo.GetRunAgentByID(ctx, runAgentID)
	if err != nil {
		return GetRunAgentScorecardResult{}, err
	}
	if err := m.authorizer.AuthorizeWorkspace(ctx, caller, runAgent.WorkspaceID); err != nil {
		return GetRunAgentScorecardResult{}, err
	}

	scorecard, err := m.repo.GetRunAgentScorecardByRunAgentID(ctx, runAgentID)
	if err != nil {
		return GetRunAgentScorecardResult{}, err
	}

	return GetRunAgentScorecardResult{
		RunAgent:  runAgent,
		Scorecard: scorecard,
	}, nil
}

type getRunAgentReplayResponse struct {
	ID                   uuid.UUID       `json:"id"`
	RunAgentID           uuid.UUID       `json:"run_agent_id"`
	RunID                uuid.UUID       `json:"run_id"`
	ArtifactID           *uuid.UUID      `json:"artifact_id,omitempty"`
	Summary              json.RawMessage `json:"summary"`
	LatestSequenceNumber *int64          `json:"latest_sequence_number,omitempty"`
	EventCount           int64           `json:"event_count"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type getRunAgentScorecardResponse struct {
	ID               uuid.UUID       `json:"id"`
	RunAgentID       uuid.UUID       `json:"run_agent_id"`
	RunID            uuid.UUID       `json:"run_id"`
	EvaluationSpecID uuid.UUID       `json:"evaluation_spec_id"`
	OverallScore     *float64        `json:"overall_score,omitempty"`
	CorrectnessScore *float64        `json:"correctness_score,omitempty"`
	ReliabilityScore *float64        `json:"reliability_score,omitempty"`
	LatencyScore     *float64        `json:"latency_score,omitempty"`
	CostScore        *float64        `json:"cost_score,omitempty"`
	Scorecard        json.RawMessage `json:"scorecard"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func getRunAgentReplayHandler(logger *slog.Logger, service ReplayReadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}

		runAgentID, err := runAgentIDFromURLParam("runAgentID")(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_run_agent_id", err.Error())
			return
		}

		result, err := service.GetRunAgentReplay(r.Context(), caller, runAgentID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrRunAgentNotFound):
				writeError(w, http.StatusNotFound, "run_agent_not_found", "run agent not found")
			case errors.Is(err, repository.ErrRunAgentReplayNotFound):
				writeError(w, http.StatusNotFound, "replay_not_found", "replay not found")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("get run-agent replay request failed",
					"method", r.Method,
					"path", r.URL.Path,
					"run_agent_id", runAgentID,
					"error", err,
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusOK, buildRunAgentReplayResponse(result.RunAgent, result.Replay))
	}
}

func getRunAgentScorecardHandler(logger *slog.Logger, service ReplayReadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}

		runAgentID, err := runAgentIDFromURLParam("runAgentID")(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_run_agent_id", err.Error())
			return
		}

		result, err := service.GetRunAgentScorecard(r.Context(), caller, runAgentID)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrRunAgentNotFound):
				writeError(w, http.StatusNotFound, "run_agent_not_found", "run agent not found")
			case errors.Is(err, repository.ErrRunAgentScorecardNotFound):
				writeError(w, http.StatusNotFound, "scorecard_not_found", "scorecard not found")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("get run-agent scorecard request failed",
					"method", r.Method,
					"path", r.URL.Path,
					"run_agent_id", runAgentID,
					"error", err,
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusOK, buildRunAgentScorecardResponse(result.RunAgent, result.Scorecard))
	}
}

func buildRunAgentReplayResponse(runAgent domain.RunAgent, replay repository.RunAgentReplay) getRunAgentReplayResponse {
	return getRunAgentReplayResponse{
		ID:                   replay.ID,
		RunAgentID:           replay.RunAgentID,
		RunID:                runAgent.RunID,
		ArtifactID:           replay.ArtifactID,
		Summary:              replay.Summary,
		LatestSequenceNumber: replay.LatestSequenceNumber,
		EventCount:           replay.EventCount,
		CreatedAt:            replay.CreatedAt,
		UpdatedAt:            replay.UpdatedAt,
	}
}

func buildRunAgentScorecardResponse(runAgent domain.RunAgent, scorecard repository.RunAgentScorecard) getRunAgentScorecardResponse {
	return getRunAgentScorecardResponse{
		ID:               scorecard.ID,
		RunAgentID:       scorecard.RunAgentID,
		RunID:            runAgent.RunID,
		EvaluationSpecID: scorecard.EvaluationSpecID,
		OverallScore:     scorecard.OverallScore,
		CorrectnessScore: scorecard.CorrectnessScore,
		ReliabilityScore: scorecard.ReliabilityScore,
		LatencyScore:     scorecard.LatencyScore,
		CostScore:        scorecard.CostScore,
		Scorecard:        scorecard.Scorecard,
		CreatedAt:        scorecard.CreatedAt,
		UpdatedAt:        scorecard.UpdatedAt,
	}
}

func runAgentIDFromURLParam(name string) func(*http.Request) (uuid.UUID, error) {
	return func(r *http.Request) (uuid.UUID, error) {
		raw := chi.URLParam(r, name)
		if raw == "" {
			return uuid.Nil, errors.New("run agent id is required")
		}

		runAgentID, err := uuid.Parse(raw)
		if err != nil {
			return uuid.Nil, errors.New("run agent id must be a valid UUID")
		}

		return runAgentID, nil
	}
}
