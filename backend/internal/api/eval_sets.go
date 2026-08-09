package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/agentclash/agentclash/runtime/evalset"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type expandEvalSetRequest struct {
	WorkspaceID string          `json:"workspace_id"`
	Manifest    json.RawMessage `json:"manifest"`
	MaxCombos   int             `json:"max_combinations,omitempty"`
}

// EvalSetManager handles eval-set dry-run expansion (persistence in Fleet 7b).
type EvalSetManager struct {
	authorizer WorkspaceAuthorizer
}

func NewEvalSetManager(authorizer WorkspaceAuthorizer) *EvalSetManager {
	return &EvalSetManager{authorizer: authorizer}
}

func (m *EvalSetManager) Expand(ctx context.Context, caller Caller, workspaceID uuid.UUID, manifestJSON json.RawMessage, maxCombos int) (evalset.ExpansionReport, error) {
	if m == nil || m.authorizer == nil {
		return evalset.ExpansionReport{}, errors.New("eval set manager is not configured")
	}
	if err := m.authorizer.AuthorizeWorkspace(ctx, caller, workspaceID); err != nil {
		return evalset.ExpansionReport{}, err
	}
	manifest, err := evalset.ParseManifest(manifestJSON)
	if err != nil {
		return evalset.ExpansionReport{}, err
	}
	if maxCombos <= 0 {
		maxCombos = evalset.DefaultMaxCombos
	}
	if maxCombos > evalset.MaxAllowedCombos {
		return evalset.ExpansionReport{}, fmt.Errorf("max_combinations %d exceeds server limit %d", maxCombos, evalset.MaxAllowedCombos)
	}
	return manifest.Expand(maxCombos)
}

func registerEvalSetRoutes(router chi.Router, logger *slog.Logger, manager *EvalSetManager) {
	if manager == nil {
		return
	}
	router.Post("/eval-sets/expand", expandEvalSetHandler(logger, manager))
}

func expandEvalSetHandler(_ *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "failed to read request body")
			return
		}
		var req expandEvalSetRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must be JSON")
			return
		}
		workspaceID, err := uuid.Parse(req.WorkspaceID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_workspace_id", "workspace_id must be a valid UUID")
			return
		}
		if len(req.Manifest) == 0 {
			writeError(w, http.StatusBadRequest, "manifest_required", "manifest is required")
			return
		}
		report, err := manager.Expand(r.Context(), caller, workspaceID, req.Manifest, req.MaxCombos)
		if err != nil {
			switch {
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				// Validation / parse errors are actionable 400s.
				writeError(w, http.StatusBadRequest, "evalset_expand_invalid", err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}
