package worker

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/workflow"
)

const (
	defaultDatabaseURL    = "postgres://agentclash:agentclash@localhost:5432/agentclash?sslmode=disable"
	defaultTemporalTarget = "localhost:7233"
	defaultNamespace      = "default"
	defaultShutdownTime   = 10 * time.Second
)

var ErrInvalidConfig = errors.New("invalid worker config")

type Config struct {
	DatabaseURL       string
	TemporalAddress   string
	TemporalNamespace string
	Identity          string
	TaskQueue         string
	ShutdownTimeout   time.Duration
}

func LoadConfigFromEnv() (Config, error) {
	databaseURL, err := envOrDefault("DATABASE_URL", defaultDatabaseURL)
	if err != nil {
		return Config{}, err
	}
	temporalAddress, err := envOrDefault("TEMPORAL_HOST_PORT", defaultTemporalTarget)
	if err != nil {
		return Config{}, err
	}
	temporalNamespace, err := envOrDefault("TEMPORAL_NAMESPACE", defaultNamespace)
	if err != nil {
		return Config{}, err
	}
	identity, err := envOrDefault("WORKER_IDENTITY", defaultWorkerIdentity())
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := durationEnvOrDefault("WORKER_SHUTDOWN_TIMEOUT", defaultShutdownTime)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:       databaseURL,
		TemporalAddress:   temporalAddress,
		TemporalNamespace: temporalNamespace,
		Identity:          identity,
		TaskQueue:         workflow.RunWorkflowName,
		ShutdownTimeout:   shutdownTimeout,
	}, nil
}

func envOrDefault(key string, fallback string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	if value == "" {
		return "", fmt.Errorf("%w: %s cannot be empty", ErrInvalidConfig, key)
	}

	return value, nil
}

func durationEnvOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	if value == "" {
		return 0, fmt.Errorf("%w: %s cannot be empty", ErrInvalidConfig, key)
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%w: %s must be a valid duration: %v", ErrInvalidConfig, key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%w: %s must be greater than zero", ErrInvalidConfig, key)
	}

	return duration, nil
}

func defaultWorkerIdentity() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "agentclash-worker"
	}

	return fmt.Sprintf("agentclash-worker@%s", hostname)
}
