package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CaseResultsStore interface {
	ListCaseResults(ctx context.Context, filter repository.ListCaseResultsFilter) ([]repository.CaseResult, error)
	SearchCaseResults(ctx context.Context, filter repository.ListCaseResultsFilter) ([]repository.CaseResult, error)
	ListCaseResultsForExport(ctx context.Context, workspaceID, evalSetID uuid.UUID, cursor *uuid.UUID, limit int32) ([]repository.CaseResult, error)
	AggregateCaseResults(ctx context.Context, workspaceID, evalSetID uuid.UUID) ([]repository.CaseResultAxisAggregate, error)
	ListCaseResultsForCompare(ctx context.Context, workspaceID, evalSetID uuid.UUID) ([]repository.CaseResultCompareRow, error)
}

// WithCaseResults wires warehouse read APIs onto the eval-set manager.
func (m *EvalSetManager) WithCaseResults(store CaseResultsStore) *EvalSetManager {
	m.cases = store
	return m
}

func registerEvalSetWarehouseRoutes(router chi.Router, logger *slog.Logger, manager *EvalSetManager) {
	if manager == nil {
		return
	}
	router.Get("/eval-sets/{evalSetID}/search", searchEvalSetCasesHandler(logger, manager))
	router.Get("/eval-sets/{evalSetID}/cases", listEvalSetCasesHandler(logger, manager))
	router.Get("/eval-sets/{evalSetID}/report", reportEvalSetHandler(logger, manager))
	router.Get("/eval-sets/{evalSetID}/export", exportEvalSetHandler(logger, manager))
	router.Get("/compare/eval-sets", compareEvalSetsHandler(logger, manager))
}

func (m *EvalSetManager) authorizeEvalSet(ctx context.Context, caller Caller, id uuid.UUID) (repository.EvalSet, error) {
	set, err := m.Get(ctx, caller, id)
	if err != nil {
		return repository.EvalSet{}, err
	}
	return set, nil
}

func (m *EvalSetManager) SearchCases(ctx context.Context, caller Caller, evalSetID uuid.UUID, filter repository.ListCaseResultsFilter) ([]repository.CaseResult, error) {
	set, err := m.authorizeEvalSet(ctx, caller, evalSetID)
	if err != nil {
		return nil, err
	}
	if m.cases == nil {
		return nil, errors.New("case results store is not configured")
	}
	filter.EvalSetID = set.ID
	filter.WorkspaceID = set.WorkspaceID
	return m.cases.SearchCaseResults(ctx, filter)
}

func (m *EvalSetManager) ListCases(ctx context.Context, caller Caller, evalSetID uuid.UUID, filter repository.ListCaseResultsFilter) ([]repository.CaseResult, error) {
	set, err := m.authorizeEvalSet(ctx, caller, evalSetID)
	if err != nil {
		return nil, err
	}
	if m.cases == nil {
		return nil, errors.New("case results store is not configured")
	}
	filter.EvalSetID = set.ID
	filter.WorkspaceID = set.WorkspaceID
	return m.cases.ListCaseResults(ctx, filter)
}

func (m *EvalSetManager) Report(ctx context.Context, caller Caller, evalSetID uuid.UUID) (map[string]any, error) {
	set, err := m.authorizeEvalSet(ctx, caller, evalSetID)
	if err != nil {
		return nil, err
	}
	if m.cases == nil {
		return nil, errors.New("case results store is not configured")
	}
	axes, err := m.cases.AggregateCaseResults(ctx, set.WorkspaceID, set.ID)
	if err != nil {
		return nil, err
	}
	winMatrix := make([]map[string]any, 0, len(axes))
	var totalN, totalWins int64
	for _, a := range axes {
		totalN += a.N
		totalWins += a.Wins
		winMatrix = append(winMatrix, map[string]any{
			"pack_ref":            a.PackRef,
			"agent_deployment_id": a.AgentDeploymentID,
			"model":               a.Model,
			"wins":                a.Wins,
			"losses":              a.Losses,
			"n":                   a.N,
		})
	}
	return map[string]any{
		"eval_set_id": set.ID,
		"marginals":   axes,
		"win_matrix":  winMatrix,
		"totals": map[string]any{
			"n":    totalN,
			"wins": totalWins,
		},
	}, nil
}

func (m *EvalSetManager) Compare(ctx context.Context, caller Caller, aID, bID uuid.UUID) (map[string]any, error) {
	a, err := m.authorizeEvalSet(ctx, caller, aID)
	if err != nil {
		return nil, err
	}
	b, err := m.authorizeEvalSet(ctx, caller, bID)
	if err != nil {
		return nil, err
	}
	if m.cases == nil {
		return nil, errors.New("case results store is not configured")
	}
	aRows, err := m.cases.ListCaseResultsForCompare(ctx, a.WorkspaceID, a.ID)
	if err != nil {
		return nil, err
	}
	bRows, err := m.cases.ListCaseResultsForCompare(ctx, b.WorkspaceID, b.ID)
	if err != nil {
		return nil, err
	}
	bByKey := map[string]repository.CaseResultCompareRow{}
	for _, row := range bRows {
		bByKey[compareKey(row)] = row
	}
	deltas := make([]map[string]any, 0)
	regressions := make([]map[string]any, 0)
	for _, row := range aRows {
		other, ok := bByKey[compareKey(row)]
		if !ok {
			continue
		}
		var delta *float64
		if row.Score != nil && other.Score != nil {
			d := *other.Score - *row.Score
			delta = &d
			if d < -1e-9 {
				regressions = append(regressions, map[string]any{
					"matrix_key":  row.MatrixKey,
					"case_key":    row.CaseKey,
					"a_score":     *row.Score,
					"b_score":     *other.Score,
					"delta":       d,
					"a_run_id":    row.RunID,
					"b_run_id":    other.RunID,
				})
			}
		}
		deltas = append(deltas, map[string]any{
			"matrix_key": row.MatrixKey,
			"case_key":   row.CaseKey,
			"pack_ref":   row.PackRef,
			"a_score":    row.Score,
			"b_score":    other.Score,
			"delta":      delta,
			"a_verdict":  row.Verdict,
			"b_verdict":  other.Verdict,
		})
	}
	return map[string]any{
		"a_eval_set_id": a.ID,
		"b_eval_set_id": b.ID,
		"deltas":        deltas,
		"regressions":   regressions,
		"shared_keys":   len(deltas),
	}, nil
}

func compareKey(row repository.CaseResultCompareRow) string {
	return row.MatrixKey + "\x00" + row.CaseKey
}

func (m *EvalSetManager) ExportCases(ctx context.Context, caller Caller, evalSetID uuid.UUID, cursor *uuid.UUID, limit int32) ([]repository.CaseResult, error) {
	set, err := m.authorizeEvalSet(ctx, caller, evalSetID)
	if err != nil {
		return nil, err
	}
	if m.cases == nil {
		return nil, errors.New("case results store is not configured")
	}
	return m.cases.ListCaseResultsForExport(ctx, set.WorkspaceID, set.ID, cursor, limit)
}

func searchEvalSetCasesHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
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
		filter, err := parseCaseResultsFilter(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_filter", err.Error())
			return
		}
		filter.Query = strings.TrimSpace(r.URL.Query().Get("q"))
		if filter.Query == "" {
			writeError(w, http.StatusBadRequest, "query_required", "q is required")
			return
		}
		rows, err := manager.SearchCases(r.Context(), caller, id, filter)
		if err != nil {
			writeEvalSetWarehouseError(logger, w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cases": rows, "query": filter.Query})
	}
}

func listEvalSetCasesHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
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
		filter, err := parseCaseResultsFilter(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_filter", err.Error())
			return
		}
		rows, err := manager.ListCases(r.Context(), caller, id, filter)
		if err != nil {
			writeEvalSetWarehouseError(logger, w, err)
			return
		}
		var nextCursor any
		if len(rows) > 0 {
			nextCursor = rows[len(rows)-1].ID
		}
		writeJSON(w, http.StatusOK, map[string]any{"cases": rows, "next_cursor": nextCursor})
	}
}

func reportEvalSetHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
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
		report, err := manager.Report(r.Context(), caller, id)
		if err != nil {
			writeEvalSetWarehouseError(logger, w, err)
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}

func compareEvalSetsHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, err := CallerFromContext(r.Context())
		if err != nil {
			writeAuthzError(w, err)
			return
		}
		aID, err := uuid.Parse(r.URL.Query().Get("a"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_a", "a must be a valid eval set UUID")
			return
		}
		bID, err := uuid.Parse(r.URL.Query().Get("b"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_b", "b must be a valid eval set UUID")
			return
		}
		result, err := manager.Compare(r.Context(), caller, aID, bID)
		if err != nil {
			writeEvalSetWarehouseError(logger, w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func exportEvalSetHandler(logger *slog.Logger, manager *EvalSetManager) http.HandlerFunc {
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
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "jsonl"
		}
		if format != "csv" && format != "jsonl" {
			writeError(w, http.StatusBadRequest, "invalid_format", "format must be csv or jsonl")
			return
		}
		flusher, _ := w.(http.Flusher)
		// Authorize and fetch the first page before committing a 200 so missing
		// or foreign-workspace eval sets return 404 instead of an empty export.
		firstPage, err := manager.ExportCases(r.Context(), caller, id, nil, 500)
		if err != nil {
			writeEvalSetWarehouseError(logger, w, err)
			return
		}
		if format == "csv" {
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"eval-set-%s.csv\"", id))
			w.WriteHeader(http.StatusOK)
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{
				"id", "eval_set_id", "run_id", "run_agent_id", "matrix_key", "pack_ref", "case_key",
				"agent_deployment_id", "model", "score", "correctness", "verdict", "cost_usd", "duration_ms",
				"failure_class", "transcript_text",
			})
			for _, row := range firstPage {
				_ = cw.Write(caseResultCSVRow(row))
			}
			cw.Flush()
			if flusher != nil {
				flusher.Flush()
			}
			if len(firstPage) < 500 {
				return
			}
			last := firstPage[len(firstPage)-1].ID
			cursor := &last
			for {
				rows, err := manager.ExportCases(r.Context(), caller, id, cursor, 500)
				if err != nil {
					logger.Error("export eval set csv failed", "error", err)
					return
				}
				if len(rows) == 0 {
					break
				}
				for _, row := range rows {
					_ = cw.Write(caseResultCSVRow(row))
				}
				cw.Flush()
				if flusher != nil {
					flusher.Flush()
				}
				last := rows[len(rows)-1].ID
				cursor = &last
				if len(rows) < 500 {
					break
				}
			}
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"eval-set-%s.jsonl\"", id))
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)
		for _, row := range firstPage {
			if err := enc.Encode(row); err != nil {
				return
			}
		}
		if flusher != nil {
			flusher.Flush()
		}
		if len(firstPage) < 500 {
			return
		}
		last := firstPage[len(firstPage)-1].ID
		cursor := &last
		for {
			rows, err := manager.ExportCases(r.Context(), caller, id, cursor, 500)
			if err != nil {
				logger.Error("export eval set jsonl failed", "error", err)
				return
			}
			if len(rows) == 0 {
				break
			}
			for _, row := range rows {
				if err := enc.Encode(row); err != nil {
					return
				}
			}
			if flusher != nil {
				flusher.Flush()
			}
			last := rows[len(rows)-1].ID
			cursor = &last
			if len(rows) < 500 {
				break
			}
		}
	}
}

func caseResultCSVRow(row repository.CaseResult) []string {
	score, correct, cost, dur, agent := "", "", "", "", ""
	if row.Score != nil {
		score = strconv.FormatFloat(*row.Score, 'f', -1, 64)
	}
	if row.Correctness != nil {
		correct = strconv.FormatBool(*row.Correctness)
	}
	if row.CostUSD != nil {
		cost = strconv.FormatFloat(*row.CostUSD, 'f', -1, 64)
	}
	if row.DurationMs != nil {
		dur = strconv.FormatInt(*row.DurationMs, 10)
	}
	if row.AgentDeploymentID != nil {
		agent = row.AgentDeploymentID.String()
	}
	setID := ""
	if row.EvalSetID != nil {
		setID = row.EvalSetID.String()
	}
	return []string{
		row.ID.String(), setID, row.RunID.String(), row.RunAgentID.String(), row.MatrixKey, row.PackRef, row.CaseKey,
		agent, row.Model, score, correct, row.Verdict, cost, dur, row.FailureClass, row.TranscriptText,
	}
}

func parseCaseResultsFilter(r *http.Request) (repository.ListCaseResultsFilter, error) {
	q := r.URL.Query()
	filter := repository.ListCaseResultsFilter{Limit: 50}
	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			return filter, fmt.Errorf("limit must be between 1 and 500")
		}
		filter.Limit = int32(n)
	}
	if v := strings.TrimSpace(q.Get("cursor")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filter, fmt.Errorf("cursor must be a UUID")
		}
		filter.CursorID = &id
	}
	if v := strings.TrimSpace(q.Get("agent")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filter, fmt.Errorf("agent must be a UUID")
		}
		filter.AgentDeploymentID = &id
	}
	if v := strings.TrimSpace(q.Get("pack")); v != "" {
		filter.PackRef = &v
	}
	if v := strings.TrimSpace(q.Get("verdict")); v != "" {
		filter.Verdict = &v
	}
	if v := strings.TrimSpace(q.Get("min_score")); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return filter, fmt.Errorf("min_score must be a number")
		}
		filter.MinScore = &f
	}
	if v := strings.TrimSpace(q.Get("max_score")); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return filter, fmt.Errorf("max_score must be a number")
		}
		filter.MaxScore = &f
	}
	return filter, nil
}

func writeEvalSetWarehouseError(logger *slog.Logger, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrEvalSetNotFound), errors.Is(err, ErrForbidden):
		// Workspace isolation: do not leak existence across tenants.
		writeError(w, http.StatusNotFound, "eval_set_not_found", "eval set not found")
	default:
		logger.Error("eval set warehouse request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
