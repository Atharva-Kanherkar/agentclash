package api

import (
	"context"
	"fmt"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/hostedruns"
	"github.com/Atharva-Kanherkar/agentclash/backend/internal/workflow"
	"github.com/google/uuid"
	temporalsdk "go.temporal.io/sdk/client"
)

type TemporalClient interface {
	ExecuteWorkflow(ctx context.Context, options temporalsdk.StartWorkflowOptions, workflow interface{}, args ...interface{}) (temporalsdk.WorkflowRun, error)
	SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error
}

type TemporalRunWorkflowStarter struct {
	client TemporalClient
}

func NewTemporalRunWorkflowStarter(client TemporalClient) TemporalRunWorkflowStarter {
	return TemporalRunWorkflowStarter{client: client}
}

func (s TemporalRunWorkflowStarter) StartRunWorkflow(ctx context.Context, runID uuid.UUID) error {
	workflowID := fmt.Sprintf("%s/%s", workflow.RunWorkflowName, runID)
	_, err := s.client.ExecuteWorkflow(ctx, temporalsdk.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflow.RunWorkflowName,
	}, workflow.RunWorkflowName, workflow.RunWorkflowInput{
		RunID: runID,
	})
	return err
}

type TemporalHostedRunWorkflowSignaler struct {
	client TemporalClient
}

func NewTemporalHostedRunWorkflowSignaler(client TemporalClient) TemporalHostedRunWorkflowSignaler {
	return TemporalHostedRunWorkflowSignaler{client: client}
}

func (s TemporalHostedRunWorkflowSignaler) SignalRunAgentWorkflow(ctx context.Context, runID uuid.UUID, runAgentID uuid.UUID, event hostedruns.Event) error {
	workflowID := fmt.Sprintf("%s/%s/%s", workflow.RunAgentWorkflowName, runID, runAgentID)
	return s.client.SignalWorkflow(ctx, workflowID, "", workflow.HostedRunEventSignal, event)
}
