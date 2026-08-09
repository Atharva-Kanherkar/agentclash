package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/agentclash/agentclash/backend/internal/sandbox/e2b"
	"github.com/agentclash/agentclash/backend/internal/sandbox/redisbudget"
	"github.com/agentclash/agentclash/runtime/sandbox"
	"github.com/agentclash/agentclash/runtime/sandbox/docker"
	"github.com/redis/go-redis/v9"
)

// SandboxStack is the wired sandbox provider plus optional cleanup.
type SandboxStack struct {
	Provider sandbox.Provider
	closeFns []func(context.Context) error
}

// Close destroys warm-pool state and closes owned docker clients.
func (s *SandboxStack) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	var first error
	for i := len(s.closeFns) - 1; i >= 0; i-- {
		if err := s.closeFns[i](ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// BuildSandboxProvider constructs the configured sandbox provider with optional
// capacity middleware (Redis-backed when redisClient is non-nil) and warm pool.
//
// Decorator order: WarmPool(Capacity(inner)). Capacity MaxConcurrent=0 and
// WarmPoolSize=0 preserve today's passthrough behavior.
func BuildSandboxProvider(cfg Config, redisClient *redis.Client, logger *slog.Logger) (*SandboxStack, error) {
	if logger == nil {
		logger = slog.Default()
	}
	stack := &SandboxStack{}

	var inner sandbox.Provider = sandbox.UnconfiguredProvider{}
	switch cfg.Sandbox.Provider {
	case "unconfigured":
		// keep noop
	case "e2b":
		inner = e2b.NewProvider(e2b.Config{
			APIKey:         cfg.Sandbox.E2B.APIKey,
			TemplateID:     cfg.Sandbox.E2B.TemplateID,
			APIBaseURL:     cfg.Sandbox.E2B.APIBaseURL,
			RequestTimeout: cfg.Sandbox.E2B.RequestTimeout,
		})
	case "docker":
		pullMissing := cfg.Sandbox.Docker.PullMissing
		dockerProvider, err := docker.NewProvider(docker.Config{
			Host:               cfg.Sandbox.Docker.Host,
			Image:              cfg.Sandbox.Docker.Image,
			PullMissing:        &pullMissing,
			StopTimeout:        cfg.Sandbox.Docker.StopTimeout,
			MaxExecOutputBytes: cfg.Sandbox.Docker.MaxExecOutputBytes,
			MemoryBytes:        cfg.Sandbox.Docker.MemoryBytes,
			NanoCPUs:           cfg.Sandbox.Docker.NanoCPUs,
		})
		if err != nil {
			return nil, fmt.Errorf("docker sandbox provider: %w", err)
		}
		inner = dockerProvider
		stack.closeFns = append(stack.closeFns, func(context.Context) error {
			return dockerProvider.Close()
		})
	default:
		return nil, fmt.Errorf("unsupported SANDBOX_PROVIDER %q", cfg.Sandbox.Provider)
	}

	provider := inner
	if cfg.Sandbox.MaxConcurrent > 0 {
		var budget sandbox.Budget
		if redisClient != nil {
			budget = redisbudget.New(redisbudget.Config{
				Client:         redisClient,
				MaxConcurrent:  cfg.Sandbox.MaxConcurrent,
				AcquireTimeout: cfg.Sandbox.AcquireTimeout,
			})
			logger.Info("sandbox capacity: redis budget enabled",
				"max_concurrent", cfg.Sandbox.MaxConcurrent,
				"acquire_timeout", cfg.Sandbox.AcquireTimeout)
		} else {
			budget = sandbox.NewLocalBudget(cfg.Sandbox.MaxConcurrent, cfg.Sandbox.AcquireTimeout)
			logger.Info("sandbox capacity: local budget enabled",
				"max_concurrent", cfg.Sandbox.MaxConcurrent,
				"acquire_timeout", cfg.Sandbox.AcquireTimeout)
		}
		provider = sandbox.WrapCapacity(provider, sandbox.CapacityConfig{
			MaxConcurrent:  cfg.Sandbox.MaxConcurrent,
			AcquireTimeout: cfg.Sandbox.AcquireTimeout,
			Budget:         budget,
		})
	}

	if warm := sandbox.WrapWarmPool(provider, sandbox.WarmPoolConfig{
		Size:        cfg.Sandbox.WarmPoolSize,
		TTL:         cfg.Sandbox.WarmPoolTTL,
		FillTimeout: cfg.Sandbox.AcquireTimeout,
		Logger:      logger,
	}); warm != nil {
		warm.Start()
		stack.closeFns = append(stack.closeFns, warm.Close)
		provider = warm
		logger.Info("sandbox warm pool: enabled (per-worker)",
			"size", cfg.Sandbox.WarmPoolSize,
			"ttl", cfg.Sandbox.WarmPoolTTL)
		if cfg.Sandbox.Provider == "e2b" && cfg.Sandbox.E2B.TemplateID != "" {
			go warm.EnsureWarm(context.Background(), sandbox.CreateRequest{
				TemplateID: cfg.Sandbox.E2B.TemplateID,
			})
		}
	}

	stack.Provider = provider
	return stack, nil
}
