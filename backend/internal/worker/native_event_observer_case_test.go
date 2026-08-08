package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/engine"
	"github.com/agentclash/agentclash/backend/internal/repository"
	"github.com/agentclash/agentclash/runtime/domain"
	"github.com/agentclash/agentclash/runtime/runevents"
	"github.com/google/uuid"
)

type capturingRecorder struct {
	events []runevents.Envelope
}

func (c *capturingRecorder) RecordRunEvent(_ context.Context, params repository.RecordRunEventParams) (repository.RunEvent, error) {
	c.events = append(c.events, params.Event)
	return repository.RunEvent{
		RunID:      params.Event.RunID,
		RunAgentID: params.Event.RunAgentID,
		EventType:  params.Event.EventType,
		Payload:    params.Event.Payload,
	}, nil
}

func TestNativeObserverEmbedsCaseKey(t *testing.T) {
	runID := uuid.New()
	runAgentID := uuid.New()
	recorder := &capturingRecorder{}
	observer := &NativeRunEventObserver{
		recorder: recorder,
		executionContext: repository.RunAgentExecutionContext{
			Run:      domain.Run{ID: runID},
			RunAgent: domain.RunAgent{ID: runAgentID, RunID: runID},
			ChallengeInputSet: &repository.ChallengeInputSetExecutionContext{
				Cases: []repository.ChallengeCaseExecutionContext{
					{CaseKey: "refund-1", ItemKey: "refund-1"},
				},
			},
		},
	}

	if err := observer.OnRunComplete(context.Background(), engine.Result{
		FinalOutput: "done",
		StopReason:  engine.StopReasonCompleted,
	}); err != nil {
		t.Fatalf("OnRunComplete: %v", err)
	}
	if len(recorder.events) < 2 {
		// run.started + run.completed
		t.Fatalf("events = %d, want ≥2", len(recorder.events))
	}
	for _, event := range recorder.events {
		if event.Summary.CaseKey != "refund-1" {
			t.Fatalf("summary.case_key = %q, want refund-1 (type=%s)", event.Summary.CaseKey, event.EventType)
		}
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("payload: %v", err)
		}
		if payload["case_key"] != "refund-1" {
			t.Fatalf("payload.case_key = %v, want refund-1 (type=%s)", payload["case_key"], event.EventType)
		}
	}
}

func TestNativeObserverOmitsCaseKeyForMultiCaseContext(t *testing.T) {
	recorder := &capturingRecorder{}
	observer := &NativeRunEventObserver{
		recorder: recorder,
		executionContext: repository.RunAgentExecutionContext{
			Run:      domain.Run{ID: uuid.New()},
			RunAgent: domain.RunAgent{ID: uuid.New()},
			ChallengeInputSet: &repository.ChallengeInputSetExecutionContext{
				Cases: []repository.ChallengeCaseExecutionContext{
					{CaseKey: "a"},
					{CaseKey: "b"},
				},
			},
		},
	}
	if err := observer.OnRunComplete(context.Background(), engine.Result{
		FinalOutput: "done",
		StopReason:  engine.StopReasonCompleted,
	}); err != nil {
		t.Fatalf("OnRunComplete: %v", err)
	}
	for _, event := range recorder.events {
		if event.Summary.CaseKey != "" {
			t.Fatalf("expected empty case_key for multi-case mega-activity, got %q", event.Summary.CaseKey)
		}
		var payload map[string]any
		_ = json.Unmarshal(event.Payload, &payload)
		if _, ok := payload["case_key"]; ok {
			t.Fatalf("payload should not include case_key for multi-case context")
		}
	}
}
