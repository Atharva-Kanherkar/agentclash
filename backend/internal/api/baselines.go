package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// BaselineRepository is the subset of the repository surface the baseline
// service calls. Kept small so handlers can be unit-tested with a fake.
type BaselineRepository interface {
	UpsertBaseline(ctx context.Context, params repository.UpsertBaselineParams) (repository.Baseline, error)
	GetBaselineByName(ctx context.Context, workspaceID uuid.UUID, name string) (repository.Baseline, error)
	ListBaselinesByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]repository.Baseline, error)
}

// BaselineService is injected into the route registration so tests can stub
// out the whole backend. In prod it's just a thin wrapper around the
// repository.
type BaselineService interface {
	Upsert(ctx context.Context, params UpsertBaselineInput) (BaselineResponse, error)
	Get(ctx context.Context, workspaceID uuid.UUID, name string) (BaselineResponse, error)
	List(ctx context.Context, workspaceID uuid.UUID) (ListBaselinesResponse, error)
}

type BaselineManager struct {
	repo BaselineRepository
}

func NewBaselineManager(repo BaselineRepository) *BaselineManager {
	return &BaselineManager{repo: repo}
}

type UpsertBaselineInput struct {
	WorkspaceID       uuid.UUID
	Name              string
	PackVersionID     uuid.UUID
	RunID             uuid.UUID
	ScorecardSnapshot json.RawMessage
}

type BaselineResponse struct {
	ID                uuid.UUID       `json:"id"`
	WorkspaceID       uuid.UUID       `json:"workspace_id"`
	Name              string          `json:"name"`
	PackVersionID     uuid.UUID       `json:"pack_version_id"`
	RunID             uuid.UUID       `json:"run_id"`
	ScorecardSnapshot json.RawMessage `json:"scorecard_snapshot"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type ListBaselinesResponse struct {
	Items []BaselineResponse `json:"items"`
}

func (m *BaselineManager) Upsert(ctx context.Context, params UpsertBaselineInput) (BaselineResponse, error) {
	row, err := m.repo.UpsertBaseline(ctx, repository.UpsertBaselineParams{
		WorkspaceID:       params.WorkspaceID,
		Name:              params.Name,
		PackVersionID:     params.PackVersionID,
		RunID:             params.RunID,
		ScorecardSnapshot: params.ScorecardSnapshot,
	})
	if err != nil {
		return BaselineResponse{}, err
	}
	return baselineToResponse(row), nil
}

func (m *BaselineManager) Get(ctx context.Context, workspaceID uuid.UUID, name string) (BaselineResponse, error) {
	row, err := m.repo.GetBaselineByName(ctx, workspaceID, name)
	if err != nil {
		return BaselineResponse{}, err
	}
	return baselineToResponse(row), nil
}

func (m *BaselineManager) List(ctx context.Context, workspaceID uuid.UUID) (ListBaselinesResponse, error) {
	rows, err := m.repo.ListBaselinesByWorkspace(ctx, workspaceID)
	if err != nil {
		return ListBaselinesResponse{}, err
	}
	items := make([]BaselineResponse, len(rows))
	for i, row := range rows {
		items[i] = baselineToResponse(row)
	}
	return ListBaselinesResponse{Items: items}, nil
}

func baselineToResponse(row repository.Baseline) BaselineResponse {
	snap := row.ScorecardSnapshot
	if len(snap) == 0 {
		snap = json.RawMessage(`{}`)
	}
	return BaselineResponse{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		Name:              row.Name,
		PackVersionID:     row.PackVersionID,
		RunID:             row.RunID,
		ScorecardSnapshot: snap,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

// upsertBaselineRequest is the POST body for `/baselines/{name}`. name is
// taken from the URL so the body stays minimal — callers don't have to
// keep two copies in sync.
type upsertBaselineRequest struct {
	PackVersionID     string          `json:"pack_version_id"`
	RunID             string          `json:"run_id"`
	ScorecardSnapshot json.RawMessage `json:"scorecard_snapshot"`
}

// maxBaselineBodySize caps the upload at 1 MiB. A scorecard snapshot of
// dozens of (task × model) rows with lazy JSON is well under this; caps
// exist to keep a hostile client from pushing multi-MB blobs.
const maxBaselineBodySize = 1 << 20

func upsertBaselineHandler(logger *slog.Logger, service BaselineService, authorizer WorkspaceAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		workspaceID, err := WorkspaceIDFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		if err := AuthorizeWorkspaceAction(r.Context(), authorizer, caller, workspaceID, ActionManageBaseline); err != nil {
			writeAuthzError(w, err)
			return
		}

		name := strings.TrimSpace(chi.URLParam(r, "name"))
		if name == "" {
			writeError(w, http.StatusBadRequest, "invalid_baseline_name", "baseline name is required in the URL")
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBaselineBodySize)
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_body", "could not read request body")
			return
		}
		var req upsertBaselineRequest
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
				return
			}
		}

		packVersionID, err := uuid.Parse(req.PackVersionID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_pack_version_id", "pack_version_id must be a UUID")
			return
		}
		runID, err := uuid.Parse(req.RunID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_run_id", "run_id must be a UUID")
			return
		}

		resp, err := service.Upsert(r.Context(), UpsertBaselineInput{
			WorkspaceID:       workspaceID,
			Name:              name,
			PackVersionID:     packVersionID,
			RunID:             runID,
			ScorecardSnapshot: req.ScorecardSnapshot,
		})
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrBaselineNameRequired):
				writeError(w, http.StatusBadRequest, "invalid_baseline_name", err.Error())
			default:
				logger.Error("upsert baseline failed",
					"workspace_id", workspaceID,
					"name", name,
					"error", err,
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func getBaselineHandler(logger *slog.Logger, service BaselineService, authorizer WorkspaceAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		workspaceID, err := WorkspaceIDFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		// Read permission: ActionReadWorkspace covers everyone including
		// viewer-only tokens; no need for a baseline-specific read action.
		if err := AuthorizeWorkspaceAction(r.Context(), authorizer, caller, workspaceID, ActionReadWorkspace); err != nil {
			writeAuthzError(w, err)
			return
		}

		name := strings.TrimSpace(chi.URLParam(r, "name"))
		if name == "" {
			writeError(w, http.StatusBadRequest, "invalid_baseline_name", "baseline name is required in the URL")
			return
		}

		resp, err := service.Get(r.Context(), workspaceID, name)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrBaselineNotFound):
				writeError(w, http.StatusNotFound, "baseline_not_found", err.Error())
			case errors.Is(err, repository.ErrBaselineNameRequired):
				writeError(w, http.StatusBadRequest, "invalid_baseline_name", err.Error())
			default:
				logger.Error("get baseline failed",
					"workspace_id", workspaceID,
					"name", name,
					"error", err,
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func listBaselinesHandler(logger *slog.Logger, service BaselineService, authorizer WorkspaceAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		workspaceID, err := WorkspaceIDFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		if err := AuthorizeWorkspaceAction(r.Context(), authorizer, caller, workspaceID, ActionReadWorkspace); err != nil {
			writeAuthzError(w, err)
			return
		}

		resp, err := service.List(r.Context(), workspaceID)
		if err != nil {
			logger.Error("list baselines failed",
				"workspace_id", workspaceID,
				"error", err,
			)
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
