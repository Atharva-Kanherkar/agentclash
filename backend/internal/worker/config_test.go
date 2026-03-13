package worker

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Atharva-Kanherkar/agentclash/backend/internal/workflow"
)

func TestLoadConfigFromEnvUsesDefaultsWhenUnset(t *testing.T) {
	unsetEnv(t, "DATABASE_URL")
	unsetEnv(t, "TEMPORAL_HOST_PORT")
	unsetEnv(t, "TEMPORAL_NAMESPACE")
	unsetEnv(t, "WORKER_IDENTITY")
	unsetEnv(t, "WORKER_SHUTDOWN_TIMEOUT")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv returned error: %v", err)
	}

	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
	if cfg.TemporalAddress != defaultTemporalTarget {
		t.Fatalf("TemporalAddress = %q, want %q", cfg.TemporalAddress, defaultTemporalTarget)
	}
	if cfg.TemporalNamespace != defaultNamespace {
		t.Fatalf("TemporalNamespace = %q, want %q", cfg.TemporalNamespace, defaultNamespace)
	}
	if cfg.TaskQueue != workflow.RunWorkflowName {
		t.Fatalf("TaskQueue = %q, want %q", cfg.TaskQueue, workflow.RunWorkflowName)
	}
	if cfg.Identity == "" {
		t.Fatalf("Identity was empty")
	}
	if cfg.ShutdownTimeout != defaultShutdownTime {
		t.Fatalf("ShutdownTimeout = %s, want %s", cfg.ShutdownTimeout, defaultShutdownTime)
	}
}

func TestLoadConfigFromEnvRejectsEmptyDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("LoadConfigFromEnv returned nil error")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error = %v, want ErrInvalidConfig", err)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if ok {
		t.Cleanup(func() {
			_ = os.Setenv(key, value)
		})
	} else {
		t.Cleanup(func() {
			_ = os.Unsetenv(key)
		})
	}

	_ = os.Unsetenv(key)
}

func TestLoadConfigFromEnvOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("TEMPORAL_HOST_PORT", "temporal.example:7233")
	t.Setenv("TEMPORAL_NAMESPACE", "agentclash-dev")
	t.Setenv("WORKER_IDENTITY", "worker-dev-1")
	t.Setenv("WORKER_SHUTDOWN_TIMEOUT", "30s")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv returned error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://example")
	}
	if cfg.TemporalAddress != "temporal.example:7233" {
		t.Fatalf("TemporalAddress = %q, want %q", cfg.TemporalAddress, "temporal.example:7233")
	}
	if cfg.TemporalNamespace != "agentclash-dev" {
		t.Fatalf("TemporalNamespace = %q, want %q", cfg.TemporalNamespace, "agentclash-dev")
	}
	if cfg.Identity != "worker-dev-1" {
		t.Fatalf("Identity = %q, want %q", cfg.Identity, "worker-dev-1")
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want %s", cfg.ShutdownTimeout, 30*time.Second)
	}
}

func TestLoadConfigFromEnvRejectsInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("WORKER_SHUTDOWN_TIMEOUT", "later")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("LoadConfigFromEnv returned nil error")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("error = %v, want ErrInvalidConfig", err)
	}
}
