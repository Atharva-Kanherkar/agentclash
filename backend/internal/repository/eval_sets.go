package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	repositorysqlc "github.com/agentclash/agentclash/backend/internal/repository/sqlc"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrEvalSetNotFound = errors.New("eval set not found")

type EvalSet struct {
	ID                uuid.UUID            `json:"id"`
	WorkspaceID       uuid.UUID            `json:"workspace_id"`
	OrganizationID    uuid.UUID            `json:"organization_id"`
	Name              string               `json:"name"`
	Status            domain.EvalSetStatus `json:"status"`
	Manifest          json.RawMessage      `json:"manifest,omitempty"`
	Expansion         json.RawMessage      `json:"expansion,omitempty"`
	MaxConcurrentRuns int32                `json:"max_concurrent_runs"`
	BudgetUSD         *float64             `json:"budget_usd,omitempty"`
	SpentUSD          float64              `json:"spent_usd"`
	EstimatedCostUSD  *float64             `json:"estimated_cost_usd,omitempty"`
	CaseFanout        bool                 `json:"case_fanout"`
	CombinationCount  int32                `json:"combination_count"`
	CreatedByUserID   *uuid.UUID           `json:"created_by_user_id,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	StartedAt         *time.Time           `json:"started_at,omitempty"`
	FinishedAt        *time.Time           `json:"finished_at,omitempty"`
	FailureReason     *string              `json:"failure_reason,omitempty"`
}

type CreateEvalSetParams struct {
	WorkspaceID       uuid.UUID
	OrganizationID    uuid.UUID
	Name              string
	Manifest          json.RawMessage
	Expansion         json.RawMessage
	MaxConcurrentRuns int32
	BudgetUSD         *float64
	CaseFanout        bool
	CombinationCount  int32
	CreatedByUserID   *uuid.UUID
}

func (r *Repository) CreateEvalSet(ctx context.Context, params CreateEvalSetParams) (EvalSet, error) {
	var budget pgtype.Numeric
	if params.BudgetUSD != nil {
		if err := budget.Scan(fmt.Sprintf("%f", *params.BudgetUSD)); err != nil {
			return EvalSet{}, fmt.Errorf("scan budget: %w", err)
		}
	}
	row, err := r.queries.InsertEvalSet(ctx, repositorysqlc.InsertEvalSetParams{
		WorkspaceID:       params.WorkspaceID,
		OrganizationID:    params.OrganizationID,
		Name:              params.Name,
		Status:            string(domain.EvalSetStatusQueued),
		Manifest:          cloneJSON(params.Manifest),
		Expansion:         cloneJSON(params.Expansion),
		MaxConcurrentRuns: params.MaxConcurrentRuns,
		BudgetUsd:         budget,
		CaseFanout:        params.CaseFanout,
		CombinationCount:  params.CombinationCount,
		CreatedByUserID:   params.CreatedByUserID,
	})
	if err != nil {
		return EvalSet{}, fmt.Errorf("insert eval set: %w", err)
	}
	return mapEvalSet(row)
}

func (r *Repository) GetEvalSetByID(ctx context.Context, id uuid.UUID) (EvalSet, error) {
	row, err := r.queries.GetEvalSetByID(ctx, repositorysqlc.GetEvalSetByIDParams{ID: id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EvalSet{}, ErrEvalSetNotFound
		}
		return EvalSet{}, fmt.Errorf("get eval set: %w", err)
	}
	return mapEvalSet(row)
}

func (r *Repository) ListEvalSetsByWorkspaceID(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]EvalSet, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	total, err := r.queries.CountEvalSetsByWorkspaceID(ctx, repositorysqlc.CountEvalSetsByWorkspaceIDParams{WorkspaceID: workspaceID})
	if err != nil {
		return nil, 0, fmt.Errorf("count eval sets: %w", err)
	}
	rows, err := r.queries.ListEvalSetsByWorkspaceID(ctx, repositorysqlc.ListEvalSetsByWorkspaceIDParams{
		WorkspaceID: workspaceID,
		LimitCount:  limit,
		OffsetCount: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list eval sets: %w", err)
	}
	out := make([]EvalSet, 0, len(rows))
	for _, row := range rows {
		mapped, mapErr := mapEvalSet(row)
		if mapErr != nil {
			return nil, 0, mapErr
		}
		out = append(out, mapped)
	}
	return out, total, nil
}

func (r *Repository) TransitionEvalSetStatus(ctx context.Context, id uuid.UUID, from, to domain.EvalSetStatus, failureReason *string) (EvalSet, error) {
	if !from.CanTransitionTo(to) {
		return EvalSet{}, fmt.Errorf("%w: %s → %s", ErrInvalidTransition, from, to)
	}
	row, err := r.queries.UpdateEvalSetStatus(ctx, repositorysqlc.UpdateEvalSetStatusParams{
		ID:            id,
		FromStatus:    string(from),
		ToStatus:      string(to),
		FailureReason: failureReason,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EvalSet{}, ErrInvalidTransition
		}
		return EvalSet{}, fmt.Errorf("update eval set status: %w", err)
	}
	return mapEvalSet(row)
}

func (r *Repository) AttachEvalSessionToEvalSet(ctx context.Context, evalSetID, evalSessionID uuid.UUID, packRef string) error {
	if err := r.queries.AttachEvalSessionToEvalSet(ctx, repositorysqlc.AttachEvalSessionToEvalSetParams{
		EvalSetID:     evalSetID,
		EvalSessionID: evalSessionID,
		PackRef:       packRef,
	}); err != nil {
		return fmt.Errorf("attach eval session to eval set: %w", err)
	}
	return nil
}

func (r *Repository) ListEvalSessionsByEvalSetID(ctx context.Context, evalSetID uuid.UUID) ([]uuid.UUID, []string, error) {
	rows, err := r.queries.ListEvalSessionsByEvalSetID(ctx, repositorysqlc.ListEvalSessionsByEvalSetIDParams{EvalSetID: evalSetID})
	if err != nil {
		return nil, nil, fmt.Errorf("list eval set sessions: %w", err)
	}
	ids := make([]uuid.UUID, 0, len(rows))
	packs := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.EvalSessionID)
		packs = append(packs, row.PackRef)
	}
	return ids, packs, nil
}

type EvalSetResult struct {
	EvalSetID    uuid.UUID       `json:"eval_set_id"`
	Aggregate    json.RawMessage `json:"aggregate"`
	Evidence     json.RawMessage `json:"evidence"`
	SessionCount int32           `json:"session_count"`
	RunCount     int32           `json:"run_count"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (r *Repository) GetEvalSetResultByEvalSetID(ctx context.Context, evalSetID uuid.UUID) (EvalSetResult, error) {
	row, err := r.queries.GetEvalSetResultByEvalSetID(ctx, repositorysqlc.GetEvalSetResultByEvalSetIDParams{EvalSetID: evalSetID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EvalSetResult{}, ErrEvalSetNotFound
		}
		return EvalSetResult{}, fmt.Errorf("get eval set result: %w", err)
	}
	return EvalSetResult{
		EvalSetID:    row.EvalSetID,
		Aggregate:    append([]byte(nil), row.Aggregate...),
		Evidence:     append([]byte(nil), row.Evidence...),
		SessionCount: row.SessionCount,
		RunCount:     row.RunCount,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

func (r *Repository) UpsertEvalSetResult(ctx context.Context, evalSetID uuid.UUID, aggregate, evidence json.RawMessage, sessionCount, runCount int32) (EvalSetResult, error) {
	row, err := r.queries.UpsertEvalSetResult(ctx, repositorysqlc.UpsertEvalSetResultParams{
		EvalSetID:    evalSetID,
		Aggregate:    cloneJSON(aggregate),
		Evidence:     cloneJSON(evidence),
		SessionCount: sessionCount,
		RunCount:     runCount,
	})
	if err != nil {
		return EvalSetResult{}, fmt.Errorf("upsert eval set result: %w", err)
	}
	return EvalSetResult{
		EvalSetID:    row.EvalSetID,
		Aggregate:    append([]byte(nil), row.Aggregate...),
		Evidence:     append([]byte(nil), row.Evidence...),
		SessionCount: row.SessionCount,
		RunCount:     row.RunCount,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

func mapEvalSet(row repositorysqlc.EvalSet) (EvalSet, error) {
	status := domain.EvalSetStatus(row.Status)
	if err := domain.ValidateEvalSetStatus(status); err != nil {
		return EvalSet{}, err
	}
	out := EvalSet{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		OrganizationID:    row.OrganizationID,
		Name:              row.Name,
		Status:            status,
		Manifest:          append([]byte(nil), row.Manifest...),
		Expansion:         append([]byte(nil), row.Expansion...),
		MaxConcurrentRuns: row.MaxConcurrentRuns,
		CaseFanout:        row.CaseFanout,
		CombinationCount:  row.CombinationCount,
		CreatedByUserID:   row.CreatedByUserID,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
		FailureReason:     row.FailureReason,
	}
	if row.BudgetUsd.Valid {
		f, err := row.BudgetUsd.Float64Value()
		if err == nil && f.Valid {
			v := f.Float64
			out.BudgetUSD = &v
		}
	}
	if row.SpentUsd.Valid {
		f, err := row.SpentUsd.Float64Value()
		if err == nil && f.Valid {
			out.SpentUSD = f.Float64
		}
	}
	if row.EstimatedCostUsd.Valid {
		f, err := row.EstimatedCostUsd.Float64Value()
		if err == nil && f.Valid {
			v := f.Float64
			out.EstimatedCostUSD = &v
		}
	}
	if row.StartedAt.Valid {
		t := row.StartedAt.Time
		out.StartedAt = &t
	}
	if row.FinishedAt.Valid {
		t := row.FinishedAt.Time
		out.FinishedAt = &t
	}
	return out, nil
}

func (r *Repository) UpdateEvalSetSpentUSD(ctx context.Context, id uuid.UUID, spent float64) (EvalSet, error) {
	var spentNum pgtype.Numeric
	if err := spentNum.Scan(fmt.Sprintf("%f", spent)); err != nil {
		return EvalSet{}, fmt.Errorf("scan spent: %w", err)
	}
	row, err := r.queries.UpdateEvalSetSpentUSD(ctx, repositorysqlc.UpdateEvalSetSpentUSDParams{
		ID:       id,
		SpentUsd: spentNum,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EvalSet{}, ErrEvalSetNotFound
		}
		return EvalSet{}, fmt.Errorf("update eval set spent: %w", err)
	}
	return mapEvalSet(row)
}

func (r *Repository) UpdateEvalSetEstimatedCostUSD(ctx context.Context, id uuid.UUID, estimated *float64) (EvalSet, error) {
	var est pgtype.Numeric
	if estimated != nil {
		if err := est.Scan(fmt.Sprintf("%f", *estimated)); err != nil {
			return EvalSet{}, fmt.Errorf("scan estimated: %w", err)
		}
	}
	row, err := r.queries.UpdateEvalSetEstimatedCostUSD(ctx, repositorysqlc.UpdateEvalSetEstimatedCostUSDParams{
		ID:               id,
		EstimatedCostUsd: est,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EvalSet{}, ErrEvalSetNotFound
		}
		return EvalSet{}, fmt.Errorf("update eval set estimated cost: %w", err)
	}
	return mapEvalSet(row)
}

func (r *Repository) SumCaseResultCostByEvalSetID(ctx context.Context, evalSetID uuid.UUID) (float64, error) {
	id := evalSetID
	total, err := r.queries.SumCaseResultCostByEvalSetID(ctx, repositorysqlc.SumCaseResultCostByEvalSetIDParams{EvalSetID: &id})
	if err != nil {
		return 0, fmt.Errorf("sum case result cost: %w", err)
	}
	return total, nil
}

func (r *Repository) ListActiveEvalSetIDsByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.queries.ListActiveEvalSetIDsByWorkspaceID(ctx, repositorysqlc.ListActiveEvalSetIDsByWorkspaceIDParams{WorkspaceID: workspaceID})
	if err != nil {
		return nil, fmt.Errorf("list active eval sets: %w", err)
	}
	return ids, nil
}

// StalledEvalSetRow is a non-terminal eval set past a stall cutoff.
type StalledEvalSetRow struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	Status           string
	UpdatedAt        time.Time
	NewestChildState string
}

func (r *Repository) ListStalledEvalSets(ctx context.Context, cutoff time.Time, limit int32) ([]StalledEvalSetRow, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := r.queries.ListStalledEvalSets(ctx, repositorysqlc.ListStalledEvalSetsParams{
		Cutoff:     pgtype.Timestamptz{Time: cutoff.UTC(), Valid: true},
		LimitCount: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list stalled eval sets: %w", err)
	}
	out := make([]StalledEvalSetRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, StalledEvalSetRow{
			ID:               row.ID,
			WorkspaceID:      row.WorkspaceID,
			Status:           row.Status,
			UpdatedAt:        row.UpdatedAt.Time,
			NewestChildState: row.NewestChildState,
		})
	}
	return out, nil
}
