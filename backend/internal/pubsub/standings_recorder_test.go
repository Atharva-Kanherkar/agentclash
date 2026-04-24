package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/backend/internal/runevents"
	"github.com/google/uuid"
)

type fakeStandingsStore struct {
	mu      sync.Mutex
	calls   []standingsCall
	nextErr error
}

type standingsCall struct {
	runID   uuid.UUID
	updates StandingsEntry
}

func (f *fakeStandingsStore) Update(_ context.Context, runID uuid.UUID, updates StandingsEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, standingsCall{runID: runID, updates: updates})
	return f.nextErr
}

func (f *fakeStandingsStore) Snapshot(context.Context, uuid.UUID) (map[uuid.UUID]StandingsEntry, error) {
	return nil, nil
}

func (f *fakeStandingsStore) Close() error { return nil }

func TestStandingsRecorderRoutesEventTypes(t *testing.T) {
	runID := uuid.New()
	agentID := uuid.New()
	inner := &fakeRecorder{
		returnEvent: repository.RunEvent{RunID: runID, RunAgentID: agentID, SequenceNumber: 1},
	}
	store := &fakeStandingsStore{}
	recorder := NewStandingsRecorder(inner, store, slog.Default())

	modelPayload, _ := json.Marshal(map[string]any{
		"provider_model_id": "claude-sonnet-4-6",
		"usage":             map[string]int64{"total_tokens": 250},
	})

	cases := []struct {
		name        string
		eventType   runevents.Type
		stepIndex   int
		payload     json.RawMessage
		wantStore   bool
		wantState   StandingsState
		wantStep    int
		wantTokens  int64
		wantTool    int
		wantModel   string
	}{
		{
			name:      "run.started → state running",
			eventType: runevents.EventTypeSystemRunStarted,
			wantStore: true,
			wantState: StandingsStateRunning,
		},
		{
			name:      "step.started → step update",
			eventType: runevents.EventTypeSystemStepStarted,
			stepIndex: 5,
			wantStore: true,
			wantStep:  5,
			wantState: StandingsStateRunning,
		},
		{
			name:      "tool.call.completed → tool tick",
			eventType: runevents.EventTypeToolCallCompleted,
			wantStore: true,
			wantTool:  1,
		},
		{
			name:       "model.call.completed → tokens + model",
			eventType:  runevents.EventTypeModelCallCompleted,
			payload:    modelPayload,
			wantStore:  true,
			wantTokens: 250,
			wantModel:  "claude-sonnet-4-6",
		},
		{
			name:      "output.finalized → submitted",
			eventType: runevents.EventTypeSystemOutputFinalized,
			wantStore: true,
			wantState: StandingsStateSubmitted,
		},
		{
			name:      "run.failed → failed",
			eventType: runevents.EventTypeSystemRunFailed,
			wantStore: true,
			wantState: StandingsStateFailed,
		},
		{
			name:      "unrelated event → no store call",
			eventType: runevents.EventTypeModelOutputDelta,
			wantStore: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store.mu.Lock()
			store.calls = nil
			store.mu.Unlock()

			env := runevents.Envelope{
				RunID:      runID,
				RunAgentID: agentID,
				EventType:  tc.eventType,
				Summary:    runevents.SummaryMetadata{StepIndex: tc.stepIndex},
				Payload:    tc.payload,
			}
			_, err := recorder.RecordRunEvent(context.Background(), repository.RecordRunEventParams{Event: env})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			store.mu.Lock()
			defer store.mu.Unlock()
			if !tc.wantStore {
				if len(store.calls) != 0 {
					t.Fatalf("store should not be called, got %d calls", len(store.calls))
				}
				return
			}
			if len(store.calls) != 1 {
				t.Fatalf("expected 1 store call, got %d", len(store.calls))
			}
			call := store.calls[0]
			if call.runID != runID {
				t.Errorf("store runID = %s, want %s", call.runID, runID)
			}
			if call.updates.RunAgentID != agentID {
				t.Errorf("store runAgentID = %s, want %s", call.updates.RunAgentID, agentID)
			}
			if tc.wantState != "" && call.updates.State != tc.wantState {
				t.Errorf("state = %q, want %q", call.updates.State, tc.wantState)
			}
			if call.updates.Step != tc.wantStep {
				t.Errorf("step = %d, want %d", call.updates.Step, tc.wantStep)
			}
			if call.updates.TokensUsed != tc.wantTokens {
				t.Errorf("tokens = %d, want %d", call.updates.TokensUsed, tc.wantTokens)
			}
			if call.updates.ToolCalls != tc.wantTool {
				t.Errorf("tool_calls = %d, want %d", call.updates.ToolCalls, tc.wantTool)
			}
			if call.updates.Model != tc.wantModel {
				t.Errorf("model = %q, want %q", call.updates.Model, tc.wantModel)
			}
		})
	}
}

func TestStandingsRecorderSwallowsStoreError(t *testing.T) {
	inner := &fakeRecorder{
		returnEvent: repository.RunEvent{SequenceNumber: 7},
	}
	store := &fakeStandingsStore{nextErr: errors.New("redis down")}
	recorder := NewStandingsRecorder(inner, store, slog.Default())

	env := runevents.Envelope{
		RunID:      uuid.New(),
		RunAgentID: uuid.New(),
		EventType:  runevents.EventTypeSystemRunStarted,
	}
	event, err := recorder.RecordRunEvent(context.Background(), repository.RecordRunEventParams{Event: env})
	if err != nil {
		t.Fatalf("store error must be swallowed, got: %v", err)
	}
	if event.SequenceNumber != 7 {
		t.Fatalf("sequence = %d, want 7", event.SequenceNumber)
	}
}

func TestStandingsRecorderSkipsWhenPersistFails(t *testing.T) {
	inner := &fakeRecorder{returnErr: errors.New("db failed")}
	store := &fakeStandingsStore{}
	recorder := NewStandingsRecorder(inner, store, slog.Default())

	env := runevents.Envelope{
		RunID:      uuid.New(),
		RunAgentID: uuid.New(),
		EventType:  runevents.EventTypeSystemRunStarted,
	}
	_, err := recorder.RecordRunEvent(context.Background(), repository.RecordRunEventParams{Event: env})
	if err == nil {
		t.Fatal("expected persist error to propagate")
	}
	if len(store.calls) != 0 {
		t.Fatalf("store should not be called when persist fails, got %d calls", len(store.calls))
	}
}

func TestMergeEntryIsAdditive(t *testing.T) {
	agentID := uuid.New()
	existing := StandingsEntry{
		RunAgentID: agentID,
		Step:       3,
		ToolCalls:  2,
		TokensUsed: 100,
		State:      StandingsStateRunning,
		Model:      "gpt-5",
	}
	update := StandingsEntry{
		RunAgentID: agentID,
		Step:       2, // lower than existing; should NOT regress
		ToolCalls:  1,
		TokensUsed: 50,
		Model:      "", // empty; should NOT clobber model name
	}
	merged := mergeEntry(existing, update)

	if merged.Step != 3 {
		t.Errorf("step regressed: got %d, want 3 (max preserved)", merged.Step)
	}
	if merged.ToolCalls != 3 {
		t.Errorf("tool_calls = %d, want 3 (additive)", merged.ToolCalls)
	}
	if merged.TokensUsed != 150 {
		t.Errorf("tokens = %d, want 150 (additive)", merged.TokensUsed)
	}
	if merged.Model != "gpt-5" {
		t.Errorf("model clobbered to %q, want preserved gpt-5", merged.Model)
	}
	if merged.State != StandingsStateRunning {
		t.Errorf("state = %q, want running (empty update didn't clobber)", merged.State)
	}
}

func TestStandingsHashKeyAndField(t *testing.T) {
	runID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	agentID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if key := StandingsHashKey(runID); key != "run:12345678-1234-1234-1234-123456789abc:standings" {
		t.Errorf("StandingsHashKey = %q", key)
	}
	if field := StandingsField(agentID); field != "agent:aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("StandingsField = %q", field)
	}
}

func TestNoopStandingsStoreIsInert(t *testing.T) {
	store := NoopStandingsStore{}
	if err := store.Update(context.Background(), uuid.New(), StandingsEntry{}); err != nil {
		t.Errorf("noop Update returned error: %v", err)
	}
	snap, err := store.Snapshot(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("noop Snapshot returned error: %v", err)
	}
	if len(snap) != 0 {
		t.Errorf("noop Snapshot returned %d entries, want 0", len(snap))
	}
	if err := store.Close(); err != nil {
		t.Errorf("noop Close returned error: %v", err)
	}
}
