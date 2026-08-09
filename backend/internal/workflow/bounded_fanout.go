package workflow

import (
	"fmt"
	"sort"

	sdkworkflow "go.temporal.io/sdk/workflow"
)

// Default in-flight caps for Fleet 2 bounded orchestration. Workflow code must
// not read env (nondeterministic); these are the deterministic defaults.
// Callers may override via workflow input fields when plumbed from config.
const (
	DefaultMaxConcurrentEvalSessionRuns = 16
	DefaultMaxConcurrentRunAgents       = 8
	DefaultMaxConcurrentScoreActivities = 16

	evalSessionBoundedFanoutVersionChangeID = "eval-session-bounded-fanout"
	runAgentsBoundedFanoutVersionChangeID   = "run-agents-bounded-fanout"
	scoreAgentsBoundedFanoutVersionChangeID = "score-agents-bounded-fanout"
	// Separate from bounded-fanout: pinning TaskQueue on children/activities
	// changes Temporal command attributes and must be replay-gated.
	taskQueuePartitionVersionChangeID = "fleet-task-queue-partition"
)

// Task queue class names (Fleet 2 partitioning). Workflows start on
// TaskQueueExecution; activities pin to their class via ActivityOptions.TaskQueue.
const (
	TaskQueueExecution  = "execution"
	TaskQueueScoring    = "scoring"
	TaskQueueBackground = "background"
	// LegacyTaskQueue is the pre-Fleet single queue name. Workers that list it
	// (or set WORKER_TASK_QUEUE to it) are treated as serving all three classes.
	LegacyTaskQueue = "RunWorkflow"
)

// launchBounded runs launch(i) for i in [0,n) with at most maxInFlight futures
// outstanding. Completions are processed via onComplete. Registration order is
// made deterministic by sorting pending futures by index each drain round.
//
// beforeLaunch, when non-nil, is invoked immediately before each launch(i). If
// it returns an error, no further launches occur; pending children are cancelled
// via the CancelFunc returned from launch (Fleet 13 budget gate), then drained.
//
// This is workflow-safe: no goroutines or channels.
func launchBounded(
	ctx sdkworkflow.Context,
	maxInFlight int,
	n int,
	launch func(index int) (future sdkworkflow.Future, cancel sdkworkflow.CancelFunc),
	onComplete func(index int, future sdkworkflow.Future) error,
	beforeLaunch func(index int) error,
) error {
	if n <= 0 {
		return nil
	}
	if maxInFlight <= 0 {
		maxInFlight = 1
	}
	if maxInFlight > n {
		maxInFlight = n
	}

	type pendingItem struct {
		index  int
		future sdkworkflow.Future
		cancel sdkworkflow.CancelFunc
	}
	pending := make(map[sdkworkflow.Future]pendingItem, maxInFlight)
	nextIndex := 0
	var gateErr error

	start := func(index int) error {
		if beforeLaunch != nil {
			if err := beforeLaunch(index); err != nil {
				return err
			}
		}
		future, cancel := launch(index)
		pending[future] = pendingItem{index: index, future: future, cancel: cancel}
		return nil
	}

	cancelPending := func() {
		for _, item := range pending {
			if item.cancel != nil {
				item.cancel()
			}
		}
	}

	for nextIndex < maxInFlight {
		if err := start(nextIndex); err != nil {
			gateErr = err
			cancelPending()
			break
		}
		nextIndex++
	}

	for len(pending) > 0 {
		selector := sdkworkflow.NewSelector(ctx)
		futures := make([]sdkworkflow.Future, 0, len(pending))
		for f := range pending {
			futures = append(futures, f)
		}
		sort.Slice(futures, func(i, j int) bool {
			return pending[futures[i]].index < pending[futures[j]].index
		})

		var (
			completed   pendingItem
			completeErr error
		)
		for _, f := range futures {
			future := f
			item := pending[future]
			selector.AddFuture(future, func(fut sdkworkflow.Future) {
				completed = item
				completeErr = onComplete(item.index, fut)
			})
		}
		selector.Select(ctx)
		if completeErr != nil {
			cancelPending()
			delete(pending, completed.future)
			// Drain remaining after cancel so histories settle cleanly.
			for len(pending) > 0 {
				drainSelector := sdkworkflow.NewSelector(ctx)
				for f := range pending {
					future := f
					item := pending[future]
					drainSelector.AddFuture(future, func(sdkworkflow.Future) {
						completed = item
						_ = onComplete(item.index, future)
					})
				}
				drainSelector.Select(ctx)
				delete(pending, completed.future)
			}
			return completeErr
		}
		delete(pending, completed.future)

		if gateErr != nil {
			continue
		}
		if nextIndex < n {
			if err := start(nextIndex); err != nil {
				gateErr = err
				cancelPending()
				continue
			}
			nextIndex++
		}
	}
	return gateErr
}

// launchAllUnbounded preserves the pre-Fleet launch-all-then-drain shape for
// GetVersion DefaultVersion replay of in-flight histories.
func launchAllUnbounded(
	ctx sdkworkflow.Context,
	n int,
	launch func(index int) sdkworkflow.Future,
	onComplete func(index int, future sdkworkflow.Future) error,
) error {
	if n <= 0 {
		return nil
	}
	selector := sdkworkflow.NewSelector(ctx)
	type item struct {
		index  int
		future sdkworkflow.Future
	}
	items := make([]item, 0, n)
	for i := 0; i < n; i++ {
		future := launch(i)
		items = append(items, item{index: i, future: future})
	}
	completed := 0
	var firstErr error
	for _, it := range items {
		it := it
		selector.AddFuture(it.future, func(fut sdkworkflow.Future) {
			completed++
			if err := onComplete(it.index, fut); err != nil && firstErr == nil {
				firstErr = err
			}
		})
	}
	for completed < n {
		selector.Select(ctx)
	}
	return firstErr
}

func resolvePositiveCap(value, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 1
}

func boundedFanoutVersion(ctx sdkworkflow.Context, changeID string) sdkworkflow.Version {
	return sdkworkflow.GetVersion(ctx, changeID, sdkworkflow.DefaultVersion, 1)
}

func taskQueuePartitionVersion(ctx sdkworkflow.Context) sdkworkflow.Version {
	return sdkworkflow.GetVersion(ctx, taskQueuePartitionVersionChangeID, sdkworkflow.DefaultVersion, 1)
}

// withChildExecutionTaskQueue pins child workflows to TaskQueueExecution only
// for workflows started after the partition change. DefaultVersion omits
// TaskQueue so replay matches pre-Fleet histories (inherited parent queue).
func withChildExecutionTaskQueue(ctx sdkworkflow.Context, opts sdkworkflow.ChildWorkflowOptions) sdkworkflow.ChildWorkflowOptions {
	if taskQueuePartitionVersion(ctx) != sdkworkflow.DefaultVersion {
		opts.TaskQueue = TaskQueueExecution
	}
	return opts
}

// withActivityTaskQueue pins activities to a class queue after the partition
// change. DefaultVersion leaves TaskQueue unset for replay compatibility.
func withActivityTaskQueue(ctx sdkworkflow.Context, opts sdkworkflow.ActivityOptions, queue string) sdkworkflow.ActivityOptions {
	if taskQueuePartitionVersion(ctx) != sdkworkflow.DefaultVersion {
		opts.TaskQueue = queue
	}
	return opts
}

// adaptLaunch wraps a legacy index→Future launcher for launchBounded.
func adaptLaunch(launch func(index int) sdkworkflow.Future) func(index int) (sdkworkflow.Future, sdkworkflow.CancelFunc) {
	return func(index int) (sdkworkflow.Future, sdkworkflow.CancelFunc) {
		return launch(index), nil
	}
}

// AllTaskQueues returns the three Fleet queue class names in stable order.
func AllTaskQueues() []string {
	return []string{TaskQueueExecution, TaskQueueScoring, TaskQueueBackground}
}

// ExpandTaskQueues normalizes a configured queue list. "RunWorkflow" (legacy)
// expands to all three class queues so a single-process upgrade matches today.
func ExpandTaskQueues(queues []string) []string {
	if len(queues) == 0 {
		return AllTaskQueues()
	}
	seen := make(map[string]struct{}, len(queues))
	out := make([]string, 0, len(queues))
	add := func(q string) {
		if q == "" {
			return
		}
		if _, ok := seen[q]; ok {
			return
		}
		seen[q] = struct{}{}
		out = append(out, q)
	}
	for _, q := range queues {
		if q == LegacyTaskQueue {
			for _, class := range AllTaskQueues() {
				add(class)
			}
			continue
		}
		switch q {
		case TaskQueueExecution, TaskQueueScoring, TaskQueueBackground:
			add(q)
		default:
			add(q)
		}
	}
	if len(out) == 0 {
		return AllTaskQueues()
	}
	return out
}

func validateTaskQueueClass(name string) error {
	switch name {
	case TaskQueueExecution, TaskQueueScoring, TaskQueueBackground, LegacyTaskQueue:
		return nil
	default:
		return fmt.Errorf("unknown task queue class %q", name)
	}
}
