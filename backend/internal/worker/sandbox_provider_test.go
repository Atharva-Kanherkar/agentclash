package worker

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/agentclash/agentclash/runtime/sandbox"
)

func TestBuildSandboxProvider_UnconfiguredPassthrough(t *testing.T) {
	stack, err := BuildSandboxProvider(Config{
		Sandbox: SandboxConfig{Provider: "unconfigured"},
	}, nil, slog.Default())
	if err != nil {
		t.Fatalf("BuildSandboxProvider: %v", err)
	}
	defer stack.Close(context.Background())

	_, err = stack.Provider.Create(context.Background(), sandbox.CreateRequest{})
	if err != sandbox.ErrProviderNotConfigured {
		t.Fatalf("error = %v, want ErrProviderNotConfigured", err)
	}
}

func TestBuildSandboxProvider_CapacityEnabled(t *testing.T) {
	stack, err := BuildSandboxProvider(Config{
		Sandbox: SandboxConfig{
			Provider:       "unconfigured",
			MaxConcurrent:  1,
			AcquireTimeout: time.Second,
		},
	}, nil, slog.Default())
	if err != nil {
		t.Fatalf("BuildSandboxProvider: %v", err)
	}
	defer stack.Close(context.Background())

	if _, ok := stack.Provider.(*sandbox.CapacityProvider); !ok {
		t.Fatalf("provider type = %T, want *sandbox.CapacityProvider", stack.Provider)
	}
	_, err = stack.Provider.Create(context.Background(), sandbox.CreateRequest{})
	if err != sandbox.ErrProviderNotConfigured {
		t.Fatalf("create error = %v, want ErrProviderNotConfigured", err)
	}
}
