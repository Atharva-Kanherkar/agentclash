package workflow

import (
	"fmt"

	"github.com/google/uuid"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

const (
	ScanEvalSetWorkflowName                 = "ScanEvalSetWorkflow"
	DefaultMaxConcurrentScanTargets         = 8
	scanEvalSetBoundedFanoutVersionChangeID = "scan-eval-set-bounded-fanout"
)

type ScanEvalSetWorkflowInput struct {
	EvalSetID uuid.UUID `json:"eval_set_id"`
	Scanners  []string  `json:"scanners"`
}

func ScanEvalSetWorkflow(ctx sdkworkflow.Context, input ScanEvalSetWorkflowInput) error {
	ctx = sdkworkflow.WithActivityOptions(ctx, sdkworkflow.ActivityOptions{
		TaskQueue:           TaskQueueBackground,
		StartToCloseTimeout: defaultActivityOptions.StartToCloseTimeout,
		RetryPolicy:         defaultActivityOptions.RetryPolicy,
	})

	var targets []ScanTarget
	if err := sdkworkflow.ExecuteActivity(ctx, listScanTargetsActivityName, ListScanTargetsInput{
		EvalSetID: input.EvalSetID,
		Scanners:  input.Scanners,
	}).Get(ctx, &targets); err != nil {
		return fmt.Errorf("list scan targets: %w", err)
	}
	if len(targets) == 0 {
		return nil
	}

	_ = boundedFanoutVersion(ctx, scanEvalSetBoundedFanoutVersionChangeID)
	cap := resolvePositiveCap(DefaultMaxConcurrentScanTargets, DefaultMaxConcurrentScanTargets)
	launch := func(index int) (sdkworkflow.Future, sdkworkflow.CancelFunc) {
		childCtx, cancel := sdkworkflow.WithCancel(ctx)
		future := sdkworkflow.ExecuteActivity(childCtx, scanOneTargetActivityName, targets[index])
		return future, cancel
	}
	onComplete := func(index int, future sdkworkflow.Future) error {
		if err := future.Get(ctx, nil); err != nil {
			return err
		}
		return nil
	}
	if err := launchBounded(ctx, cap, len(targets), launch, onComplete, nil); err != nil {
		return fmt.Errorf("scan eval set: %w", err)
	}
	return nil
}
