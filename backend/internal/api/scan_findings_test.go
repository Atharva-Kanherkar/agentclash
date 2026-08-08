package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeScanFindingStore struct {
	findings map[uuid.UUID]repository.ScanFinding
	counts   map[uuid.UUID]map[string]int64
}

func (f *fakeScanFindingStore) ListScanFindingsByEvalSetID(_ context.Context, evalSetID uuid.UUID, _, _ *string, _, _ int32) ([]repository.ScanFinding, error) {
	out := make([]repository.ScanFinding, 0)
	for _, finding := range f.findings {
		if finding.EvalSetID == evalSetID {
			out = append(out, finding)
		}
	}
	return out, nil
}

func (f *fakeScanFindingStore) GetScanFindingByID(_ context.Context, id uuid.UUID) (repository.ScanFinding, error) {
	finding, ok := f.findings[id]
	if !ok {
		return repository.ScanFinding{}, repository.ErrScanFindingNotFound
	}
	return finding, nil
}

func (f *fakeScanFindingStore) UpdateScanFindingStatus(_ context.Context, id uuid.UUID, status string, updatedBy *uuid.UUID) (repository.ScanFinding, error) {
	finding, ok := f.findings[id]
	if !ok {
		return repository.ScanFinding{}, repository.ErrScanFindingNotFound
	}
	finding.Status = status
	finding.StatusUpdatedBy = updatedBy
	now := time.Now().UTC()
	finding.StatusUpdatedAt = &now
	f.findings[id] = finding
	return finding, nil
}

func (f *fakeScanFindingStore) CountScanFindingsBySeverity(_ context.Context, evalSetID uuid.UUID) (map[string]int64, error) {
	if f.counts == nil {
		return map[string]int64{}, nil
	}
	return f.counts[evalSetID], nil
}

type fakeScanStarter struct {
	started []uuid.UUID
}

func (f *fakeScanStarter) StartScanEvalSetWorkflow(_ context.Context, evalSetID uuid.UUID, _ []string) error {
	f.started = append(f.started, evalSetID)
	return nil
}

func TestScanFindingsWorkspaceIsolation(t *testing.T) {
	setID := uuid.New()
	store := newFakeEvalSetStore()
	store.sets[setID] = repository.EvalSet{ID: setID, WorkspaceID: uuid.New(), Status: domain.EvalSetStatusCompleted}
	findings := &fakeScanFindingStore{findings: map[uuid.UUID]repository.ScanFinding{}}
	manager := NewEvalSetManager(denyWorkspaceAuthorizer{}).
		WithPersistence(store, &fakeEvalSessionCreator{}, &fakeEvalSetStarter{}).
		WithScanFindings(findings, &fakeScanStarter{})

	req := httptest.NewRequest(http.MethodGet, "/v1/eval-sets/"+setID.String()+"/findings", nil)
	req = req.WithContext(context.WithValue(req.Context(), callerContextKey{}, Caller{UserID: uuid.New()}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("evalSetID", setID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	listFindingsHandler(discardLogger(), manager).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want forbidden/not found", rr.Code)
	}
}

func TestUpdateFindingStatusAuditsActor(t *testing.T) {
	setID := uuid.New()
	ws := uuid.New()
	findingID := uuid.New()
	actor := uuid.New()
	store := newFakeEvalSetStore()
	store.sets[setID] = repository.EvalSet{ID: setID, WorkspaceID: ws, Status: domain.EvalSetStatusCompleted}
	findings := &fakeScanFindingStore{findings: map[uuid.UUID]repository.ScanFinding{
		findingID: {
			ID:          findingID,
			WorkspaceID: ws,
			EvalSetID:   setID,
			CaseKey:     "c1",
			Scanner:     "reward-hacking",
			Status:      "open",
		},
	}}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).
		WithPersistence(store, &fakeEvalSessionCreator{}, &fakeEvalSetStarter{}).
		WithScanFindings(findings, &fakeScanStarter{})

	body, _ := json.Marshal(map[string]string{"status": "triaged"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/scan-findings/"+findingID.String(), bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), callerContextKey{}, Caller{UserID: actor}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("findingID", findingID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	updateFindingStatusHandler(discardLogger(), manager).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	updated := findings.findings[findingID]
	if updated.Status != "triaged" {
		t.Fatalf("status = %q", updated.Status)
	}
	if updated.StatusUpdatedBy == nil || *updated.StatusUpdatedBy != actor {
		t.Fatalf("status_updated_by = %v, want %s", updated.StatusUpdatedBy, actor)
	}
	if updated.StatusUpdatedAt == nil {
		t.Fatal("expected status_updated_at")
	}
}

func TestStartScanAccepted(t *testing.T) {
	setID := uuid.New()
	ws := uuid.New()
	store := newFakeEvalSetStore()
	store.sets[setID] = repository.EvalSet{ID: setID, WorkspaceID: ws, Status: domain.EvalSetStatusCompleted}
	starter := &fakeScanStarter{}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).
		WithPersistence(store, &fakeEvalSessionCreator{}, &fakeEvalSetStarter{}).
		WithScanFindings(&fakeScanFindingStore{findings: map[uuid.UUID]repository.ScanFinding{}}, starter)

	body, _ := json.Marshal(map[string]any{"scanners": []string{"reward-hacking"}})
	req := httptest.NewRequest(http.MethodPost, "/v1/eval-sets/"+setID.String()+"/scan", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), callerContextKey{}, Caller{UserID: uuid.New()}))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("evalSetID", setID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	scanEvalSetHandler(discardLogger(), manager).ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if len(starter.started) != 1 || starter.started[0] != setID {
		t.Fatalf("started = %v", starter.started)
	}
}
