package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/backend/internal/workflow"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/agentclash/agentclash/runtime/evalset"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EvalSetStore interface {
	CreateEvalSet(ctx context.Context, params repository.CreateEvalSetParams) (repository.EvalSet, error)
	GetEvalSetByID(ctx context.Context, id uuid.UUID) (repository.EvalSet, error)
	ListEvalSetsByWorkspaceID(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]repository.EvalSet, int64, error)
	AttachEvalSessionToEvalSet(ctx context.Context, evalSetID, evalSessionID uuid.UUID, packRef string) error
	ListEvalSessionsByEvalSetID(ctx context.Context, evalSetID uuid.UUID) ([]uuid.UUID, []string, error)
	TransitionEvalSetStatus(ctx context.Context, id uuid.UUID, from, to domain.EvalSetStatus, failureReason *string) (repository.EvalSet, error)
	TransitionEvalSessionStatus(ctx context.Context, params repository.TransitionEvalSessionStatusParams) (domain.EvalSession, error)
	GetOrganizationIDByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (uuid.UUID, error)
	GetEvalSetResultByEvalSetID(ctx context.Context, evalSetID uuid.UUID) (repository.EvalSetResult, error)
}

type EvalSetWorkflowStarter interface {
	StartEvalSetWorkflow(ctx context.Context, evalSetID uuid.UUID) error
	CancelEvalSetWorkflow(ctx context.Context, evalSetID uuid.UUID) error
}

type EvalSessionCreator interface {
	CreateEvalSession(ctx context.Context, caller Caller, input CreateEvalSessionInput) (CreateEvalSessionResult, error)
}

type createEvalSetRequest struct {
	WorkspaceID string          `json:"workspace_id"`
	Manifest    json.RawMessage `json:"manifest"`
	MaxCombos   int             `json:"max_combinations,omitempty"`
}

// WithPersistence wires create/list/get/cancel dependencies.
func (m *EvalSetManager) WithPersistence(store EvalSetStore, sessions EvalSessionCreator, starter EvalSetWorkflowStarter) *EvalSetManager {
	m.store = store
	m.sessions = sessions
	m.starter = starter
	return m
}

func (m *EvalSetManager) Create(ctx context.Context, caller Caller, workspaceID uuid.UUID, manifestJSON json.RawMessage, maxCombos int) (set repository.EvalSet, sessionIDs []uuid.UUID, err error) {
	if m.store == nil || m.sessions == nil {
		return repository.EvalSet{}, nil, errors.New("eval set persistence is not configured")
	}
	if err := AuthorizeWorkspaceAction(ctx, m.authorizer, caller, workspaceID, ActionCreateRun); err != nil {
		return repository.EvalSet{}, nil, err
	}
	expanded, err := m.ExpandWithEstimate(ctx, caller, workspaceID, manifestJSON, maxCombos)
	if err != nil {
		return repository.EvalSet{}, nil, err
	}
	report := expanded.Report
	if err := validateUUIDRefs(report); err != nil {
		return repository.EvalSet{}, nil, err
	}

	orgID, err := m.store.GetOrganizationIDByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return repository.EvalSet{}, nil, err
	}
	expansionJSON, _ := json.Marshal(report)
	userID := caller.UserID
	set, err = m.store.CreateEvalSet(ctx, repository.CreateEvalSetParams{
		WorkspaceID:       workspaceID,
		OrganizationID:    orgID,
		Name:              report.Name,
		Manifest:          append([]byte(nil), manifestJSON...),
		Expansion:         expansionJSON,
		MaxConcurrentRuns: int32(report.MaxConcurrent),
		BudgetUSD:         budgetPtr(report.BudgetUSD),
		CaseFanout:        report.CaseFanout,
		CombinationCount:  int32(report.Count),
		CreatedByUserID:   &userID,
	})
	if err != nil {
		return repository.EvalSet{}, nil, err
	}
	if updater, ok := m.store.(interface {
		UpdateEvalSetEstimatedCostUSD(ctx context.Context, id uuid.UUID, estimated *float64) (repository.EvalSet, error)
	}); ok {
		est := expanded.Estimate.EstimatedUSD
		if updated, updErr := updater.UpdateEvalSetEstimatedCostUSD(ctx, set.ID, &est); updErr == nil {
			set = updated
		}
	}

	sessionIDs = make([]uuid.UUID, 0)
	defer func() {
		if err == nil {
			return
		}
		m.compensatePartialCreate(ctx, set.ID, sessionIDs, err)
	}()

	byPack := groupCombinationsByPack(report.Combinations)
	packRefs := make([]string, 0, len(byPack))
	for pack := range byPack {
		packRefs = append(packRefs, pack)
	}
	sort.Strings(packRefs)

	for _, packRef := range packRefs {
		combos := byPack[packRef]
		packVersionID := uuid.MustParse(packRef)
		matrix := make([]EvalSessionRunMatrixEntryInput, 0, len(combos))
		for _, c := range combos {
			agentID := uuid.MustParse(c.AgentRef)
			entry := EvalSessionRunMatrixEntryInput{
				Key: c.MatrixKey,
				Participants: []EvalSessionParticipantInput{{
					AgentDeploymentID: &agentID,
					Label:             c.AgentRef,
				}},
				Seed: c.Seed,
			}
			matrix = append(matrix, entry)
		}
		var result CreateEvalSessionResult
		result, err = m.sessions.CreateEvalSession(ctx, caller, CreateEvalSessionInput{
			WorkspaceID:            workspaceID,
			ChallengePackVersionID: packVersionID,
			ExecutionMode:          "single_agent",
			Name:                   fmt.Sprintf("%s/%s", report.Name, packRef),
			SkipWorkflowStart:      true,
			EvalSession: CreateEvalSessionConfigInput{
				Repetitions:   int32(len(matrix)),
				RunMatrix:     matrix,
				SchemaVersion: 1,
				Aggregation: EvalSessionAggregationInput{
					Method:             "median",
					ReportVariance:     false,
					ConfidenceInterval: 0.95,
				},
			},
		})
		if err != nil {
			err = fmt.Errorf("create eval session for pack %s: %w", packRef, err)
			return set, sessionIDs, err
		}
		sessionIDs = append(sessionIDs, result.Session.ID)
		if err = m.store.AttachEvalSessionToEvalSet(ctx, set.ID, result.Session.ID, packRef); err != nil {
			return set, sessionIDs, err
		}
	}

	if m.starter != nil {
		if err = m.starter.StartEvalSetWorkflow(ctx, set.ID); err != nil {
			err = fmt.Errorf("start eval set workflow: %w", err)
			return set, sessionIDs, err
		}
	}
	return set, sessionIDs, nil
}

func (m *EvalSetManager) compensatePartialCreate(ctx context.Context, evalSetID uuid.UUID, sessionIDs []uuid.UUID, cause error) {
	reason := "create failed: " + cause.Error()
	if _, transErr := m.store.TransitionEvalSetStatus(ctx, evalSetID, domain.EvalSetStatusQueued, domain.EvalSetStatusFailed, &reason); transErr != nil {
		slog.Default().Warn("compensate eval set status failed",
			"eval_set_id", evalSetID, "error", transErr)
	}
	for _, sessionID := range sessionIDs {
		if _, cancelErr := m.store.TransitionEvalSessionStatus(ctx, repository.TransitionEvalSessionStatusParams{
			EvalSessionID: sessionID,
			ToStatus:      domain.EvalSessionStatusCancelled,
		}); cancelErr != nil {
			slog.Default().Warn("compensate eval session cancel failed",
				"eval_set_id", evalSetID, "eval_session_id", sessionID, "error", cancelErr)
		}
	}
}

func (m *EvalSetManager) Get(ctx context.Context, caller Caller, id uuid.UUID) (repository.EvalSet, error) {
	if m.store == nil {
		return repository.EvalSet{}, errors.New("eval set persistence is not configured")
	}
	set, err := m.store.GetEvalSetByID(ctx, id)
	if err != nil {
		return repository.EvalSet{}, err
	}
	if err := m.authorizer.AuthorizeWorkspace(ctx, caller, set.WorkspaceID); err != nil {
		return repository.EvalSet{}, err
	}
	return set, nil
}

func (m *EvalSetManager) List(ctx context.Context, caller Caller, workspaceID uuid.UUID, limit, offset int32) ([]repository.EvalSet, int64, error) {
	if m.store == nil {
		return nil, 0, errors.New("eval set persistence is not configured")
	}
	if err := m.authorizer.AuthorizeWorkspace(ctx, caller, workspaceID); err != nil {
		return nil, 0, err
	}
	return m.store.ListEvalSetsByWorkspaceID(ctx, workspaceID, limit, offset)
}

func (m *EvalSetManager) Cancel(ctx context.Context, caller Caller, id uuid.UUID) (repository.EvalSet, error) {
	if m.store == nil {
		return repository.EvalSet{}, errors.New("eval set persistence is not configured")
	}
	set, err := m.store.GetEvalSetByID(ctx, id)
	if err != nil {
		return repository.EvalSet{}, err
	}
	if err := AuthorizeWorkspaceAction(ctx, m.authorizer, caller, set.WorkspaceID, ActionCancelRun); err != nil {
		return repository.EvalSet{}, err
	}
	if domain.IsEvalSetTerminal(set.Status) {
		return set, nil
	}
	if !set.Status.CanTransitionTo(domain.EvalSetStatusCancelled) {
		return set, nil
	}
	if m.starter != nil {
		if err := m.starter.CancelEvalSetWorkflow(ctx, id); err != nil {
			latest, latestErr := m.store.GetEvalSetByID(ctx, id)
			if latestErr == nil && !latest.Status.CanTransitionTo(domain.EvalSetStatusCancelled) {
				return latest, nil
			}
			// Mirror run cancel: only NotFound means the workflow is already gone.
			if !isTemporalNotFound(err) {
				return repository.EvalSet{}, fmt.Errorf("cancel eval set workflow: %w", err)
			}
		}
	}
	reason := "cancelled by caller"
	cancelled, err := m.store.TransitionEvalSetStatus(ctx, id, set.Status, domain.EvalSetStatusCancelled, &reason)
	if err != nil {
		latest, getErr := m.store.GetEvalSetByID(ctx, id)
		if getErr == nil && (latest.Status == domain.EvalSetStatusCancelled ||
			latest.Status == domain.EvalSetStatusCompleted ||
			latest.Status == domain.EvalSetStatusFailed) {
			return latest, nil
		}
		return repository.EvalSet{}, err
	}
	return cancelled, nil
}

func validateUUIDRefs(report evalset.ExpansionReport) error {
	for _, c := range report.Combinations {
		if _, err := uuid.Parse(c.PackRef); err != nil {
			return fmt.Errorf("pack_ref %q must be a challenge_pack_version UUID (catalog slug resolution is CLI-side in Fleet 8)", c.PackRef)
		}
		if _, err := uuid.Parse(c.AgentRef); err != nil {
			return fmt.Errorf("agent_ref %q must be an agent_deployment UUID", c.AgentRef)
		}
	}
	return nil
}

func groupCombinationsByPack(combos []evalset.Combination) map[string][]evalset.Combination {
	out := map[string][]evalset.Combination{}
	for _, c := range combos {
		out[c.PackRef] = append(out[c.PackRef], c)
	}
	return out
}

func budgetPtr(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func createEvalSetHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
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
		var req createEvalSetRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must be JSON")
			return
		}
		workspaceID, err := uuid.Parse(req.WorkspaceID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_workspace_id", "workspace_id must be a valid UUID")
			return
		}
		set, sessionIDs, err := manager.Create(r.Context(), caller, workspaceID, req.Manifest, req.MaxCombos)
		if err != nil {
			switch {
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("create eval set failed", "error", err)
				writeError(w, http.StatusBadRequest, "evalset_create_invalid", err.Error())
			}
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"eval_set":          set,
			"eval_session_ids":  sessionIDs,
			"combination_count": set.CombinationCount,
			"workflow":          workflow.EvalSetWorkflowName,
		})
	}
}

func listEvalSetsHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		workspaceID, err := uuid.Parse(r.URL.Query().Get("workspace_id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_workspace_id", "workspace_id must be a valid UUID")
			return
		}
		sets, total, err := manager.List(r.Context(), caller, workspaceID, 20, 0)
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				writeAuthzError(w, err)
				return
			}
			logger.Error("list eval sets failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"eval_sets": sets, "total": total})
	}
}

func getEvalSetHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
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
		set, err := manager.Get(r.Context(), caller, id)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrEvalSetNotFound):
				writeError(w, http.StatusNotFound, "eval_set_not_found", "eval set not found")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("get eval set failed", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}
		resp := map[string]any{"eval_set": set}
		if manager.store != nil {
			if result, resultErr := manager.store.GetEvalSetResultByEvalSetID(r.Context(), id); resultErr == nil {
				resp["result"] = result
			}
			if sessionIDs, _, listErr := manager.store.ListEvalSessionsByEvalSetID(r.Context(), id); listErr == nil {
				resp["eval_session_ids"] = sessionIDs
			}
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func cancelEvalSetHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
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
		set, err := manager.Cancel(r.Context(), caller, id)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrEvalSetNotFound):
				writeError(w, http.StatusNotFound, "eval_set_not_found", "eval set not found")
			case errors.Is(err, ErrForbidden):
				writeAuthzError(w, err)
			default:
				logger.Error("cancel eval set failed", "error", err)
				writeError(w, http.StatusBadRequest, "evalset_cancel_invalid", err.Error())
			}
			return
		}
		writeJSON(w, http.StatusOK, set)
	}
}
