package repository

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/runevents"
)

type ReasoningRunExecution struct {
	ID                      uuid.UUID
	RunID                   uuid.UUID
	RunAgentID              uuid.UUID
	ReasoningRunID          *string
	EndpointURL             string
	Status                  string
	SandboxMetadata         json.RawMessage
	PendingProposalEventID  *string
	PendingProposalPayload  json.RawMessage
	LastEventType           *string
	LastEventPayload        json.RawMessage
	ResultPayload           json.RawMessage
	ErrorMessage            *string
	DeadlineAt              time.Time
	AcceptedAt              *time.Time
	StartedAt               *time.Time
	FinishedAt              *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type CreateReasoningRunExecutionParams struct {
	RunID       uuid.UUID
	RunAgentID  uuid.UUID
	EndpointURL string
	DeadlineAt  time.Time
}

type MarkReasoningRunExecutionTimedOutParams struct {
	RunAgentID   uuid.UUID
	ErrorMessage string
}

type ApplyReasoningRunEventParams struct {
	RunAgentID             uuid.UUID
	Status                 string
	LastEventType          string
	LastEventPayload       json.RawMessage
	PendingProposalEventID *string
	PendingProposalPayload json.RawMessage
}

type RecordReasoningRunEventParams struct {
	Event runevents.Envelope
}

