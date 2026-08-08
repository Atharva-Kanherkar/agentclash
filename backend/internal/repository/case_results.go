package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	repositorysqlc "github.com/agentclash/agentclash/backend/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const MaxTranscriptTextBytes = 64 * 1024

type CaseResult struct {
	ID                    uuid.UUID  `json:"id"`
	WorkspaceID           uuid.UUID  `json:"workspace_id"`
	OrganizationID        uuid.UUID  `json:"organization_id"`
	EvalSetID             *uuid.UUID `json:"eval_set_id,omitempty"`
	EvalSessionID         *uuid.UUID `json:"eval_session_id,omitempty"`
	RunID                 uuid.UUID  `json:"run_id"`
	RunAgentID            uuid.UUID  `json:"run_agent_id"`
	MatrixKey             string     `json:"matrix_key"`
	PackRef               string     `json:"pack_ref"`
	CaseKey               string     `json:"case_key"`
	AgentDeploymentID     *uuid.UUID `json:"agent_deployment_id,omitempty"`
	Model                 string     `json:"model,omitempty"`
	Score                 *float64   `json:"score,omitempty"`
	Correctness           *bool      `json:"correctness,omitempty"`
	Verdict               string     `json:"verdict,omitempty"`
	CostUSD               *float64   `json:"cost_usd,omitempty"`
	DurationMs            *int64     `json:"duration_ms,omitempty"`
	FailureClass          string     `json:"failure_class,omitempty"`
	TranscriptArtifactRef string     `json:"transcript_artifact_ref,omitempty"`
	TranscriptText        string     `json:"transcript_text,omitempty"`
	Snippet               string     `json:"snippet,omitempty"`
	Rank                  float64    `json:"rank,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type UpsertCaseResultParams struct {
	WorkspaceID           uuid.UUID
	OrganizationID        uuid.UUID
	EvalSetID             *uuid.UUID
	EvalSessionID         *uuid.UUID
	RunID                 uuid.UUID
	RunAgentID            uuid.UUID
	MatrixKey             string
	PackRef               string
	CaseKey               string
	AgentDeploymentID     *uuid.UUID
	Model                 string
	Score                 *float64
	Correctness           *bool
	Verdict               string
	CostUSD               *float64
	DurationMs            *int64
	FailureClass          string
	TranscriptArtifactRef string
	TranscriptText        string
}

type ListCaseResultsFilter struct {
	EvalSetID         uuid.UUID
	WorkspaceID       uuid.UUID
	AgentDeploymentID *uuid.UUID
	PackRef           *string
	Verdict           *string
	MinScore          *float64
	MaxScore          *float64
	CursorID          *uuid.UUID
	Limit             int32
	Query             string
}

type CaseResultAxisAggregate struct {
	PackRef           string  `json:"pack_ref"`
	AgentDeploymentID string  `json:"agent_deployment_id"`
	Model             string  `json:"model"`
	N                 int64   `json:"n"`
	MeanScore         float64 `json:"mean_score"`
	P50Score          float64 `json:"p50_score"`
	P95Score          float64 `json:"p95_score"`
	ScoreStddev       float64 `json:"score_stddev"`
	Wins              int64   `json:"wins"`
	Losses            int64   `json:"losses"`
}

type CaseResultCompareRow struct {
	MatrixKey         string     `json:"matrix_key"`
	PackRef           string     `json:"pack_ref"`
	CaseKey           string     `json:"case_key"`
	AgentDeploymentID *uuid.UUID `json:"agent_deployment_id,omitempty"`
	Model             string     `json:"model,omitempty"`
	Score             *float64   `json:"score,omitempty"`
	Verdict           string     `json:"verdict,omitempty"`
	RunID             uuid.UUID  `json:"run_id"`
}

func CapTranscriptText(text string) string {
	if len(text) <= MaxTranscriptTextBytes {
		return text
	}
	// Truncate on a rune boundary.
	truncated := text[:MaxTranscriptTextBytes]
	for !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

func HighlightSnippet(text, query string, window int) string {
	if window <= 0 {
		window = 80
	}
	lower := strings.ToLower(text)
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		if len(text) <= window*2 {
			return text
		}
		return text[:window*2] + "…"
	}
	idx := strings.Index(lower, q)
	if idx < 0 {
		if len(text) <= window*2 {
			return text
		}
		return text[:window*2] + "…"
	}
	start := idx - window
	if start < 0 {
		start = 0
	}
	end := idx + len(q) + window
	if end > len(text) {
		end = len(text)
	}
	snippet := text[start:end]
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(text) {
		snippet = snippet + "…"
	}
	return snippet
}

func (r *Repository) UpsertCaseResult(ctx context.Context, params UpsertCaseResultParams) (CaseResult, error) {
	row, err := r.queries.UpsertCaseResult(ctx, repositorysqlc.UpsertCaseResultParams{
		WorkspaceID:           params.WorkspaceID,
		OrganizationID:        params.OrganizationID,
		EvalSetID:             params.EvalSetID,
		EvalSessionID:         params.EvalSessionID,
		RunID:                 params.RunID,
		RunAgentID:            params.RunAgentID,
		MatrixKey:             params.MatrixKey,
		PackRef:               params.PackRef,
		CaseKey:               params.CaseKey,
		AgentDeploymentID:     params.AgentDeploymentID,
		Model:                 params.Model,
		Score:                 params.Score,
		Correctness:           params.Correctness,
		Verdict:               params.Verdict,
		CostUsd:               params.CostUSD,
		DurationMs:            params.DurationMs,
		FailureClass:          params.FailureClass,
		TranscriptArtifactRef: params.TranscriptArtifactRef,
		TranscriptText:        CapTranscriptText(params.TranscriptText),
	})
	if err != nil {
		return CaseResult{}, fmt.Errorf("upsert case result: %w", err)
	}
	return mapCaseResult(row, 0, ""), nil
}

func (r *Repository) GetCaseResultByID(ctx context.Context, id uuid.UUID) (CaseResult, error) {
	row, err := r.queries.GetCaseResultByID(ctx, repositorysqlc.GetCaseResultByIDParams{ID: id})
	if err != nil {
		return CaseResult{}, fmt.Errorf("get case result: %w", err)
	}
	return mapCaseResult(row, 0, ""), nil
}

func (r *Repository) ListCaseResults(ctx context.Context, filter ListCaseResultsFilter) ([]CaseResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	rows, err := r.queries.ListCaseResultsByEvalSetID(ctx, repositorysqlc.ListCaseResultsByEvalSetIDParams{
		EvalSetID:         uuidPtr(filter.EvalSetID),
		WorkspaceID:       filter.WorkspaceID,
		AgentDeploymentID: filter.AgentDeploymentID,
		PackRef:           filter.PackRef,
		Verdict:           filter.Verdict,
		MinScore:          filter.MinScore,
		MaxScore:          filter.MaxScore,
		CursorID:          filter.CursorID,
		LimitCount:        filter.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list case results: %w", err)
	}
	out := make([]CaseResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCaseResult(row, 0, ""))
	}
	return out, nil
}

func (r *Repository) SearchCaseResults(ctx context.Context, filter ListCaseResultsFilter) ([]CaseResult, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	rows, err := r.queries.SearchCaseResultsByEvalSetID(ctx, repositorysqlc.SearchCaseResultsByEvalSetIDParams{
		EvalSetID:         uuidPtr(filter.EvalSetID),
		WorkspaceID:       filter.WorkspaceID,
		Query:             filter.Query,
		AgentDeploymentID: filter.AgentDeploymentID,
		PackRef:           filter.PackRef,
		Verdict:           filter.Verdict,
		MinScore:          filter.MinScore,
		MaxScore:          filter.MaxScore,
		LimitCount:        filter.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search case results: %w", err)
	}
	out := make([]CaseResult, 0, len(rows))
	for _, row := range rows {
		cr := CaseResult{
			ID:                    row.ID,
			WorkspaceID:           row.WorkspaceID,
			OrganizationID:        row.OrganizationID,
			EvalSetID:             row.EvalSetID,
			EvalSessionID:         row.EvalSessionID,
			RunID:                 row.RunID,
			RunAgentID:            row.RunAgentID,
			MatrixKey:             row.MatrixKey,
			PackRef:               row.PackRef,
			CaseKey:               row.CaseKey,
			AgentDeploymentID:     row.AgentDeploymentID,
			Model:                 row.Model,
			Score:                 row.Score,
			Correctness:           row.Correctness,
			Verdict:               row.Verdict,
			CostUSD:               row.CostUsd,
			DurationMs:            row.DurationMs,
			FailureClass:          row.FailureClass,
			TranscriptArtifactRef: row.TranscriptArtifactRef,
			TranscriptText:        row.TranscriptText,
			Rank:                  float64FromInterface(row.Rank),
			CreatedAt:             row.CreatedAt.Time,
			UpdatedAt:             row.UpdatedAt.Time,
		}
		cr.Snippet = HighlightSnippet(row.TranscriptText, filter.Query, 80)
		out = append(out, cr)
	}
	return out, nil
}

func (r *Repository) ListCaseResultsForExport(ctx context.Context, workspaceID, evalSetID uuid.UUID, cursor *uuid.UUID, limit int32) ([]CaseResult, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.queries.ListCaseResultsForExport(ctx, repositorysqlc.ListCaseResultsForExportParams{
		EvalSetID:   uuidPtr(evalSetID),
		WorkspaceID: workspaceID,
		CursorID:    cursor,
		LimitCount:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("export case results: %w", err)
	}
	out := make([]CaseResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCaseResult(row, 0, ""))
	}
	return out, nil
}

func (r *Repository) AggregateCaseResults(ctx context.Context, workspaceID, evalSetID uuid.UUID) ([]CaseResultAxisAggregate, error) {
	rows, err := r.queries.AggregateCaseResultsByEvalSet(ctx, repositorysqlc.AggregateCaseResultsByEvalSetParams{
		EvalSetID:   uuidPtr(evalSetID),
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("aggregate case results: %w", err)
	}
	out := make([]CaseResultAxisAggregate, 0, len(rows))
	for _, row := range rows {
		out = append(out, CaseResultAxisAggregate{
			PackRef:           row.PackRef,
			AgentDeploymentID: fmt.Sprint(row.AgentDeploymentID),
			Model:             row.Model,
			N:                 row.N,
			MeanScore:         row.MeanScore,
			P50Score:          row.P50Score,
			P95Score:          row.P95Score,
			ScoreStddev:       row.ScoreStddev,
			Wins:              row.Wins,
			Losses:            row.Losses,
		})
	}
	return out, nil
}

func (r *Repository) ListCaseResultsForCompare(ctx context.Context, workspaceID, evalSetID uuid.UUID) ([]CaseResultCompareRow, error) {
	rows, err := r.queries.ListCaseResultsByEvalSetForCompare(ctx, repositorysqlc.ListCaseResultsByEvalSetForCompareParams{
		EvalSetID:   uuidPtr(evalSetID),
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("list case results for compare: %w", err)
	}
	out := make([]CaseResultCompareRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, CaseResultCompareRow{
			MatrixKey:         row.MatrixKey,
			PackRef:           row.PackRef,
			CaseKey:           row.CaseKey,
			AgentDeploymentID: row.AgentDeploymentID,
			Model:             row.Model,
			Score:             row.Score,
			Verdict:           row.Verdict,
			RunID:             row.RunID,
		})
	}
	return out, nil
}

func mapCaseResult(row repositorysqlc.CaseResult, rank float64, snippet string) CaseResult {
	return CaseResult{
		ID:                    row.ID,
		WorkspaceID:           row.WorkspaceID,
		OrganizationID:        row.OrganizationID,
		EvalSetID:             row.EvalSetID,
		EvalSessionID:         row.EvalSessionID,
		RunID:                 row.RunID,
		RunAgentID:            row.RunAgentID,
		MatrixKey:             row.MatrixKey,
		PackRef:               row.PackRef,
		CaseKey:               row.CaseKey,
		AgentDeploymentID:     row.AgentDeploymentID,
		Model:                 row.Model,
		Score:                 row.Score,
		Correctness:           row.Correctness,
		Verdict:               row.Verdict,
		CostUSD:               row.CostUsd,
		DurationMs:            row.DurationMs,
		FailureClass:          row.FailureClass,
		TranscriptArtifactRef: row.TranscriptArtifactRef,
		TranscriptText:        row.TranscriptText,
		Rank:                  rank,
		Snippet:               snippet,
		CreatedAt:             row.CreatedAt.Time,
		UpdatedAt:             row.UpdatedAt.Time,
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func float64FromInterface(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case pgtype.Float8:
		if x.Valid {
			return x.Float64
		}
	case *float64:
		if x != nil {
			return *x
		}
	}
	return 0
}
