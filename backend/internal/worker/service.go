package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	workflowpkg "github.com/Atharva-Kanherkar/agentclash/backend/internal/workflow"
	temporalsdk "go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
)

type TemporalWorker interface {
	Start() error
	Stop()
}

type Dependencies struct {
	ExecutionHooks workflowpkg.FakeWorkHooks
}

func NewTemporalWorker(client temporalsdk.Client, cfg Config, repo *repository.Repository, deps Dependencies) TemporalWorker {
	temporalWorker := sdkworker.New(client, cfg.TaskQueue, sdkworker.Options{
		Identity: cfg.Identity,
	})

	activities := workflowpkg.NewActivities(repo, deps.ExecutionHooks)
	workflowpkg.Register(temporalWorker, activities)

	return temporalWorker
}

func Run(ctx context.Context, cfg Config, temporalWorker TemporalWorker, logger *slog.Logger) error {
	logger.Info("starting worker",
		"task_queue", cfg.TaskQueue,
		"identity", cfg.Identity,
		"temporal_address", cfg.TemporalAddress,
		"temporal_namespace", cfg.TemporalNamespace,
	)

	if err := temporalWorker.Start(); err != nil {
		return fmt.Errorf("start temporal worker: %w", err)
	}

	<-ctx.Done()

	logger.Info("stopping worker", "shutdown_timeout", cfg.ShutdownTimeout.String())

	stopErrCh := make(chan error, 1)
	go func() {
		temporalWorker.Stop()
		stopErrCh <- nil
	}()

	timer := time.NewTimer(cfg.ShutdownTimeout)
	defer timer.Stop()

	select {
	case err := <-stopErrCh:
		return err
	case <-timer.C:
		return fmt.Errorf("worker shutdown timed out after %s", cfg.ShutdownTimeout)
	}
}
