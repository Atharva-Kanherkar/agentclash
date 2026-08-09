package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeCaseResultsStore struct {
	rows map[uuid.UUID][]repository.CaseResult
	agg  map[uuid.UUID][]repository.CaseResultAxisAggregate
}

func (f *fakeCaseResultsStore) ListCaseResults(_ context.Context, filter repository.ListCaseResultsFilter) ([]repository.CaseResult, error) {
	return f.rows[filter.EvalSetID], nil
}

func (f *fakeCaseResultsStore) SearchCaseResults(_ context.Context, filter repository.ListCaseResultsFilter) ([]repository.CaseResult, error) {
	out := make([]repository.CaseResult, 0)
	for _, row := range f.rows[filter.EvalSetID] {
		if strings.Contains(strings.ToLower(row.TranscriptText), strings.ToLower(filter.Query)) {
			row.Snippet = repository.HighlightSnippet(row.TranscriptText, filter.Query, 40)
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeCaseResultsStore) ListCaseResultsForExport(ctx context.Context, workspaceID, evalSetID uuid.UUID, cursor *uuid.UUID, limit int32) ([]repository.CaseResult, error) {
	return f.ListCaseResults(ctx, repository.ListCaseResultsFilter{EvalSetID: evalSetID, WorkspaceID: workspaceID, CursorID: cursor, Limit: limit})
}

func (f *fakeCaseResultsStore) AggregateCaseResults(_ context.Context, _, evalSetID uuid.UUID) ([]repository.CaseResultAxisAggregate, error) {
	return f.agg[evalSetID], nil
}

func (f *fakeCaseResultsStore) ListCaseResultsForCompare(_ context.Context, _, evalSetID uuid.UUID) ([]repository.CaseResultCompareRow, error) {
	out := make([]repository.CaseResultCompareRow, 0)
	for _, row := range f.rows[evalSetID] {
		out = append(out, repository.CaseResultCompareRow{
			MatrixKey: row.MatrixKey,
			PackRef:   row.PackRef,
			CaseKey:   row.CaseKey,
			Score:     row.Score,
			Verdict:   row.Verdict,
			RunID:     row.RunID,
		})
	}
	return out, nil
}

func TestEvalSetSearchReturnsSnippets(t *testing.T) {
	setID := uuid.New()
	ws := uuid.New()
	store := newFakeEvalSetStore()
	store.sets[setID] = repository.EvalSet{ID: setID, WorkspaceID: ws, Status: domain.EvalSetStatusCompleted}
	score := 0.9
	cases := &fakeCaseResultsStore{rows: map[uuid.UUID][]repository.CaseResult{
		setID: {{
			ID:             uuid.New(),
			EvalSetID:      &setID,
			MatrixKey:      "p/a/1",
			TranscriptText: "customer asked for a refund on order 12",
			Score:          &score,
			Verdict:        "pass",
		}},
	}}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(store, &fakeEvalSessionCreator{}, &fakeEvalSetStarter{}).WithCaseResults(cases)

	req := httptest.NewRequest(http.MethodGet, "/v1/eval-sets/"+setID.String()+"/search?q=refund", nil)
	req = req.WithContext(context.WithValue(req.Context(), callerContextKey{}, Caller{UserID: uuid.New()}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("evalSetID", setID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	searchEvalSetCasesHandler(discardLogger(), manager).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "refund") {
		t.Fatalf("expected snippet with refund: %s", rr.Body.String())
	}
}

func TestEvalSetWarehouseForeignWorkspace404(t *testing.T) {
	setID := uuid.New()
	store := newFakeEvalSetStore()
	store.sets[setID] = repository.EvalSet{ID: setID, WorkspaceID: uuid.New(), Status: domain.EvalSetStatusCompleted}
	manager := NewEvalSetManager(denyWorkspaceAuthorizer{}).WithPersistence(store, &fakeEvalSessionCreator{}, &fakeEvalSetStarter{}).WithCaseResults(&fakeCaseResultsStore{})

	req := httptest.NewRequest(http.MethodGet, "/v1/eval-sets/"+setID.String()+"/cases", nil)
	req = req.WithContext(context.WithValue(req.Context(), callerContextKey{}, Caller{UserID: uuid.New()}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("evalSetID", setID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	listEvalSetCasesHandler(discardLogger(), manager).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestEvalSetCompareFlagsRegressions(t *testing.T) {
	aID, bID := uuid.New(), uuid.New()
	ws := uuid.New()
	store := newFakeEvalSetStore()
	store.sets[aID] = repository.EvalSet{ID: aID, WorkspaceID: ws, Status: domain.EvalSetStatusCompleted}
	store.sets[bID] = repository.EvalSet{ID: bID, WorkspaceID: ws, Status: domain.EvalSetStatusCompleted}
	aScore, bScore := 0.9, 0.4
	cases := &fakeCaseResultsStore{rows: map[uuid.UUID][]repository.CaseResult{
		aID: {{MatrixKey: "p/a/1", CaseKey: "p/a/1", Score: &aScore, RunID: uuid.New()}},
		bID: {{MatrixKey: "p/a/1", CaseKey: "p/a/1", Score: &bScore, RunID: uuid.New()}},
	}}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(store, &fakeEvalSessionCreator{}, &fakeEvalSetStarter{}).WithCaseResults(cases)
	result, err := manager.Compare(context.Background(), Caller{UserID: uuid.New()}, aID, bID)
	if err != nil {
		t.Fatal(err)
	}
	regs, ok := result["regressions"].([]map[string]any)
	if !ok || len(regs) != 1 {
		t.Fatalf("regressions = %#v", result["regressions"])
	}
}

func TestEvalSetReportMarginals(t *testing.T) {
	setID := uuid.New()
	ws := uuid.New()
	store := newFakeEvalSetStore()
	store.sets[setID] = repository.EvalSet{ID: setID, WorkspaceID: ws, Status: domain.EvalSetStatusCompleted}
	cases := &fakeCaseResultsStore{agg: map[uuid.UUID][]repository.CaseResultAxisAggregate{
		setID: {{PackRef: "p", AgentDeploymentID: "d", N: 2, Wins: 2, MeanScore: 0.8}},
	}}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(store, &fakeEvalSessionCreator{}, &fakeEvalSetStarter{}).WithCaseResults(cases)
	report, err := manager.Report(context.Background(), Caller{UserID: uuid.New()}, setID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(report)
	if !strings.Contains(string(raw), `"n":2`) && !strings.Contains(string(raw), `"n": 2`) {
		t.Fatalf("report = %s", raw)
	}
}

func TestHighlightSnippet(t *testing.T) {
	got := repository.HighlightSnippet("aaa refund bbb", "refund", 3)
	if !strings.Contains(got, "refund") {
		t.Fatalf("snippet = %q", got)
	}
}

type denyWorkspaceAuthorizer struct{}

func (denyWorkspaceAuthorizer) AuthorizeWorkspace(context.Context, Caller, uuid.UUID) error {
	return ErrForbidden
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
