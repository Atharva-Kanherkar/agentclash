package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/repository"
	workerapp "github.com/Atharva-Kanherkar/agentclash/backend/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	temporalsdk "go.temporal.io/sdk/client"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := workerapp.LoadConfigFromEnv()
	if err != nil {
		logger.Error("failed to load worker config", "error", err)
		os.Exit(1)
	}

	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	temporalClient, err := temporalsdk.Dial(temporalsdk.Options{
		HostPort:  cfg.TemporalAddress,
		Namespace: cfg.TemporalNamespace,
	})
	if err != nil {
		logger.Error("failed to connect to temporal", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	repo := repository.New(db)
	temporalWorker := workerapp.NewTemporalWorker(temporalClient, cfg, repo, workerapp.Dependencies{})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := workerapp.Run(ctx, cfg, temporalWorker, logger); err != nil {
		logger.Error("worker stopped with error", "error", err)
		os.Exit(1)
	}
}
