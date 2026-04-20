package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/google/uuid"
)

func TestRunReadManagerGetEvalSessionReturnsAuthorizedSession(t *testing.T) {
	workspaceID := uuid.New()
	sessionID := uuid.New()
	runID := uuid.New()
	caller := Caller{
		UserID: uuid.New(),
		WorkspaceMemberships: map[uuid.UUID]WorkspaceMembership{
			workspaceID: {WorkspaceID: workspaceID, Role: "workspace_member"},
		},
	}

	manager := NewRunReadManager(NewCallerWorkspaceAuthorizer(), &fakeRunReadRepository{
		evalSession: repository.EvalSessionWithRuns{
			Session: domain.EvalSession{
				ID:          sessionID,
				Status:      domain.EvalSessionStatusQueued,
				Repetitions: 2,
			},
			Runs: []domain.Run{
				{
					ID:            runID,
					WorkspaceID:   workspaceID,
					EvalSessionID: &sessionID,
					Status:        domain.RunStatusQueued,
					CreatedAt:     time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
					UpdatedAt:     time.Date(2026, 4, 20, 12, 1, 0, 0, time.UTC),
				},
			},
		},
	})

	result, err := manager.GetEvalSession(context.Background(), caller, sessionID)
	if err != nil {
		t.Fatalf("GetEvalSession returned error: %v", err)
	}
	if result.Session.ID != sessionID {
		t.Fatalf("session id = %s, want %s", result.Session.ID, sessionID)
	}
	if result.Summary.RunCounts.Queued != 1 {
		t.Fatalf("queued run count = %d, want 1", result.Summary.RunCounts.Queued)
	}
	if !slices.Contains(result.EvidenceWarnings, "aggregate result unavailable: eval session has not reached aggregation yet") {
		t.Fatalf("evidence warnings = %v, want aggregate unavailable warning", result.EvidenceWarnings)
	}
}

func TestRunReadManagerGetEvalSessionRejectsForbiddenWorkspaceAccess(t *testing.T) {
	workspaceID := uuid.New()
	manager := NewRunReadManager(NewCallerWorkspaceAuthorizer(), &fakeRunReadRepository{
		evalSession: repository.EvalSessionWithRuns{
			Session: domain.EvalSession{
				ID:          uuid.New(),
				Status:      domain.EvalSessionStatusQueued,
				Repetitions: 1,
			},
			Runs: []domain.Run{
				{
					ID:          uuid.New(),
					WorkspaceID: workspaceID,
					Status:      domain.RunStatusQueued,
				},
			},
		},
	})

	_, err := manager.GetEvalSession(context.Background(), Caller{
		UserID:               uuid.New(),
		WorkspaceMemberships: map[uuid.UUID]WorkspaceMembership{},
	}, uuid.New())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestGetEvalSessionEndpointReturnsDetail(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	sessionID := uuid.New()
	runID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/v1/eval-sessions/"+sessionID.String(), nil)
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerWorkspaceMemberships, workspaceID.String()+":workspace_member")
	recorder := httptest.NewRecorder()

	newRouter("dev", nil,
		testLogger(t),
		NewDevelopmentAuthenticator(),
		NewCallerWorkspaceAuthorizer(),
		nil,
		0,
		stubRunCreationService{},
		&fakeRunReadService{
			getEvalSessionResult: GetEvalSessionResult{
				Session: domain.EvalSession{
					ID:          sessionID,
					Status:      domain.EvalSessionStatusQueued,
					Repetitions: 2,
					CreatedAt:   time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2026, 4, 20, 12, 1, 0, 0, time.UTC),
				},
				Runs: []domain.Run{
					{
						ID:               runID,
						WorkspaceID:      workspaceID,
						EvalSessionID:    &sessionID,
						OfficialPackMode: domain.OfficialPackModeFull,
						Name:             "Repeated Eval [1/2]",
						Status:           domain.RunStatusQueued,
						ExecutionMode:    "single_agent",
						CreatedAt:        time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
						UpdatedAt:        time.Date(2026, 4, 20, 12, 1, 0, 0, time.UTC),
					},
				},
				Summary: EvalSessionSummary{
					RunCounts: EvalSessionRunCounts{Total: 1, Queued: 1},
				},
				EvidenceWarnings: []string{"aggregate result unavailable: eval session has not reached aggregation yet"},
			},
		},
		&fakeReplayReadService{},
		stubHostedRunIngestionService{},
		nil,
		stubAgentDeploymentReadService{},
		stubChallengePackReadService{},
		stubAgentBuildService{},
		noopReleaseGateService{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response getEvalSessionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.EvalSession.ID != sessionID {
		t.Fatalf("session id = %s, want %s", response.EvalSession.ID, sessionID)
	}
	if len(response.Runs) != 1 || response.Runs[0].ID != runID {
		t.Fatalf("runs = %#v, want one run %s", response.Runs, runID)
	}
	if response.Summary.RunCounts.Queued != 1 {
		t.Fatalf("queued run count = %d, want 1", response.Summary.RunCounts.Queued)
	}
	if string(response.AggregateResult) != "null" {
		t.Fatalf("aggregate_result = %s, want null", string(response.AggregateResult))
	}
}

func TestListEvalSessionsEndpointRequiresWorkspaceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/eval-sessions", nil)
	req.Header.Set(headerUserID, uuid.New().String())
	recorder := httptest.NewRecorder()

	newRouter("dev", nil,
		testLogger(t),
		NewDevelopmentAuthenticator(),
		NewCallerWorkspaceAuthorizer(),
		nil,
		0,
		stubRunCreationService{},
		&fakeRunReadService{},
		&fakeReplayReadService{},
		stubHostedRunIngestionService{},
		nil,
		stubAgentDeploymentReadService{},
		stubChallengePackReadService{},
		stubAgentBuildService{},
		noopReleaseGateService{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestListEvalSessionsEndpointReturnsItems(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	sessionID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/v1/eval-sessions?workspace_id="+workspaceID.String()+"&limit=5", nil)
	req.Header.Set(headerUserID, userID.String())
	req.Header.Set(headerWorkspaceMemberships, workspaceID.String()+":workspace_member")
	recorder := httptest.NewRecorder()

	newRouter("dev", nil,
		testLogger(t),
		NewDevelopmentAuthenticator(),
		NewCallerWorkspaceAuthorizer(),
		nil,
		0,
		stubRunCreationService{},
		&fakeRunReadService{
			listEvalSessionsResult: ListEvalSessionsResult{
				Items: []GetEvalSessionResult{
					{
						Session: domain.EvalSession{
							ID:          sessionID,
							Status:      domain.EvalSessionStatusQueued,
							Repetitions: 3,
						},
						Summary: EvalSessionSummary{
							RunCounts: EvalSessionRunCounts{Total: 3, Queued: 3},
						},
						EvidenceWarnings: []string{"aggregate result unavailable: eval session has not reached aggregation yet"},
					},
				},
			},
		},
		&fakeReplayReadService{},
		stubHostedRunIngestionService{},
		nil,
		stubAgentDeploymentReadService{},
		stubChallengePackReadService{},
		stubAgentBuildService{},
		noopReleaseGateService{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response listEvalSessionsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(response.Items))
	}
	if response.Items[0].EvalSession.ID != sessionID {
		t.Fatalf("session id = %s, want %s", response.Items[0].EvalSession.ID, sessionID)
	}
	if response.Items[0].Summary.RunCounts.Queued != 3 {
		t.Fatalf("queued run count = %d, want 3", response.Items[0].Summary.RunCounts.Queued)
	}
}
