package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
)

func TestPublicShareManager_CreateChallengePackShareAuthorizesWorkspace(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	versionID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.version = repository.RunnableChallengePackVersion{
		ID:              versionID,
		ChallengePackID: uuid.New(),
		WorkspaceID:     &workspaceID,
	}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")

	result, err := manager.CreateShareLink(ctx, callerWithWorkspace(workspaceID), CreateShareLinkInput{
		ResourceType: repository.PublicShareResourceChallengePackVersion,
		ResourceID:   versionID,
	})
	if err != nil {
		t.Fatalf("CreateShareLink returned error: %v", err)
	}
	if result.Share.ResourceType != repository.PublicShareResourceChallengePackVersion {
		t.Fatalf("resource type = %q", result.Share.ResourceType)
	}
	if result.Share.WorkspaceID != workspaceID || result.Share.OrganizationID != orgID {
		t.Fatalf("share scope = org %s workspace %s, want org %s workspace %s", result.Share.OrganizationID, result.Share.WorkspaceID, orgID, workspaceID)
	}
	if !strings.Contains(result.URL, "/share/") || result.Token == "" {
		t.Fatalf("token/url should be populated: token=%q url=%q", result.Token, result.URL)
	}
}

func TestPublicShareManager_ViewerCanCreatePrivateShareButCannotPublish(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	versionID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.version = repository.RunnableChallengePackVersion{
		ID:              versionID,
		ChallengePackID: uuid.New(),
		WorkspaceID:     &workspaceID,
	}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")
	viewer := callerWithWorkspaceRole(workspaceID, RoleWorkspaceViewer)

	if _, err := manager.CreateShareLink(ctx, viewer, CreateShareLinkInput{
		ResourceType: repository.PublicShareResourceChallengePackVersion,
		ResourceID:   versionID,
	}); err != nil {
		t.Fatalf("viewer private CreateShareLink returned error: %v", err)
	}

	if _, err := manager.CreateShareLink(ctx, viewer, CreateShareLinkInput{
		ResourceType:   repository.PublicShareResourceChallengePackVersion,
		ResourceID:     versionID,
		SearchIndexing: true,
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer indexable CreateShareLink error = %v, want ErrForbidden", err)
	}
}

func TestPublicShareManager_CreateIndexableShareReturnsCanonicalPublicationURL(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	versionID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.version = repository.RunnableChallengePackVersion{
		ID:              versionID,
		ChallengePackID: uuid.New(),
		WorkspaceID:     &workspaceID,
	}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")

	result, err := manager.CreateShareLink(ctx, callerWithWorkspace(workspaceID), CreateShareLinkInput{
		ResourceType:   repository.PublicShareResourceChallengePackVersion,
		ResourceID:     versionID,
		SearchIndexing: true,
	})
	if err != nil {
		t.Fatalf("CreateShareLink returned error: %v", err)
	}
	want := "https://www.agentclash.dev/publications/" + result.Share.ID.String()
	if result.PublicationURL != want {
		t.Fatalf("publication URL = %q, want %q", result.PublicationURL, want)
	}
	if strings.Contains(result.PublicationURL, result.Token) {
		t.Fatalf("publication URL leaked capability token: %q", result.PublicationURL)
	}
}

func TestPublicShareManager_CreateShareRejectsCrossWorkspaceCaller(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()
	runID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.run = domain.Run{
		ID:             runID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		Name:           "private run",
		Status:         domain.RunStatusCompleted,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.runScorecard = repository.RunScorecard{ID: uuid.New(), RunID: runID, Scorecard: json.RawMessage(`{"ok":true}`)}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")

	_, err := manager.CreateShareLink(ctx, callerWithWorkspace(otherWorkspaceID), CreateShareLinkInput{
		ResourceType: repository.PublicShareResourceRunScorecard,
		ResourceID:   runID,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("CreateShareLink error = %v, want ErrForbidden", err)
	}
}

func TestPublicShareManager_CreateRunShareAuthorizesWorkspace(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	runID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.run = domain.Run{
		ID:             runID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		Name:           "shared run",
		Status:         domain.RunStatusCompleted,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.runScorecard = repository.RunScorecard{ID: uuid.New(), RunID: runID, Scorecard: json.RawMessage(`{"summary":"public"}`)}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")

	result, err := manager.CreateShareLink(ctx, callerWithWorkspace(workspaceID), CreateShareLinkInput{
		ResourceType: repository.PublicShareResourceRunScorecard,
		ResourceID:   runID,
	})
	if err != nil {
		t.Fatalf("CreateShareLink returned error: %v", err)
	}
	if result.Share.ResourceType != repository.PublicShareResourceRunScorecard {
		t.Fatalf("resource type = %q, want run_scorecard", result.Share.ResourceType)
	}
	if result.Share.WorkspaceID != workspaceID || result.Share.OrganizationID != orgID {
		t.Fatalf("share scope = org %s workspace %s, want org %s workspace %s", result.Share.OrganizationID, result.Share.WorkspaceID, orgID, workspaceID)
	}
	if !strings.Contains(result.URL, "/share/") || result.Token == "" {
		t.Fatalf("token/url should be populated: token=%q url=%q", result.Token, result.URL)
	}
}

func TestPublicShareManager_GetPublicShareReturnsNarrowPayload(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	runID := uuid.New()
	createdByUserID := uuid.New()
	evalSessionID := uuid.New()
	agentDeploymentID := uuid.New()
	agentDeploymentSnapshotID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             uuid.New(),
		Key:            "share-key",
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		ResourceType:   repository.PublicShareResourceRunScorecard,
		ResourceID:     runID,
		IsActive:       true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.run = domain.Run{
		ID:                 runID,
		OrganizationID:     orgID,
		WorkspaceID:        workspaceID,
		Name:               "blog run",
		Status:             domain.RunStatusCompleted,
		EvalSessionID:      &evalSessionID,
		CreatedByUserID:    &createdByUserID,
		TemporalWorkflowID: ptrString("private-workflow"),
		TemporalRunID:      ptrString("private-run"),
		ExecutionPlan:      json.RawMessage(`{"credential_reference":"workspace-secret://OPENAI_API_KEY","api_key":"sk-live-private"}`),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.runAgent = domain.RunAgent{
		ID:                        uuid.New(),
		OrganizationID:            orgID,
		WorkspaceID:               workspaceID,
		RunID:                     runID,
		AgentDeploymentID:         agentDeploymentID,
		AgentDeploymentSnapshotID: agentDeploymentSnapshotID,
		LaneIndex:                 0,
		Label:                     "candidate",
		Status:                    domain.RunAgentStatusCompleted,
		CreatedAt:                 time.Now().UTC(),
		UpdatedAt:                 time.Now().UTC(),
	}
	repo.runScorecard = repository.RunScorecard{ID: uuid.New(), RunID: runID, Scorecard: json.RawMessage(`{"summary":"public"}`)}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")

	payload, err := manager.GetPublicShare(ctx, "share-key", "https://api.test")
	if err != nil {
		t.Fatalf("GetPublicShare returned error: %v", err)
	}
	if payload.Share.ResourceType != string(repository.PublicShareResourceRunScorecard) {
		t.Fatalf("share resource type = %q", payload.Share.ResourceType)
	}
	resourceEncoded, err := json.Marshal(payload.Resource)
	if err != nil {
		t.Fatalf("marshal payload resource: %v", err)
	}
	if string(resourceEncoded) == "" || !json.Valid(resourceEncoded) {
		t.Fatalf("payload resource is not valid JSON: %s", resourceEncoded)
	}
	responseEncoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload response: %v", err)
	}
	if contains := jsonContainsKey(resourceEncoded, "workspace_id"); contains {
		t.Fatalf("public payload leaked workspace_id: %s", resourceEncoded)
	}
	for _, key := range []string{
		"organization_id",
		"created_by_user_id",
		"eval_session_id",
		"temporal_workflow_id",
		"temporal_run_id",
		"execution_plan",
		"agent_deployment_id",
		"agent_deployment_snapshot_id",
		"credential_reference",
		"api_key",
	} {
		if jsonContainsKey(resourceEncoded, key) {
			t.Fatalf("public payload leaked %s: %s", key, resourceEncoded)
		}
	}
	for _, secret := range []string{
		orgID.String(),
		workspaceID.String(),
		createdByUserID.String(),
		evalSessionID.String(),
		agentDeploymentID.String(),
		agentDeploymentSnapshotID.String(),
		"workspace-secret://OPENAI_API_KEY",
		"sk-live-private",
		"private-workflow",
		"private-run",
	} {
		if strings.Contains(string(responseEncoded), secret) {
			t.Fatalf("public response leaked private value %q: %s", secret, responseEncoded)
		}
	}
}

func TestPublicShareManager_GetPublicShareKeepsReplayAgentDistinct(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	runID := uuid.New()
	firstAgentID := uuid.New()
	secondAgentID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             uuid.New(),
		Key:            "second-agent-replay",
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		ResourceType:   repository.PublicShareResourceRunAgentReplay,
		ResourceID:     secondAgentID,
		IsActive:       true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.run = domain.Run{
		ID:             runID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		Name:           "distinct agents",
		Status:         domain.RunStatusCompleted,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.runAgent = domain.RunAgent{
		ID:             secondAgentID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		RunID:          runID,
		LaneIndex:      1,
		Label:          "anthropic",
		Status:         domain.RunAgentStatusCompleted,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.replay = repository.RunAgentReplay{
		ID:         uuid.New(),
		RunAgentID: secondAgentID,
		Summary:    json.RawMessage(`{"steps":[{"headline":"second agent step"}]}`),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")

	payload, err := manager.GetPublicShare(ctx, "second-agent-replay", "https://api.test")
	if err != nil {
		t.Fatalf("GetPublicShare returned error: %v", err)
	}
	resource, ok := payload.Resource.(map[string]any)
	if !ok {
		t.Fatalf("resource type = %T, want map", payload.Resource)
	}
	runAgent, ok := resource["run_agent"].(map[string]any)
	if !ok {
		t.Fatalf("run_agent type = %T, want map", resource["run_agent"])
	}
	if got := runAgent["id"]; got != secondAgentID {
		t.Fatalf("shared replay agent id = %v, want %s and not %s", got, secondAgentID, firstAgentID)
	}
}

func TestPublicShareManager_GetPublicShareAgentTryoutReturnsNarrowPayload(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	tryoutID := uuid.New()
	createdByUserID := uuid.New()
	claimedByUserID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             uuid.New(),
		Key:            "tryout-share",
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		ResourceType:   repository.PublicShareResourceAgentTryout,
		ResourceID:     tryoutID,
		IsActive:       true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.agentTryout = repository.AgentTryout{
		ID:                     tryoutID,
		OrganizationID:         &orgID,
		WorkspaceID:            &workspaceID,
		TemplateSlug:           "meeting-minutes",
		Status:                 repository.AgentTryoutStatusCompleted,
		InputSnapshot:          json.RawMessage(`{"notes":"public notes"}`),
		TemplateSnapshot:       json.RawMessage(`{"slug":"meeting-minutes"}`),
		ToolPolicySnapshot:     json.RawMessage(`{"tools":["file_writer"]}`),
		EvaluationSpecSnapshot: json.RawMessage(`{"validators":[]}`),
		SelectedModelPolicy:    json.RawMessage(`{"mode":"hosted_default"}`),
		Summary:                json.RawMessage(`{"ok":true}`),
		RedactionStatus:        repository.AgentTryoutRedactionPassed,
		CostLimitUSD:           0.25,
		MaxDurationSeconds:     120,
		CreatedByUserID:        &createdByUserID,
		ClaimedByUserID:        &claimedByUserID,
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
	}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://agentclash.dev")

	payload, err := manager.GetPublicShare(ctx, "tryout-share", "https://api.test")
	if err != nil {
		t.Fatalf("GetPublicShare returned error: %v", err)
	}
	resourceEncoded, err := json.Marshal(payload.Resource)
	if err != nil {
		t.Fatalf("marshal payload resource: %v", err)
	}
	for _, key := range []string{
		"organization_id",
		"workspace_id",
		"created_by_user_id",
		"claimed_by_user_id",
		"claimed_at",
	} {
		if jsonContainsKey(resourceEncoded, key) {
			t.Fatalf("public tryout payload leaked %s: %s", key, resourceEncoded)
		}
	}
	responseEncoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload response: %v", err)
	}
	for _, secret := range []string{orgID.String(), workspaceID.String(), createdByUserID.String(), claimedByUserID.String()} {
		if strings.Contains(string(responseEncoded), secret) {
			t.Fatalf("public tryout response leaked private value %q: %s", secret, responseEncoded)
		}
	}
}

func TestPublicShareManager_GetPublicationUsesShareIDAndOmitsCapabilityData(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	runID := uuid.New()
	runAgentID := uuid.New()
	scorecardID := uuid.New()
	evaluationSpecID := uuid.New()
	nestedPrivateID := uuid.New()
	shareID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             shareID,
		Key:            "secret-capability-token",
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		ResourceType:   repository.PublicShareResourceRunScorecard,
		ResourceID:     runID,
		IsActive:       true,
		SearchIndexing: true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.run = domain.Run{
		ID:             runID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		Name:           "Published scorecard",
		Status:         domain.RunStatusCompleted,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.runAgent = domain.RunAgent{
		ID:        runAgentID,
		RunID:     runID,
		LaneIndex: 0,
		Label:     "candidate",
		Status:    domain.RunAgentStatusCompleted,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	repo.agentScorecard = repository.RunAgentScorecard{
		ID:               scorecardID,
		RunAgentID:       runAgentID,
		EvaluationSpecID: evaluationSpecID,
		Scorecard:        json.RawMessage(`{"passed":true,"workspace_id":"` + nestedPrivateID.String() + `"}`),
	}
	repo.runScorecard = repository.RunScorecard{
		ID:               scorecardID,
		RunID:            runID,
		EvaluationSpecID: evaluationSpecID,
		Scorecard:        json.RawMessage(`{"passed":true,"run_id":"` + runID.String() + `"}`),
	}

	payload, err := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "https://www.agentclash.dev").GetPublication(ctx, shareID)
	if err != nil {
		t.Fatalf("GetPublication returned error: %v", err)
	}
	if payload.Publication.CanonicalPath != "/publications/"+shareID.String() {
		t.Fatalf("canonical path = %q", payload.Publication.CanonicalPath)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal publication: %v", err)
	}
	for _, privateValue := range []string{
		"secret-capability-token",
		orgID.String(),
		workspaceID.String(),
		runID.String(),
		runAgentID.String(),
		scorecardID.String(),
		evaluationSpecID.String(),
		nestedPrivateID.String(),
	} {
		if strings.Contains(string(encoded), privateValue) {
			t.Fatalf("publication leaked private value %q: %s", privateValue, encoded)
		}
	}
}

func TestPublicShareManager_PublicationUpdatedAtTracksRenderedResource(t *testing.T) {
	ctx := context.Background()
	shareUpdatedAt := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	resourceUpdatedAt := shareUpdatedAt.Add(2 * time.Hour)
	orgID := uuid.New()
	workspaceID := uuid.New()
	runID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             uuid.New(),
		ResourceType:   repository.PublicShareResourceRunScorecard,
		ResourceID:     runID,
		IsActive:       true,
		SearchIndexing: true,
		CreatedAt:      shareUpdatedAt,
		UpdatedAt:      shareUpdatedAt,
	}
	repo.run = domain.Run{
		ID:             runID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		Name:           "Resource freshness",
		Status:         domain.RunStatusCompleted,
		UpdatedAt:      resourceUpdatedAt,
	}
	repo.runScorecard = repository.RunScorecard{
		ID:        uuid.New(),
		RunID:     runID,
		Scorecard: json.RawMessage(`{"passed":true}`),
		UpdatedAt: resourceUpdatedAt,
	}

	payload, err := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "").GetPublication(ctx, repo.share.ID)
	if err != nil {
		t.Fatalf("GetPublication returned error: %v", err)
	}
	if !payload.Publication.UpdatedAt.Equal(resourceUpdatedAt) {
		t.Fatalf("publication updated_at = %s, want resource update %s", payload.Publication.UpdatedAt, resourceUpdatedAt)
	}
}

func TestPublicShareManager_PublicationSerializerAllowlistByResourceType(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	resourceTypes := []repository.PublicShareResourceType{
		repository.PublicShareResourceChallengePackVersion,
		repository.PublicShareResourceRunScorecard,
		repository.PublicShareResourceRunAgentScorecard,
		repository.PublicShareResourceRunAgentReplay,
		repository.PublicShareResourceAgentTryout,
	}

	for _, resourceType := range resourceTypes {
		t.Run(string(resourceType), func(t *testing.T) {
			orgID := uuid.New()
			workspaceID := uuid.New()
			runID := uuid.New()
			runAgentID := uuid.New()
			resourceID := uuid.New()
			internalScorecardID := uuid.New()
			internalEvaluationID := uuid.New()
			repo := newFakePublicShareRepository(orgID, workspaceID)
			repo.share = repository.PublicShareLink{
				ID:             uuid.New(),
				Key:            "capability-secret",
				OrganizationID: orgID,
				WorkspaceID:    workspaceID,
				ResourceType:   resourceType,
				ResourceID:     resourceID,
				IsActive:       true,
				SearchIndexing: true,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			repo.version.ID = resourceID
			repo.run = domain.Run{
				ID:             runID,
				OrganizationID: orgID,
				WorkspaceID:    workspaceID,
				Name:           "Published run",
				Status:         domain.RunStatusCompleted,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			repo.runAgent = domain.RunAgent{
				ID:             runAgentID,
				OrganizationID: orgID,
				WorkspaceID:    workspaceID,
				RunID:          runID,
				Label:          "candidate",
				Status:         domain.RunAgentStatusCompleted,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			repo.runScorecard = repository.RunScorecard{
				ID:               internalScorecardID,
				RunID:            runID,
				EvaluationSpecID: internalEvaluationID,
				Scorecard:        json.RawMessage(`{"passed":true,"api_key":"nested-secret","run_id":"` + runID.String() + `"}`),
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			repo.agentScorecard = repository.RunAgentScorecard{
				ID:               internalScorecardID,
				RunAgentID:       runAgentID,
				EvaluationSpecID: internalEvaluationID,
				Scorecard:        json.RawMessage(`{"passed":true,"credential":"nested-secret"}`),
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			repo.replay = repository.RunAgentReplay{
				ID:         internalScorecardID,
				RunAgentID: runAgentID,
				Summary:    json.RawMessage(`{"steps":[],"workspace_id":"` + workspaceID.String() + `"}`),
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			repo.agentTryout = repository.AgentTryout{
				ID:              resourceID,
				OrganizationID:  &orgID,
				WorkspaceID:     &workspaceID,
				TemplateSlug:    "meeting-minutes",
				Status:          repository.AgentTryoutStatusCompleted,
				InputSnapshot:   json.RawMessage(`{"brief":"public","authorization":"nested-secret"}`),
				Summary:         json.RawMessage(`{"result":"public","user_id":"` + uuid.NewString() + `"}`),
				RedactionStatus: repository.AgentTryoutRedactionPassed,
				CreatedAt:       now,
				UpdatedAt:       now,
			}

			switch resourceType {
			case repository.PublicShareResourceChallengePackVersion:
				repo.share.ResourceID = repo.version.ID
			case repository.PublicShareResourceRunScorecard:
				repo.share.ResourceID = runID
			case repository.PublicShareResourceRunAgentScorecard, repository.PublicShareResourceRunAgentReplay:
				repo.share.ResourceID = runAgentID
			case repository.PublicShareResourceAgentTryout:
				repo.share.ResourceID = resourceID
			}

			payload, err := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "").GetPublication(ctx, repo.share.ID)
			if err != nil {
				t.Fatalf("GetPublication returned error: %v", err)
			}
			encoded, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal publication: %v", err)
			}
			if !strings.Contains(string(encoded), `"type":"`+string(resourceType)+`"`) {
				t.Fatalf("publication type missing from payload: %s", encoded)
			}
			for _, privateValue := range []string{
				"capability-secret",
				"nested-secret",
				orgID.String(),
				workspaceID.String(),
				resourceID.String(),
				runID.String(),
				runAgentID.String(),
				internalScorecardID.String(),
				internalEvaluationID.String(),
			} {
				if strings.Contains(string(encoded), privateValue) {
					t.Fatalf("publication leaked private value %q: %s", privateValue, encoded)
				}
			}
		})
	}
}

func TestSanitizePublicationValueRedactsSecretValuesAndEnvironmentFields(t *testing.T) {
	cleaned := sanitizePublicationValue(map[string]any{
		"summary":                 "called with Bearer abcdefghijklmnop",
		"provider_environment":    map[string]any{"region": "private"},
		"deployment_access_token": "must-not-appear",
		"reviewer_email":          "private@example.test",
		"nested": []any{
			map[string]any{"note": "key sk-abcdefghijklmnopqrstuvwxyz"},
		},
	})
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		t.Fatalf("marshal sanitized publication value: %v", err)
	}
	text := string(encoded)
	for _, secret := range []string{
		"abcdefghijklmnop",
		"provider_environment",
		"deployment_access_token",
		"private@example.test",
		"sk-abcdefghijklmnopqrstuvwxyz",
	} {
		if strings.Contains(text, secret) {
			t.Fatalf("sanitized publication leaked %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("sanitized publication should preserve explicit redaction markers: %s", text)
	}
}

func TestSanitizePublicationValueRedactsAllPEMPrivateKeyMarkers(t *testing.T) {
	for _, marker := range []string{
		"-----BEGIN EC PRIVATE KEY-----\nsecret\n-----END EC PRIVATE KEY-----",
		"-----BEGIN ENCRYPTED PRIVATE KEY-----\nsecret\n-----END ENCRYPTED PRIVATE KEY-----",
		"-----BEGIN PGP PRIVATE KEY BLOCK-----\nsecret\n-----END PGP PRIVATE KEY BLOCK-----",
	} {
		if got := redactPublicationSecretValues("prefix " + marker + " suffix"); got != "[REDACTED]" {
			t.Fatalf("redactPublicationSecretValues(%q) = %q, want full redaction", marker, got)
		}
	}
}

func TestPublicShareManager_GetPublicationFailsClosed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	base := repository.PublicShareLink{
		ID:             uuid.New(),
		ResourceType:   repository.PublicShareResourceRunScorecard,
		ResourceID:     uuid.New(),
		IsActive:       true,
		SearchIndexing: true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	revokedAt := now.Add(-time.Minute)
	expiredAt := now.Add(-time.Second)

	cases := map[string]func(*repository.PublicShareLink){
		"inactive":          func(share *repository.PublicShareLink) { share.IsActive = false },
		"indexing disabled": func(share *repository.PublicShareLink) { share.SearchIndexing = false },
		"revoked":           func(share *repository.PublicShareLink) { share.RevokedAt = &revokedAt },
		"expired":           func(share *repository.PublicShareLink) { share.ExpiresAt = &expiredAt },
		"unsupported": func(share *repository.PublicShareLink) {
			share.ResourceType = repository.PublicShareResourceType("unsupported")
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			repo := newFakePublicShareRepository(uuid.New(), uuid.New())
			repo.share = base
			mutate(&repo.share)
			manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "")
			manager.now = func() time.Time { return now }
			_, err := manager.GetPublication(ctx, repo.share.ID)
			if !errors.Is(err, repository.ErrPublicShareLinkNotFound) {
				t.Fatalf("GetPublication error = %v, want not found", err)
			}
		})
	}
}

func TestPublicShareManager_GetPublicationRejectsMalformedRenderedJSON(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	runID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             uuid.New(),
		ResourceType:   repository.PublicShareResourceRunScorecard,
		ResourceID:     runID,
		IsActive:       true,
		SearchIndexing: true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.run = domain.Run{
		ID:             runID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		Name:           "Malformed scorecard",
		Status:         domain.RunStatusCompleted,
	}
	repo.runScorecard = repository.RunScorecard{
		ID:        uuid.New(),
		RunID:     runID,
		Scorecard: json.RawMessage(`{"unterminated":`),
	}

	_, err := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "").GetPublication(ctx, repo.share.ID)
	if !errors.Is(err, repository.ErrPublicShareLinkNotFound) {
		t.Fatalf("GetPublication error = %v, want not found for malformed JSON", err)
	}
}

func TestPublicShareManager_DisablingIndexingImmediatelyHidesPublication(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	shareID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             shareID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		ResourceType:   repository.PublicShareResourceRunScorecard,
		ResourceID:     uuid.New(),
		IsActive:       true,
		SearchIndexing: true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "")
	if err := manager.SetShareSearchIndexing(ctx, callerWithWorkspace(workspaceID), shareID, false); err != nil {
		t.Fatalf("SetShareSearchIndexing returned error: %v", err)
	}
	if _, err := manager.GetPublication(ctx, shareID); !errors.Is(err, repository.ErrPublicShareLinkNotFound) {
		t.Fatalf("GetPublication error = %v, want not found", err)
	}
}

func TestPublicShareManager_ViewerCannotEnableIndexingButCanDisableIt(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	workspaceID := uuid.New()
	shareID := uuid.New()
	repo := newFakePublicShareRepository(orgID, workspaceID)
	repo.share = repository.PublicShareLink{
		ID:             shareID,
		OrganizationID: orgID,
		WorkspaceID:    workspaceID,
		ResourceType:   repository.PublicShareResourceRunScorecard,
		ResourceID:     uuid.New(),
		IsActive:       true,
		SearchIndexing: false,
	}
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "")
	viewer := callerWithWorkspaceRole(workspaceID, RoleWorkspaceViewer)

	if err := manager.SetShareSearchIndexing(ctx, viewer, shareID, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer enable indexing error = %v, want ErrForbidden", err)
	}
	if repo.share.SearchIndexing {
		t.Fatal("viewer enable indexing changed repository state")
	}

	repo.share.SearchIndexing = true
	if err := manager.SetShareSearchIndexing(ctx, viewer, shareID, false); err != nil {
		t.Fatalf("viewer disable indexing returned error: %v", err)
	}
	if repo.share.SearchIndexing {
		t.Fatal("viewer disable indexing did not change repository state")
	}
}

func TestPublicShareManager_ListPublicationsSkipsIneligibleRecords(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	repo := newFakePublicShareRepository(uuid.New(), uuid.New())
	eligibleID := uuid.New()
	ineligibleID := uuid.New()
	resourceID := uuid.New()
	repo.publications = []repository.PublicShareLink{
		{
			ID:             eligibleID,
			ResourceType:   repository.PublicShareResourceChallengePackVersion,
			ResourceID:     resourceID,
			IsActive:       true,
			SearchIndexing: true,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			ID:             ineligibleID,
			ResourceType:   repository.PublicShareResourceChallengePackVersion,
			ResourceID:     resourceID,
			IsActive:       true,
			SearchIndexing: false,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
	repo.version.ID = resourceID
	manager := NewPublicShareManager(NewCallerWorkspaceAuthorizer(), repo, "")
	manager.now = func() time.Time { return now }

	result, err := manager.ListPublications(ctx, nil, 2)
	if err != nil {
		t.Fatalf("ListPublications returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Publication.ID != eligibleID {
		t.Fatalf("items = %#v, want eligible publication only", result.Items)
	}
	if result.NextCursor == nil || *result.NextCursor != ineligibleID.String() {
		t.Fatalf("next cursor = %v, want final scanned record", result.NextCursor)
	}
}

func TestPublicationHandlersDisableCaching(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("list", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/public/publications", nil)
		response := httptest.NewRecorder()
		listPublicationsHandler(logger, noopPublicShareService{}).ServeHTTP(response, request)
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control = %q, want no-store", got)
		}
	})

	t.Run("detail 404", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/public/publications/not-a-uuid", nil)
		response := httptest.NewRecorder()
		getPublicationHandler(logger, noopPublicShareService{}).ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", response.Code)
		}
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control = %q, want no-store", got)
		}
	})
}

func callerWithWorkspace(workspaceID uuid.UUID) Caller {
	return callerWithWorkspaceRole(workspaceID, RoleWorkspaceAdmin)
}

func callerWithWorkspaceRole(workspaceID uuid.UUID, role string) Caller {
	userID := uuid.New()
	return Caller{
		UserID: userID,
		WorkspaceMemberships: map[uuid.UUID]WorkspaceMembership{
			workspaceID: {WorkspaceID: workspaceID, Role: role},
		},
		OrganizationMemberships: map[uuid.UUID]OrganizationMembership{},
	}
}

func ptrString(value string) *string {
	return &value
}

func jsonContainsKey(data []byte, key string) bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}
	return jsonValueContainsKey(value, key)
}

func jsonValueContainsKey(value any, key string) bool {
	switch typed := value.(type) {
	case map[string]any:
		if _, ok := typed[key]; ok {
			return true
		}
		for _, child := range typed {
			if jsonValueContainsKey(child, key) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if jsonValueContainsKey(child, key) {
				return true
			}
		}
	}
	return false
}

type fakePublicShareRepository struct {
	orgID          uuid.UUID
	workspaceID    uuid.UUID
	share          repository.PublicShareLink
	version        repository.RunnableChallengePackVersion
	run            domain.Run
	runAgent       domain.RunAgent
	runScorecard   repository.RunScorecard
	agentScorecard repository.RunAgentScorecard
	replay         repository.RunAgentReplay
	agentTryout    repository.AgentTryout
	runArtifacts   []repository.Artifact
	publications   []repository.PublicShareLink
}

func (r *fakePublicShareRepository) ListArtifactsByRunID(_ context.Context, _ uuid.UUID) ([]repository.Artifact, error) {
	return r.runArtifacts, nil
}

func newFakePublicShareRepository(orgID, workspaceID uuid.UUID) *fakePublicShareRepository {
	return &fakePublicShareRepository{orgID: orgID, workspaceID: workspaceID}
}

func (r *fakePublicShareRepository) CreatePublicShareLink(_ context.Context, params repository.CreatePublicShareLinkParams) (repository.PublicShareLink, error) {
	r.share = repository.PublicShareLink{
		ID:              uuid.New(),
		Key:             params.Key,
		OrganizationID:  params.OrganizationID,
		WorkspaceID:     params.WorkspaceID,
		ResourceType:    params.ResourceType,
		ResourceID:      params.ResourceID,
		CreatedByUserID: params.CreatedByUserID,
		IsActive:        true,
		SearchIndexing:  params.SearchIndexing,
		ExpiresAt:       params.ExpiresAt,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	return r.share, nil
}

func (r *fakePublicShareRepository) RevokePublicShareLink(_ context.Context, id uuid.UUID) error {
	if r.share.ID != id {
		return repository.ErrPublicShareLinkNotFound
	}
	r.share.IsActive = false
	return nil
}

func (r *fakePublicShareRepository) SetPublicShareSearchIndexing(_ context.Context, id uuid.UUID, enabled bool) error {
	if r.share.ID != id || !r.share.IsActive {
		return repository.ErrPublicShareLinkNotFound
	}
	r.share.SearchIndexing = enabled
	return nil
}

func (r *fakePublicShareRepository) GetPublicShareLinkByID(_ context.Context, id uuid.UUID) (repository.PublicShareLink, error) {
	if r.share.ID != id {
		return repository.PublicShareLink{}, repository.ErrPublicShareLinkNotFound
	}
	return r.share, nil
}

func (r *fakePublicShareRepository) GetIndexablePublicShareLinkByID(_ context.Context, id uuid.UUID) (repository.PublicShareLink, error) {
	if r.share.ID != id {
		return repository.PublicShareLink{}, repository.ErrPublicShareLinkNotFound
	}
	return r.share, nil
}

func (r *fakePublicShareRepository) ListIndexablePublicShareLinks(_ context.Context, _ *uuid.UUID, limit int) ([]repository.PublicShareLink, error) {
	if len(r.publications) <= limit {
		return append([]repository.PublicShareLink(nil), r.publications...), nil
	}
	return append([]repository.PublicShareLink(nil), r.publications[:limit]...), nil
}

func (r *fakePublicShareRepository) GetActivePublicShareLinkByKey(_ context.Context, key string) (repository.PublicShareLink, error) {
	if r.share.Key != key || !r.share.IsActive {
		return repository.PublicShareLink{}, repository.ErrPublicShareLinkNotFound
	}
	r.share.ViewCount++
	return r.share, nil
}

func (r *fakePublicShareRepository) GetOrganizationIDByWorkspaceID(_ context.Context, workspaceID uuid.UUID) (uuid.UUID, error) {
	if workspaceID != r.workspaceID {
		return uuid.Nil, repository.ErrWorkspaceSecretNotFound
	}
	return r.orgID, nil
}

func (r *fakePublicShareRepository) GetRunnableChallengePackVersionByID(_ context.Context, id uuid.UUID) (repository.RunnableChallengePackVersion, error) {
	if r.version.ID != id {
		return repository.RunnableChallengePackVersion{}, repository.ErrChallengePackVersionNotFound
	}
	return r.version, nil
}

func (r *fakePublicShareRepository) GetRunByID(_ context.Context, id uuid.UUID) (domain.Run, error) {
	if r.run.ID != id {
		return domain.Run{}, repository.ErrRunNotFound
	}
	return r.run, nil
}

func (r *fakePublicShareRepository) GetRunScorecardByRunID(_ context.Context, runID uuid.UUID) (repository.RunScorecard, error) {
	if r.runScorecard.RunID != runID {
		return repository.RunScorecard{}, repository.ErrRunScorecardNotFound
	}
	return r.runScorecard, nil
}

func (r *fakePublicShareRepository) GetRunAgentByID(_ context.Context, id uuid.UUID) (domain.RunAgent, error) {
	if r.runAgent.ID != id {
		return domain.RunAgent{}, repository.ErrRunAgentNotFound
	}
	return r.runAgent, nil
}

func (r *fakePublicShareRepository) ListRunAgentsByRunID(_ context.Context, runID uuid.UUID) ([]domain.RunAgent, error) {
	if r.run.ID != runID {
		return nil, repository.ErrRunNotFound
	}
	agents := []domain.RunAgent{}
	if r.runAgent.ID != uuid.Nil {
		agents = append(agents, r.runAgent)
	}
	return agents, nil
}

func (r *fakePublicShareRepository) GetRunAgentScorecardByRunAgentID(_ context.Context, runAgentID uuid.UUID) (repository.RunAgentScorecard, error) {
	if r.agentScorecard.RunAgentID != runAgentID {
		return repository.RunAgentScorecard{}, repository.ErrRunAgentScorecardNotFound
	}
	return r.agentScorecard, nil
}

func (r *fakePublicShareRepository) GetRunAgentReplayByRunAgentID(_ context.Context, runAgentID uuid.UUID) (repository.RunAgentReplay, error) {
	if r.replay.RunAgentID != runAgentID {
		return repository.RunAgentReplay{}, repository.ErrRunAgentReplayNotFound
	}
	return r.replay, nil
}

func (r *fakePublicShareRepository) GetAgentTryoutByID(_ context.Context, id uuid.UUID) (repository.AgentTryout, error) {
	if r.agentTryout.ID != id {
		return repository.AgentTryout{}, repository.ErrAgentTryoutNotFound
	}
	return r.agentTryout, nil
}

func (r *fakePublicShareRepository) GetPublicChallengePackVersionSnapshot(context.Context, uuid.UUID) (repository.PublicChallengePackVersionSnapshot, error) {
	return repository.PublicChallengePackVersionSnapshot{
		PackID:          uuid.New(),
		PackSlug:        "pack",
		PackName:        "Pack",
		PackFamily:      "evals",
		VersionID:       r.version.ID,
		VersionNumber:   1,
		LifecycleStatus: "runnable",
		Manifest:        json.RawMessage(`{"schema_version":1}`),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}, nil
}

func (r *fakePublicShareRepository) GetPublicRunScorecardSnapshot(_ context.Context, runID uuid.UUID) (repository.PublicRunScorecardSnapshot, error) {
	if r.run.ID != runID {
		return repository.PublicRunScorecardSnapshot{}, repository.ErrRunNotFound
	}
	agents := []domain.RunAgent{}
	if r.runAgent.ID != uuid.Nil {
		agents = append(agents, r.runAgent)
	}
	scorecards := []repository.RunAgentScorecard{}
	if r.agentScorecard.ID != uuid.Nil {
		scorecards = append(scorecards, r.agentScorecard)
	}
	return repository.PublicRunScorecardSnapshot{Run: r.run, Agents: agents, AgentScorecards: scorecards, Scorecard: r.runScorecard}, nil
}

func (r *fakePublicShareRepository) GetPublicRunAgentScorecardSnapshot(_ context.Context, runAgentID uuid.UUID) (repository.PublicRunAgentScorecardSnapshot, error) {
	if r.runAgent.ID != runAgentID {
		return repository.PublicRunAgentScorecardSnapshot{}, repository.ErrRunAgentNotFound
	}
	return repository.PublicRunAgentScorecardSnapshot{Run: r.run, RunAgent: r.runAgent, SiblingAgents: []domain.RunAgent{r.runAgent}, AgentScorecards: []repository.RunAgentScorecard{r.agentScorecard}, Scorecard: r.agentScorecard}, nil
}

func (r *fakePublicShareRepository) GetPublicRunAgentReplaySnapshot(_ context.Context, runAgentID uuid.UUID) (repository.PublicRunAgentReplaySnapshot, error) {
	if r.runAgent.ID != runAgentID {
		return repository.PublicRunAgentReplaySnapshot{}, repository.ErrRunAgentNotFound
	}
	return repository.PublicRunAgentReplaySnapshot{Run: r.run, RunAgent: r.runAgent, Replay: r.replay}, nil
}
