package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	repositorysqlc "github.com/agentclash/agentclash/backend/internal/repository/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrScanFindingNotFound = errors.New("scan finding not found")

type ScanFinding struct {
	ID              uuid.UUID  `json:"id"`
	WorkspaceID     uuid.UUID  `json:"workspace_id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	EvalSetID       uuid.UUID  `json:"eval_set_id"`
	CaseResultID    *uuid.UUID `json:"case_result_id,omitempty"`
	MatrixKey       string     `json:"matrix_key"`
	CaseKey         string     `json:"case_key"`
	Scanner         string     `json:"scanner"`
	ScannerVersion  string     `json:"scanner_version"`
	Severity        string     `json:"severity"`
	Category        string     `json:"category"`
	Evidence        string     `json:"evidence"`
	Confidence      float64    `json:"confidence"`
	Status          string     `json:"status"`
	StatusUpdatedBy *uuid.UUID `json:"status_updated_by,omitempty"`
	StatusUpdatedAt *time.Time `json:"status_updated_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UpsertScanFindingParams struct {
	WorkspaceID    uuid.UUID
	OrganizationID uuid.UUID
	EvalSetID      uuid.UUID
	CaseResultID   *uuid.UUID
	MatrixKey      string
	CaseKey        string
	Scanner        string
	ScannerVersion string
	Severity       string
	Category       string
	Evidence       string
	Confidence     float64
	Status         string
}

func (r *Repository) UpsertScanFinding(ctx context.Context, params UpsertScanFindingParams) (ScanFinding, error) {
	status := params.Status
	if status == "" {
		status = "open"
	}
	row, err := r.queries.UpsertScanFinding(ctx, repositorysqlc.UpsertScanFindingParams{
		WorkspaceID:    params.WorkspaceID,
		OrganizationID: params.OrganizationID,
		EvalSetID:      params.EvalSetID,
		CaseResultID:   params.CaseResultID,
		MatrixKey:      params.MatrixKey,
		CaseKey:        params.CaseKey,
		Scanner:        params.Scanner,
		ScannerVersion: params.ScannerVersion,
		Severity:       params.Severity,
		Category:       params.Category,
		Evidence:       params.Evidence,
		Confidence:     params.Confidence,
		Status:         status,
	})
	if err != nil {
		return ScanFinding{}, fmt.Errorf("upsert scan finding: %w", err)
	}
	return mapScanFinding(row), nil
}

func (r *Repository) ListScanFindingsByEvalSetID(ctx context.Context, evalSetID uuid.UUID, severity, status *string, limit, offset int32) ([]ScanFinding, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.queries.ListScanFindingsByEvalSetID(ctx, repositorysqlc.ListScanFindingsByEvalSetIDParams{
		EvalSetID:   evalSetID,
		Severity:    severity,
		Status:      status,
		LimitCount:  limit,
		OffsetCount: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list scan findings: %w", err)
	}
	out := make([]ScanFinding, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapScanFinding(row))
	}
	return out, nil
}

func (r *Repository) GetScanFindingByID(ctx context.Context, id uuid.UUID) (ScanFinding, error) {
	row, err := r.queries.GetScanFindingByID(ctx, repositorysqlc.GetScanFindingByIDParams{ID: id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScanFinding{}, ErrScanFindingNotFound
		}
		return ScanFinding{}, fmt.Errorf("get scan finding: %w", err)
	}
	return mapScanFinding(row), nil
}

func (r *Repository) CountScanFindingsBySeverity(ctx context.Context, evalSetID uuid.UUID) (map[string]int64, error) {
	rows, err := r.queries.CountScanFindingsBySeverity(ctx, repositorysqlc.CountScanFindingsBySeverityParams{EvalSetID: evalSetID})
	if err != nil {
		return nil, fmt.Errorf("count scan findings: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Severity] = row.N
	}
	return out, nil
}

func (r *Repository) UpdateScanFindingStatus(ctx context.Context, id uuid.UUID, status string, updatedBy *uuid.UUID) (ScanFinding, error) {
	row, err := r.queries.UpdateScanFindingStatus(ctx, repositorysqlc.UpdateScanFindingStatusParams{
		ID:              id,
		Status:          status,
		StatusUpdatedBy: updatedBy,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScanFinding{}, ErrScanFindingNotFound
		}
		return ScanFinding{}, fmt.Errorf("update scan finding status: %w", err)
	}
	return mapScanFinding(row), nil
}

func (r *Repository) ClearScanFindingsForTarget(ctx context.Context, evalSetID uuid.UUID, caseKey, scanner, scannerVersion string) error {
	if err := r.queries.ClearScanFindingsForTarget(ctx, repositorysqlc.ClearScanFindingsForTargetParams{
		EvalSetID:      evalSetID,
		CaseKey:        caseKey,
		Scanner:        scanner,
		ScannerVersion: scannerVersion,
	}); err != nil {
		return fmt.Errorf("clear scan findings for target: %w", err)
	}
	return nil
}

func mapScanFinding(row repositorysqlc.ScanFinding) ScanFinding {
	out := ScanFinding{
		ID:             row.ID,
		WorkspaceID:    row.WorkspaceID,
		OrganizationID: row.OrganizationID,
		EvalSetID:      row.EvalSetID,
		CaseResultID:   row.CaseResultID,
		MatrixKey:      row.MatrixKey,
		CaseKey:        row.CaseKey,
		Scanner:        row.Scanner,
		ScannerVersion: row.ScannerVersion,
		Severity:       row.Severity,
		Category:       row.Category,
		Evidence:       row.Evidence,
		Confidence:     row.Confidence,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
	out.StatusUpdatedBy = row.StatusUpdatedBy
	if row.StatusUpdatedAt.Valid {
		t := row.StatusUpdatedAt.Time
		out.StatusUpdatedAt = &t
	}
	return out
}
