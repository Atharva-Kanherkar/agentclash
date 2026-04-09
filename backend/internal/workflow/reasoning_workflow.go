package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/domain"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/reasoning"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	"go.temporal.io/sdk/temporal"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

const (
	reasoningActivityCleanupBuffer = 15 * time.Second

	startReasoningRunActivityName       = "workflow.start_reasoning_run"
	executeReasoningToolBatchName        = "workflow.execute_reasoning_tool_batch"
	submitReasoningToolResultsName       = "workflow.submit_reasoning_tool_results"
	cancelReasoningRunActivityName       = "workflow.cancel_reasoning_run"
	markReasoningRunTimedOutActivityName = "workflow.mark_reasoning_run_timed_out"
)

func runReasoningRunAgent(ctx sdkworkflow.Context, input RunAgentWorkflowInput, executionContext repository.RunAgentExecutionContext) error {
	if executionContext.Deployment.RuntimeProfile.TraceMode == "disabled" {
		return errors.New("reasoning lane requires trace_mode 'required' or 'preferred'; got 'disabled'")
	}

	reason := "reasoning run starting"
	if err := transitionRunAgentStatus(ctx, input.RunAgentID, domain.RunAgentStatusExecuting, &reason, nil); err != nil {
		return err
	}

	var startResult StartReasoningRunResult
	if err := sdkworkflow.ExecuteActivity(
		sdkworkflow.WithActivityOptions(ctx, sdkworkflow.ActivityOptions{
			StartToCloseTimeout: 10 * time.Second,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
		}),
		startReasoningRunActivityName,
		StartReasoningRunInput{RunAgentID: input.RunAgentID},
	).Get(ctx, &startResult); err != nil {
		return err
	}

	statusReason := fmt.Sprintf("reasoning run accepted as %s", startResult.ReasoningRunID)
	if err := transitionRunAgentStatus(ctx, input.RunAgentID, domain.RunAgentStatusExecuting, &statusReason, nil); err != nil {
		return err
	}

	if err := waitForReasoningTerminalEvent(ctx, input, executionContext, startResult.ReasoningRunID); err != nil {
		return err
	}

	if err := transitionRunAgentStatus(ctx, input.RunAgentID, domain.RunAgentStatusEvaluating, stringPtr("reasoning execution completed; parent scoring pending"), nil); err != nil {
		return err
	}
	warnOnReplayBuildFailure(ctx, input.RunAgentID, "successful reasoning execution")
	return nil
}

func waitForReasoningTerminalEvent(ctx sdkworkflow.Context, input RunAgentWorkflowInput, executionContext repository.RunAgentExecutionContext, reasoningRunID string) error {
	timeout := time.Duration(executionContext.Deployment.RuntimeProfile.RunTimeoutSeconds) * time.Second
	signalCh := sdkworkflow.GetSignalChannel(ctx, reasoning.ReasoningRunEventSignal)
	timer := sdkworkflow.NewTimer(ctx, timeout)

	for {
		var (
			signal   reasoning.ReasoningEventSignal
			timedOut bool
			gotEvent bool
		)

		selector := sdkworkflow.NewSelector(ctx)
		selector.AddReceive(signalCh, func(c sdkworkflow.ReceiveChannel, more bool) {
			gotEvent = true
			c.Receive(ctx, &signal)
		})
		selector.AddFuture(timer, func(sdkworkflow.Future) {
			timedOut = true
		})
		selector.Select(ctx)

		if timedOut {
			reason := fmt.Sprintf("reasoning run timed out waiting for callback after %s", timeout)
			_ = sdkworkflow.ExecuteActivity(
				sdkworkflow.WithActivityOptions(ctx, sdkworkflow.ActivityOptions{
					StartToCloseTimeout: 10 * time.Second,
					RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 2},
				}),
				cancelReasoningRunActivityName,
				CancelReasoningRunInput{
					RunAgentID:     input.RunAgentID,
					ReasoningRunID: reasoningRunID,
					Reason:         reason,
				},
			).Get(ctx, nil)
			_ = sdkworkflow.ExecuteActivity(ctx, markReasoningRunTimedOutActivityName, MarkReasoningRunTimedOutInput{
				RunAgentID:   input.RunAgentID,
				ErrorMessage: reason,
			}).Get(ctx, nil)
			return errors.New(reason)
		}
		if !gotEvent {
			continue
		}

		switch signal.EventType {
		case "model.tool_calls.proposed":
			if err := handleReasoningToolProposal(ctx, input, executionContext, reasoningRunID); err != nil {
				return err
			}
			continue

		case "system.run.completed":
			return nil

		case "system.run.failed":
			return fmt.Errorf("reasoning run failed (event_id: %s)", signal.EventID)

		default:
			// Non-actionable event: ignore and continue waiting.
			continue
		}
	}
}

func handleReasoningToolProposal(ctx sdkworkflow.Context, input RunAgentWorkflowInput, executionContext repository.RunAgentExecutionContext, reasoningRunID string) error {
	toolTimeout := time.Duration(executionContext.Deployment.RuntimeProfile.StepTimeoutSeconds)*time.Second + reasoningActivityCleanupBuffer
	toolCtx := sdkworkflow.WithActivityOptions(ctx, sdkworkflow.ActivityOptions{
		StartToCloseTimeout: toolTimeout,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})

	var toolResult ExecuteReasoningToolBatchResult
	if err := sdkworkflow.ExecuteActivity(toolCtx, executeReasoningToolBatchName, ExecuteReasoningToolBatchInput{
		RunAgentID: input.RunAgentID,
	}).Get(ctx, &toolResult); err != nil {
		return err
	}

	submitCtx := sdkworkflow.WithActivityOptions(ctx, sdkworkflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})
	if err := sdkworkflow.ExecuteActivity(submitCtx, submitReasoningToolResultsName, SubmitReasoningToolResultsInput{
		RunAgentID:     input.RunAgentID,
		ReasoningRunID: reasoningRunID,
		ToolResults:    toolResult.Results,
	}).Get(ctx, nil); err != nil {
		return err
	}

	return nil
}
