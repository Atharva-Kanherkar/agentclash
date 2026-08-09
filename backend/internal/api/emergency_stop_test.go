package api

import (
	"context"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/google/uuid"
)

type emergencyStopStore struct {
	*fakeEvalSetStore
	active []uuid.UUID
}

func (s *emergencyStopStore) ListActiveEvalSetIDsByWorkspaceID(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return append([]uuid.UUID(nil), s.active...), nil
}

func TestEmergencyStopCancelsActiveSets(t *testing.T) {
	ws := uuid.New()
	setID := uuid.New()
	base := newFakeEvalSetStore()
	base.sets[setID] = repository.EvalSet{
		ID:          setID,
		WorkspaceID: ws,
		Status:      domain.EvalSetStatusRunning,
	}
	store := &emergencyStopStore{fakeEvalSetStore: base, active: []uuid.UUID{setID}}
	starter := &fakeEvalSetStarter{}
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{}).WithPersistence(store, &fakeEvalSessionCreator{}, starter)

	resp, err := manager.EmergencyStop(context.Background(), Caller{UserID: uuid.New()}, ws)
	if err != nil {
		t.Fatalf("EmergencyStop: %v", err)
	}
	if resp.CancelledSets != 1 {
		t.Fatalf("cancelled_sets=%d resp=%+v", resp.CancelledSets, resp)
	}
	if store.sets[setID].Status != domain.EvalSetStatusCancelled {
		t.Fatalf("status=%s", store.sets[setID].Status)
	}
	if len(starter.canceled) != 1 {
		t.Fatalf("workflow cancels=%v", starter.canceled)
	}
	if resp.AuditEvent != "workspace.emergency_stop" {
		t.Fatalf("audit=%q", resp.AuditEvent)
	}
}
