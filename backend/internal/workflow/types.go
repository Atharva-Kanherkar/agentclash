package workflow

import "github.com/google/uuid"

const (
	EvalSessionWorkflowName                = "EvalSessionWorkflow"
	RunWorkflowName                        = "RunWorkflow"
	RunAgentWorkflowName                   = "RunAgentWorkflow"
	AgentHarnessExecutionWorkflowName      = "AgentHarnessExecutionWorkflow"
	PublicAgentTryoutExecutionWorkflowName = "PublicAgentTryoutExecutionWorkflow"
	SyntheticDatasetGenerationWorkflowName = "SyntheticDatasetGenerationWorkflow"
	HostedRunEventSignal                   = "hosted_run_event"
	// WorkflowTaskQueue is where parent/child workflows are started. Fleet 2
	// partitions activity work across TaskQueueExecution/Scoring/Background;
	// workflows themselves run on the execution queue.
	WorkflowTaskQueue = TaskQueueExecution
)

type EvalSessionWorkflowInput struct {
	EvalSessionID uuid.UUID `json:"eval_session_id"`
	// MaxConcurrentRuns caps in-flight child RunWorkflows. Zero → DefaultMaxConcurrentEvalSessionRuns.
	MaxConcurrentRuns int `json:"max_concurrent_runs,omitempty"`
	// EvalSetID, when set, enables per-run budget gating for this session's
	// child runs (Fleet 11). Standalone sessions leave this empty.
	EvalSetID uuid.UUID `json:"eval_set_id,omitempty"`
}

type RunWorkflowInput struct {
	RunID uuid.UUID `json:"run_id"`
	// MaxConcurrentRunAgents caps in-flight child RunAgentWorkflows. Zero → DefaultMaxConcurrentRunAgents.
	MaxConcurrentRunAgents int `json:"max_concurrent_run_agents,omitempty"`
	// MaxConcurrentScoreActivities caps in-flight score activities. Zero → DefaultMaxConcurrentScoreActivities.
	MaxConcurrentScoreActivities int `json:"max_concurrent_score_activities,omitempty"`
}

type RunAgentWorkflowInput struct {
	RunID      uuid.UUID `json:"run_id"`
	RunAgentID uuid.UUID `json:"run_agent_id"`
}

type AgentHarnessExecutionWorkflowInput struct {
	ExecutionID    uuid.UUID `json:"execution_id"`
	TimeoutSeconds int       `json:"timeout_seconds,omitempty"`
}

type PublicAgentTryoutExecutionWorkflowInput struct {
	TryoutID uuid.UUID `json:"tryout_id"`
}

type SyntheticDatasetGenerationWorkflowInput struct {
	JobID uuid.UUID `json:"job_id"`
}
