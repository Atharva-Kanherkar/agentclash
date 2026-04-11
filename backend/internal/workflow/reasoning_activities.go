package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/provider"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/reasoning"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
)

func (a *Activities) StartReasoningRun(ctx context.Context, input StartReasoningRunInput) (StartReasoningRunResult, error) {
	if a.hooks.ReasoningRunStarter == nil {
		return StartReasoningRunResult{}, fmt.Errorf("reasoning run starter is not configured")
	}

	executionContext, err := a.repo.GetRunAgentExecutionContextByID(ctx, input.RunAgentID)
	if err != nil {
		return StartReasoningRunResult{}, wrapActivityError(err)
	}

	tools := buildReasoningToolset(executionContext)
	executionContextJSON, err := json.Marshal(executionContext)
	if err != nil {
		return StartReasoningRunResult{}, fmt.Errorf("marshal execution context: %w", err)
	}

	deadlineAt := time.Now().UTC().Add(time.Duration(executionContext.Deployment.RuntimeProfile.RunTimeoutSeconds) * time.Second)
	idempotencyKey := fmt.Sprintf("start:%s", input.RunAgentID)

	callbackURL := a.hooks.ReasoningCallbackBaseURL + "/v1/integrations/reasoning-runs/" + executionContext.Run.ID.String() + "/events"
	callbackToken, err := a.hooks.ReasoningCallbackTokenSigner.Sign(executionContext.Run.ID, input.RunAgentID)
	if err != nil {
		return StartReasoningRunResult{}, fmt.Errorf("sign callback token: %w", err)
	}

	// Create the execution tracking record.
	if err := a.repo.CreateReasoningRunExecution(ctx, repository.CreateReasoningRunExecutionParams{
		RunID:       executionContext.Run.ID,
		RunAgentID:  input.RunAgentID,
		EndpointURL: a.hooks.ReasoningServiceURL,
		DeadlineAt:  deadlineAt,
	}); err != nil {
		return StartReasoningRunResult{}, wrapActivityError(err)
	}

	response, err := a.hooks.ReasoningRunStarter.Start(ctx, reasoning.StartRequest{
		RunID:            executionContext.Run.ID,
		RunAgentID:       input.RunAgentID,
		IdempotencyKey:   idempotencyKey,
		ExecutionContext:  executionContextJSON,
		Tools:            tools,
		CallbackURL:      callbackURL,
		CallbackToken:    callbackToken,
		DeadlineAt:       deadlineAt,
	})
	if err != nil {
		return StartReasoningRunResult{}, fmt.Errorf("start reasoning run: %w", err)
	}
	if !response.Accepted {
		return StartReasoningRunResult{}, fmt.Errorf("reasoning service rejected run: %s", response.Error)
	}
	if response.ReasoningRunID == "" {
		return StartReasoningRunResult{}, fmt.Errorf("reasoning service returned empty reasoning_run_id")
	}

	return StartReasoningRunResult{
		ReasoningRunID: response.ReasoningRunID,
	}, nil
}

func (a *Activities) ExecuteReasoningToolBatch(ctx context.Context, input ExecuteReasoningToolBatchInput) (ExecuteReasoningToolBatchResult, error) {
	execution, err := a.repo.GetReasoningRunExecutionByRunAgentID(ctx, input.RunAgentID)
	if err != nil {
		return ExecuteReasoningToolBatchResult{}, wrapActivityError(err)
	}
	if execution.PendingProposalPayload == nil {
		return ExecuteReasoningToolBatchResult{}, fmt.Errorf("no pending tool proposal for run_agent %s", input.RunAgentID)
	}

	var proposedCalls []proposedToolCall
	if err := json.Unmarshal(execution.PendingProposalPayload, &proposedCalls); err != nil {
		return ExecuteReasoningToolBatchResult{}, fmt.Errorf("unmarshal pending proposal: %w", err)
	}

	executionContext, err := a.repo.GetRunAgentExecutionContextByID(ctx, input.RunAgentID)
	if err != nil {
		return ExecuteReasoningToolBatchResult{}, wrapActivityError(err)
	}

	// Build sandbox session: create on first call, reconnect on subsequent.
	session, err := a.getOrCreateReasoningSandbox(ctx, execution, executionContext)
	if err != nil {
		// Sandbox failure: mark all tools as failed.
		results := make([]reasoning.ToolResult, len(proposedCalls))
		for i, call := range proposedCalls {
			results[i] = reasoning.ToolResult{
				ToolCallID:   call.ID,
				Status:       reasoning.ToolResultStatusFailed,
				ErrorMessage: fmt.Sprintf("sandbox unavailable: %v", err),
			}
		}
		return ExecuteReasoningToolBatchResult{Results: results}, nil
	}

	results := executeToolBatch(ctx, session, executionContext, proposedCalls)
	return ExecuteReasoningToolBatchResult{Results: results}, nil
}

func (a *Activities) SubmitReasoningToolResults(ctx context.Context, input SubmitReasoningToolResultsInput) error {
	if a.hooks.ReasoningRunStarter == nil {
		return fmt.Errorf("reasoning run starter is not configured")
	}

	idempotencyKey := fmt.Sprintf("tools:%s:%s", input.RunAgentID, input.ReasoningRunID)
	return a.hooks.ReasoningRunStarter.SubmitToolResults(ctx, input.ReasoningRunID, reasoning.ToolResultsBatch{
		IdempotencyKey: idempotencyKey,
		ToolResults:    input.ToolResults,
	})
}

func (a *Activities) CancelReasoningRun(ctx context.Context, input CancelReasoningRunInput) error {
	if a.hooks.ReasoningRunStarter == nil {
		return nil
	}

	return a.hooks.ReasoningRunStarter.Cancel(ctx, input.ReasoningRunID, reasoning.CancelRequest{
		IdempotencyKey: fmt.Sprintf("cancel:%s", input.RunAgentID),
		Reason:         input.Reason,
	})
}

func (a *Activities) MarkReasoningRunTimedOut(ctx context.Context, input MarkReasoningRunTimedOutInput) error {
	return a.repo.MarkReasoningRunExecutionTimedOut(ctx, repository.MarkReasoningRunExecutionTimedOutParams{
		RunAgentID:   input.RunAgentID,
		ErrorMessage: input.ErrorMessage,
	})
}

// getOrCreateReasoningSandbox creates a new sandbox on first tool proposal
// or reconnects to the existing one using persisted metadata.
func (a *Activities) getOrCreateReasoningSandbox(ctx context.Context, execution repository.ReasoningRunExecution, executionContext repository.RunAgentExecutionContext) (interface{ ID() string }, error) {
	// This is a placeholder. Full sandbox lifecycle management will use
	// the sandbox.Provider interface with Create/Reconnect based on
	// execution.SandboxMetadata being nil or populated.
	//
	// For now, return an error indicating sandbox support requires the
	// NativeModelInvoker's sandbox provider to be wired in.
	return nil, fmt.Errorf("sandbox lifecycle not yet wired for reasoning lane")
}

// proposedToolCall represents a single tool call from a model.tool_calls.proposed event.
type proposedToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// executeToolBatch runs each proposed tool call against the sandbox and
// returns results. This is a placeholder that will be wired to the actual
// sandbox execution logic (mirroring native_executor.go executeSandboxTool).
func executeToolBatch(ctx context.Context, session interface{ ID() string }, executionContext repository.RunAgentExecutionContext, calls []proposedToolCall) []reasoning.ToolResult {
	results := make([]reasoning.ToolResult, len(calls))
	for i, call := range calls {
		results[i] = reasoning.ToolResult{
			ToolCallID:   call.ID,
			Status:       reasoning.ToolResultStatusFailed,
			ErrorMessage: "tool execution not yet implemented",
		}
	}
	return results
}

// buildReasoningToolset builds the tool definitions that will be sent to
// the Python reasoning service. It mirrors the native buildToolset() logic
// but excludes the submit tool (the reasoning lane uses system.output.finalized
// instead of a submit tool call).
func buildReasoningToolset(executionContext repository.RunAgentExecutionContext) []provider.ToolDefinition {
	// This will be implemented to mirror engine.buildToolset() minus submit.
	// For now return nil; tools are resolved at StartReasoningRun time.
	return nil
}
