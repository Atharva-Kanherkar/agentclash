package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// stubBaselineRepo is a small fake that drives BaselineManager without a
// real database. It lets tests assert which methods ran and with what args
// — more precise than an httptest against the real storage stack.
type stubBaselineRepo struct {
	upsertIn  repository.UpsertBaselineParams
	upsertOut repository.Baseline
	upsertErr error

	getIn  struct {
		WorkspaceID uuid.UUID
		Name        string
	}
	getOut repository.Baseline
	getErr error

	listIn  uuid.UUID
	listOut []repository.Baseline
	listErr error
}

func (s *stubBaselineRepo) UpsertBaseline(_ context.Context, p repository.UpsertBaselineParams) (repository.Baseline, error) {
	s.upsertIn = p
	return s.upsertOut, s.upsertErr
}

func (s *stubBaselineRepo) GetBaselineByName(_ context.Context, workspaceID uuid.UUID, name string) (repository.Baseline, error) {
	s.getIn.WorkspaceID = workspaceID
	s.getIn.Name = name
	return s.getOut, s.getErr
}

func (s *stubBaselineRepo) ListBaselinesByWorkspace(_ context.Context, workspaceID uuid.UUID) ([]repository.Baseline, error) {
	s.listIn = workspaceID
	return s.listOut, s.listErr
}

// baselineTestRouter wires just the three baseline routes under /v1 behind a
// no-op auth/authorization chain that injects a workspace ID into ctx. This
// stays narrow so tests exercise the handler logic, not the routing stack.
func baselineTestRouter(svc BaselineService, workspaceID uuid.UUID) http.Handler {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), &slog.HandlerOptions{Level: slog.LevelError}))
	authorizer := testAllowAllWorkspaceAuthorizer{}
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			ctx = context.WithValue(ctx, callerContextKey{}, Caller{
				UserID: uuid.New(),
				WorkspaceMemberships: map[uuid.UUID]WorkspaceMembership{
					workspaceID: {WorkspaceID: workspaceID, Role: RoleWorkspaceMember},
				},
			})
			ctx = context.WithValue(ctx, workspaceIDContextKey{}, workspaceID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/v1/workspaces/{workspaceID}/baselines", listBaselinesHandler(logger, svc, authorizer))
	r.Get("/v1/workspaces/{workspaceID}/baselines/{name}", getBaselineHandler(logger, svc, authorizer))
	r.Post("/v1/workspaces/{workspaceID}/baselines/{name}", upsertBaselineHandler(logger, svc, authorizer))
	return r
}

type testAllowAllWorkspaceAuthorizer struct{}

func (testAllowAllWorkspaceAuthorizer) AuthorizeWorkspace(_ context.Context, _ Caller, _ uuid.UUID) error {
	return nil
}

func TestBaselineUpsertRoundTripsBodyAndURLName(t *testing.T) {
	workspaceID := uuid.New()
	repo := &stubBaselineRepo{
		upsertOut: repository.Baseline{
			ID:                uuid.New(),
			WorkspaceID:       workspaceID,
			Name:              "main",
			PackVersionID:     uuid.New(),
			RunID:             uuid.New(),
			ScorecardSnapshot: json.RawMessage(`{"winner":"claude-4.7"}`),
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		},
	}
	svc := NewBaselineManager(repo)
	router := baselineTestRouter(svc, workspaceID)

	body := bytes.NewBufferString(`{
		"pack_version_id":"11111111-1111-1111-1111-111111111111",
		"run_id":"22222222-2222-2222-2222-222222222222",
		"scorecard_snapshot":{"winner":"claude-4.7","tasks":3}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/workspaces/"+workspaceID.String()+"/baselines/main", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if repo.upsertIn.Name != "main" {
		t.Fatalf("upsert Name = %q, want main", repo.upsertIn.Name)
	}
	if repo.upsertIn.WorkspaceID != workspaceID {
		t.Fatalf("upsert WorkspaceID = %s, want %s", repo.upsertIn.WorkspaceID, workspaceID)
	}
	if repo.upsertIn.PackVersionID.String() != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("upsert PackVersionID = %s", repo.upsertIn.PackVersionID)
	}
	if !strings.Contains(string(repo.upsertIn.ScorecardSnapshot), "claude-4.7") {
		t.Fatalf("scorecard_snapshot not forwarded: %s", repo.upsertIn.ScorecardSnapshot)
	}

	var resp BaselineResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "main" {
		t.Fatalf("response Name = %q, want main", resp.Name)
	}
}

func TestBaselineUpsertRejectsMalformedBody(t *testing.T) {
	workspaceID := uuid.New()
	repo := &stubBaselineRepo{}
	router := baselineTestRouter(NewBaselineManager(repo), workspaceID)

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"empty body", ``, http.StatusBadRequest, "invalid_pack_version_id"},
		{"bad pack id", `{"pack_version_id":"not-a-uuid","run_id":"22222222-2222-2222-2222-222222222222"}`, http.StatusBadRequest, "invalid_pack_version_id"},
		{"bad run id", `{"pack_version_id":"11111111-1111-1111-1111-111111111111","run_id":"nope"}`, http.StatusBadRequest, "invalid_run_id"},
		{"bad json", `{`, http.StatusBadRequest, "invalid_json"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/workspaces/"+workspaceID.String()+"/baselines/main", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			var env errorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if env.Error.Code != tc.wantCode {
				t.Fatalf("error.code = %q, want %q", env.Error.Code, tc.wantCode)
			}
		})
	}
}

func TestBaselineGetReturnsNotFoundOnMissing(t *testing.T) {
	workspaceID := uuid.New()
	repo := &stubBaselineRepo{getErr: repository.ErrBaselineNotFound}
	router := baselineTestRouter(NewBaselineManager(repo), workspaceID)

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+workspaceID.String()+"/baselines/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	var env errorEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "baseline_not_found" {
		t.Fatalf("error.code = %q, want baseline_not_found", env.Error.Code)
	}
	if repo.getIn.Name != "does-not-exist" {
		t.Fatalf("repo was asked for %q, expected does-not-exist", repo.getIn.Name)
	}
}

func TestBaselineGetReturnsBodyOnHit(t *testing.T) {
	workspaceID := uuid.New()
	repo := &stubBaselineRepo{
		getOut: repository.Baseline{
			ID:                uuid.New(),
			WorkspaceID:       workspaceID,
			Name:              "main",
			PackVersionID:     uuid.New(),
			RunID:             uuid.New(),
			ScorecardSnapshot: json.RawMessage(`{"winner":"gpt-5"}`),
			CreatedAt:         time.Now().UTC(),
			UpdatedAt:         time.Now().UTC(),
		},
	}
	router := baselineTestRouter(NewBaselineManager(repo), workspaceID)

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+workspaceID.String()+"/baselines/main", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var resp BaselineResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "main" {
		t.Fatalf("Name = %q, want main", resp.Name)
	}
	if !strings.Contains(string(resp.ScorecardSnapshot), "gpt-5") {
		t.Fatalf("ScorecardSnapshot lost: %s", resp.ScorecardSnapshot)
	}
}

func TestBaselineListWrapsItems(t *testing.T) {
	workspaceID := uuid.New()
	repo := &stubBaselineRepo{
		listOut: []repository.Baseline{
			{ID: uuid.New(), WorkspaceID: workspaceID, Name: "main", PackVersionID: uuid.New(), RunID: uuid.New(), ScorecardSnapshot: json.RawMessage(`{}`)},
			{ID: uuid.New(), WorkspaceID: workspaceID, Name: "staging", PackVersionID: uuid.New(), RunID: uuid.New(), ScorecardSnapshot: json.RawMessage(`{}`)},
		},
	}
	router := baselineTestRouter(NewBaselineManager(repo), workspaceID)

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+workspaceID.String()+"/baselines", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var resp ListBaselinesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(resp.Items))
	}
	names := []string{resp.Items[0].Name, resp.Items[1].Name}
	if !((names[0] == "main" && names[1] == "staging") || (names[0] == "staging" && names[1] == "main")) {
		t.Fatalf("unexpected items: %+v", resp.Items)
	}
	if repo.listIn != workspaceID {
		t.Fatalf("list called with %s, want %s", repo.listIn, workspaceID)
	}
}

func TestBaselineUpsertSurfacesEmptyNameFromRepo(t *testing.T) {
	workspaceID := uuid.New()
	repo := &stubBaselineRepo{upsertErr: repository.ErrBaselineNameRequired}
	router := baselineTestRouter(NewBaselineManager(repo), workspaceID)

	body := strings.NewReader(`{"pack_version_id":"11111111-1111-1111-1111-111111111111","run_id":"22222222-2222-2222-2222-222222222222"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/workspaces/"+workspaceID.String()+"/baselines/ok", body)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var env errorEnvelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "invalid_baseline_name" {
		t.Fatalf("error.code = %q, want invalid_baseline_name", env.Error.Code)
	}
}

// Sanity: constructing the manager and delegating to a real repository stub
// shouldn't add surprises (ensures baselineToResponse's nil handling works).
func TestBaselineManagerPassthroughKeepsSnapshot(t *testing.T) {
	wsID := uuid.New()
	repo := &stubBaselineRepo{
		upsertOut: repository.Baseline{
			WorkspaceID:       wsID,
			Name:              "main",
			ScorecardSnapshot: nil, // emulate a zero-length snapshot from the DB
		},
	}
	mgr := NewBaselineManager(repo)
	resp, err := mgr.Upsert(context.Background(), UpsertBaselineInput{
		WorkspaceID:       wsID,
		Name:              "main",
		PackVersionID:     uuid.New(),
		RunID:             uuid.New(),
		ScorecardSnapshot: nil,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if string(resp.ScorecardSnapshot) != "{}" {
		t.Fatalf("expected empty-object fallback, got %s", resp.ScorecardSnapshot)
	}
}

// Exercised indirectly — keep the import alive.
var _ = errors.New
