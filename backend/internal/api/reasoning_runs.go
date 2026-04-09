package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/hostedruns"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/reasoning"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/runevents"
)

// ReasoningRunIngestionService handles incoming canonical events from the
// Python reasoning service.
type ReasoningRunIngestionService interface {
	IngestEvent(ctx context.Context, runID uuid.UUID, token string, event runevents.Envelope) error
}

// ReasoningRunWorkflowSignaler signals the reasoning workflow when an
// actionable event arrives.
type ReasoningRunWorkflowSignaler interface {
	SignalReasoningRunWorkflow(ctx context.Context, runID uuid.UUID, runAgentID uuid.UUID, signal reasoning.ReasoningEventSignal) error
}

// ReasoningRunExecutionRepository provides access to reasoning_run_executions.
type ReasoningRunExecutionRepository interface {
	GetReasoningRunExecutionByRunAgentID(ctx context.Context, runAgentID uuid.UUID) (repository.ReasoningRunExecution, error)
	ApplyReasoningRunEvent(ctx context.Context, params repository.ApplyReasoningRunEventParams) error
	RecordReasoningRunEvent(ctx context.Context, params repository.RecordReasoningRunEventParams) error
}

// ReasoningRunIngestionManager processes incoming reasoning events.
type ReasoningRunIngestionManager struct {
	repo     ReasoningRunExecutionRepository
	signer   hostedruns.CallbackTokenSigner
	signaler ReasoningRunWorkflowSignaler
}

type noopReasoningRunIngestionService struct{}

func NewReasoningRunIngestionManager(repo ReasoningRunExecutionRepository, secret string, signaler ReasoningRunWorkflowSignaler) *ReasoningRunIngestionManager {
	return &ReasoningRunIngestionManager{
		repo:     repo,
		signer:   hostedruns.NewCallbackTokenSigner(secret),
		signaler: signaler,
	}
}

func (noopReasoningRunIngestionService) IngestEvent(context.Context, uuid.UUID, string, runevents.Envelope) error {
	return errors.New("reasoning run ingestion is not configured")
}

func (m *ReasoningRunIngestionManager) IngestEvent(ctx context.Context, runID uuid.UUID, token string, event runevents.Envelope) error {
	// 1. Verify callback token.
	claims, err := m.signer.Verify(token)
	if err != nil {
		return err
	}
	if claims.RunID != runID {
		return errors.New("callback run_id does not match token")
	}
	if claims.RunAgentID != event.RunAgentID {
		return errors.New("callback run_agent_id does not match token")
	}

	// 2. Validate the event envelope.
	if event.Source != runevents.SourceReasoningEngine {
		return errors.Join(reasoning.ErrInvalidReasoningEvent, errors.New("source must be reasoning_engine"))
	}
	if err := event.ValidatePending(); err != nil {
		return err
	}

	// 3. Load execution record.
	execution, err := m.repo.GetReasoningRunExecutionByRunAgentID(ctx, event.RunAgentID)
	if err != nil {
		return err
	}
	if execution.RunID != runID {
		return errors.New("callback run_id does not match reasoning execution")
	}

	// 4. Reject post-terminal events.
	if isReasoningTerminalStatus(execution.Status) {
		if execution.LastEventType != nil && *execution.LastEventType == string(event.EventType) {
			// Idempotent: same terminal event already recorded.
			return nil
		}
		return errors.Join(reasoning.ErrInvalidReasoningEvent, errors.New("execution already in terminal state"))
	}

	// 5. State machine validation: first event must be system.run.started.
	if execution.LastEventType == nil && event.EventType != runevents.EventTypeSystemRunStarted {
		return errors.Join(reasoning.ErrInvalidReasoningEvent, errors.New("first event must be system.run.started"))
	}

	// 6. Reject new proposals while one is pending.
	if event.EventType == runevents.EventTypeModelToolCallsProposed && execution.PendingProposalEventID != nil {
		return errors.Join(reasoning.ErrInvalidReasoningEvent, errors.New("proposal already pending"))
	}

	// 7. Determine new execution state.
	status := reasoningStatusForEvent(event)
	var pendingProposalEventID *string
	var pendingProposalPayload json.RawMessage
	if event.EventType == runevents.EventTypeModelToolCallsProposed {
		pendingProposalEventID = &event.EventID
		pendingProposalPayload = extractToolCallsFromPayload(event.Payload)
	}

	eventPayload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// 8. Apply state update to reasoning_run_executions.
	if err := m.repo.ApplyReasoningRunEvent(ctx, repository.ApplyReasoningRunEventParams{
		RunAgentID:             event.RunAgentID,
		Status:                 status,
		LastEventType:          string(event.EventType),
		LastEventPayload:       eventPayload,
		PendingProposalEventID: pendingProposalEventID,
		PendingProposalPayload: pendingProposalPayload,
	}); err != nil {
		return err
	}

	// 9. Persist event to run_events (transactional with replay upsert).
	if err := m.repo.RecordReasoningRunEvent(ctx, repository.RecordReasoningRunEventParams{
		Event: event,
	}); err != nil {
		return err
	}

	// 10. Signal workflow for actionable events.
	if isReasoningActionableEvent(event) {
		return m.signaler.SignalReasoningRunWorkflow(ctx, runID, event.RunAgentID, reasoning.ReasoningEventSignal{
			EventID:   event.EventID,
			EventType: string(event.EventType),
		})
	}
	return nil
}

func registerReasoningIntegrationRoutes(router chi.Router, logger *slog.Logger, service ReasoningRunIngestionService) {
	router.Route("/v1/integrations/reasoning-runs", func(r chi.Router) {
		r.Post("/{runID}/events", ingestReasoningRunEventHandler(logger, service))
	})
}

func ingestReasoningRunEventHandler(logger *slog.Logger, service ReasoningRunIngestionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := requireJSONContentType(r); err != nil {
			writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", err.Error())
			return
		}

		runID, err := runIDFromURLParam("runID")(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_run_id", err.Error())
			return
		}

		token, err := hostedruns.BearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_callback_token", "authorization bearer token is required")
			return
		}

		var event runevents.Envelope
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&event); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
			return
		}

		if err := service.IngestEvent(r.Context(), runID, token, event); err != nil {
			switch {
			case errors.Is(err, hostedruns.ErrInvalidCallbackToken):
				writeError(w, http.StatusUnauthorized, "invalid_callback_token", "callback token is invalid")
			case errors.Is(err, reasoning.ErrReasoningRunNotFound):
				writeError(w, http.StatusNotFound, "reasoning_run_not_found", "reasoning run execution not found")
			case errors.Is(err, reasoning.ErrInvalidReasoningEvent):
				writeError(w, http.StatusUnprocessableEntity, "invalid_reasoning_event", err.Error())
			default:
				logger.Error("ingest reasoning run event failed",
					"method", r.Method,
					"path", r.URL.Path,
					"run_id", runID,
					"run_agent_id", event.RunAgentID,
					"error", err,
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
	}
}

func isReasoningTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "timed_out", "cancelled":
		return true
	default:
		return false
	}
}

func reasoningStatusForEvent(event runevents.Envelope) string {
	switch event.EventType {
	case runevents.EventTypeSystemRunStarted:
		return "running"
	case runevents.EventTypeSystemRunCompleted:
		return "completed"
	case runevents.EventTypeSystemRunFailed:
		return "failed"
	default:
		return "running"
	}
}

func isReasoningActionableEvent(event runevents.Envelope) bool {
	switch event.EventType {
	case runevents.EventTypeModelToolCallsProposed,
		runevents.EventTypeSystemRunCompleted,
		runevents.EventTypeSystemRunFailed:
		return true
	default:
		return false
	}
}

func extractToolCallsFromPayload(payload json.RawMessage) json.RawMessage {
	var p struct {
		ToolCalls json.RawMessage `json:"tool_calls"`
	}
	if err := json.Unmarshal(payload, &p); err != nil || len(p.ToolCalls) == 0 {
		return payload
	}
	return p.ToolCalls
}
