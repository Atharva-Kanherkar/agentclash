package workflow

import (
	"context"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/reasoning"
	"github.com/google/uuid"
)

// ReasoningCallbackTokenSigner generates and verifies HMAC tokens for
// reasoning run callbacks. Follows the same pattern as hostedruns.CallbackTokenSigner.
type ReasoningCallbackTokenSigner interface {
	Sign(runID uuid.UUID, runAgentID uuid.UUID) (string, error)
	Verify(token string) (ReasoningCallbackClaims, error)
}

type ReasoningCallbackClaims struct {
	RunID      uuid.UUID
	RunAgentID uuid.UUID
}

type StartReasoningRunInput struct {
	RunAgentID uuid.UUID `json:"run_agent_id"`
}

type StartReasoningRunResult struct {
	ReasoningRunID string `json:"reasoning_run_id"`
}

type ExecuteReasoningToolBatchInput struct {
	RunAgentID uuid.UUID `json:"run_agent_id"`
}

type ExecuteReasoningToolBatchResult struct {
	Results []reasoning.ToolResult `json:"results"`
}

type SubmitReasoningToolResultsInput struct {
	RunAgentID     uuid.UUID              `json:"run_agent_id"`
	ReasoningRunID string                 `json:"reasoning_run_id"`
	ToolResults    []reasoning.ToolResult `json:"tool_results"`
}

type CancelReasoningRunInput struct {
	RunAgentID     uuid.UUID `json:"run_agent_id"`
	ReasoningRunID string    `json:"reasoning_run_id"`
	Reason         string    `json:"reason"`
}

type MarkReasoningRunTimedOutInput struct {
	RunAgentID   uuid.UUID `json:"run_agent_id"`
	ErrorMessage string    `json:"error_message"`
}

// ReasoningRunStarter handles HTTP communication with the Python reasoning service.
type ReasoningRunStarter interface {
	Start(ctx context.Context, request reasoning.StartRequest) (reasoning.StartResponse, error)
	SubmitToolResults(ctx context.Context, reasoningRunID string, batch reasoning.ToolResultsBatch) error
	Cancel(ctx context.Context, reasoningRunID string, request reasoning.CancelRequest) error
}
