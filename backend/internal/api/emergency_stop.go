package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EmergencyStopStore interface {
	ListActiveEvalSetIDsByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error)
}

type emergencyStopResponse struct {
	WorkspaceID    string    `json:"workspace_id"`
	CancelledSets  int       `json:"cancelled_sets"`
	EvalSetIDs     []string  `json:"eval_set_ids"`
	RequestedAt    time.Time `json:"requested_at"`
	RequestedBy    string    `json:"requested_by"`
	AuditEvent     string    `json:"audit_event"`
}

// EmergencyStop cancels every in-flight eval set in a workspace (admin-only).
func (m *EvalSetManager) EmergencyStop(ctx context.Context, caller Caller, workspaceID uuid.UUID) (emergencyStopResponse, error) {
	if m.store == nil {
		return emergencyStopResponse{}, errors.New("eval set persistence is not configured")
	}
	if err := AuthorizeWorkspaceAction(ctx, m.authorizer, caller, workspaceID, ActionManageInfrastructure); err != nil {
		return emergencyStopResponse{}, err
	}
	lister, ok := m.store.(EmergencyStopStore)
	if !ok {
		return emergencyStopResponse{}, errors.New("emergency stop store is not configured")
	}
	ids, err := lister.ListActiveEvalSetIDsByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return emergencyStopResponse{}, err
	}
	cancelled := 0
	outIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		set, err := m.Cancel(ctx, caller, id)
		if err != nil {
			slog.Warn("emergency_stop.cancel_set_failed",
				"workspace_id", workspaceID.String(),
				"eval_set_id", id.String(),
				"error", err,
				"actor_user_id", caller.UserID.String(),
			)
			continue
		}
		if set.Status == domain.EvalSetStatusCancelled {
			cancelled++
		}
		outIDs = append(outIDs, id.String())
	}
	resp := emergencyStopResponse{
		WorkspaceID:   workspaceID.String(),
		CancelledSets: cancelled,
		EvalSetIDs:    outIDs,
		RequestedAt:   time.Now().UTC(),
		RequestedBy:   caller.UserID.String(),
		AuditEvent:    "workspace.emergency_stop",
	}
	slog.Info("audit.workspace.emergency_stop",
		"workspace_id", workspaceID.String(),
		"actor_user_id", caller.UserID.String(),
		"cancelled_sets", cancelled,
		"eval_set_ids", outIDs,
	)
	return resp, nil
}

func registerEmergencyStopRoute(router chi.Router, logger *slog.Logger, manager *EvalSetManager) {
	if manager == nil {
		return
	}
	router.Post("/workspaces/{workspaceID}/emergency-stop", emergencyStopHandler(logger, manager))
}

func emergencyStopHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		workspaceID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_workspace_id", "workspace ID must be a valid UUID")
			return
		}
		resp, err := manager.EmergencyStop(r.Context(), caller, workspaceID)
		if err != nil {
			switch {
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("emergency stop failed", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
