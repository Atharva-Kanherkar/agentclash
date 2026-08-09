package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ScanFindingStore interface {
	ListScanFindingsByEvalSetID(ctx context.Context, evalSetID uuid.UUID, severity, status *string, limit, offset int32) ([]repository.ScanFinding, error)
	GetScanFindingByID(ctx context.Context, id uuid.UUID) (repository.ScanFinding, error)
	UpdateScanFindingStatus(ctx context.Context, id uuid.UUID, status string, updatedBy *uuid.UUID) (repository.ScanFinding, error)
	CountScanFindingsBySeverity(ctx context.Context, evalSetID uuid.UUID) (map[string]int64, error)
}

type ScanWorkflowStarter interface {
	StartScanEvalSetWorkflow(ctx context.Context, evalSetID uuid.UUID, scanners []string) error
}

func (m *EvalSetManager) WithScanFindings(store ScanFindingStore, starter ScanWorkflowStarter) *EvalSetManager {
	m.findings = store
	m.scanStarter = starter
	return m
}

type scanEvalSetRequest struct {
	Scanners []string `json:"scanners"`
}

type updateFindingStatusRequest struct {
	Status string `json:"status"`
}

func (m *EvalSetManager) StartScan(ctx context.Context, caller Caller, evalSetID uuid.UUID, scannerNames []string) error {
	if m.scanStarter == nil {
		return errors.New("scan workflow starter is not configured")
	}
	set, err := m.Get(ctx, caller, evalSetID)
	if err != nil {
		return err
	}
	if err := AuthorizeWorkspaceAction(ctx, m.authorizer, caller, set.WorkspaceID, ActionCreateRun); err != nil {
		return err
	}
	return m.scanStarter.StartScanEvalSetWorkflow(ctx, evalSetID, scannerNames)
}

func (m *EvalSetManager) ListFindings(ctx context.Context, caller Caller, evalSetID uuid.UUID, severity, status *string, limit, offset int32) ([]repository.ScanFinding, error) {
	if m.findings == nil {
		return nil, errors.New("scan findings store is not configured")
	}
	set, err := m.Get(ctx, caller, evalSetID)
	if err != nil {
		return nil, err
	}
	_ = set
	return m.findings.ListScanFindingsByEvalSetID(ctx, evalSetID, severity, status, limit, offset)
}

func (m *EvalSetManager) UpdateFindingStatus(ctx context.Context, caller Caller, findingID uuid.UUID, status string) (repository.ScanFinding, error) {
	if m.findings == nil {
		return repository.ScanFinding{}, errors.New("scan findings store is not configured")
	}
	finding, err := m.findings.GetScanFindingByID(ctx, findingID)
	if err != nil {
		return repository.ScanFinding{}, err
	}
	if err := AuthorizeWorkspaceAction(ctx, m.authorizer, caller, finding.WorkspaceID, ActionCreateRun); err != nil {
		return repository.ScanFinding{}, err
	}
	switch status {
	case "open", "triaged", "dismissed":
	default:
		return repository.ScanFinding{}, errors.New("status must be open, triaged, or dismissed")
	}
	// Triage is a write; reuse create_run (same role gate as starting a scan).
	userID := caller.UserID
	updated, err := m.findings.UpdateScanFindingStatus(ctx, findingID, status, &userID)
	if err != nil {
		return repository.ScanFinding{}, err
	}
	slog.Info("audit.scan_finding.status_update",
		"finding_id", findingID.String(),
		"eval_set_id", finding.EvalSetID.String(),
		"status", status,
		"actor_user_id", caller.UserID.String(),
	)
	return updated, nil
}

func registerScanFindingRoutes(router chi.Router, logger *slog.Logger, manager *EvalSetManager) {
	if manager == nil {
		return
	}
	router.Post("/eval-sets/{evalSetID}/scan", scanEvalSetHandler(logger, manager))
	router.Get("/eval-sets/{evalSetID}/findings", listFindingsHandler(logger, manager))
	router.Patch("/scan-findings/{findingID}", updateFindingStatusHandler(logger, manager))
}

func scanEvalSetHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "evalSetID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_eval_set_id", "eval set ID must be a valid UUID")
			return
		}
		var req scanEvalSetRequest
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "failed to read body")
			return
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_json", "body must be JSON")
				return
			}
		}
		if err := manager.StartScan(r.Context(), caller, id, req.Scanners); err != nil {
			switch {
			case errors.Is(err, repository.ErrEvalSetNotFound):
				writeError(w, http.StatusNotFound, "eval_set_not_found", "eval set not found")
			case errors.Is(err, ErrScanAlreadyRunning):
				writeError(w, http.StatusConflict, "scan_already_running", "a scan is already running for this eval set")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("start scan failed", "error", err)
				writeError(w, http.StatusBadRequest, "scan_start_failed", err.Error())
			}
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"eval_set_id": id, "status": "scan_started"})
	}
}

func listFindingsHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "evalSetID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_eval_set_id", "eval set ID must be a valid UUID")
			return
		}
		var severity, status *string
		if v := r.URL.Query().Get("severity"); v != "" {
			severity = &v
		}
		if v := r.URL.Query().Get("status"); v != "" {
			status = &v
		}
		limit := int32(100)
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = int32(n)
			}
		}
		findings, err := manager.ListFindings(r.Context(), caller, id, severity, status, limit, 0)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrEvalSetNotFound):
				writeError(w, http.StatusNotFound, "eval_set_not_found", "eval set not found")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("list findings failed", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"findings": findings})
	}
}

func updateFindingStatusHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "findingID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_finding_id", "finding ID must be a valid UUID")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "failed to read body")
			return
		}
		var req updateFindingStatusRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "body must be JSON")
			return
		}
		finding, err := manager.UpdateFindingStatus(r.Context(), caller, id, req.Status)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrScanFindingNotFound):
				writeError(w, http.StatusNotFound, "finding_not_found", "finding not found")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				writeError(w, http.StatusBadRequest, "finding_update_invalid", err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, finding)
	}
}
