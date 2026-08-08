package throttle_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/agentclash/agentclash/runtime/provider/throttle"
)

func TestLocalLimiter_RPM(t *testing.T) {
	lim := throttle.NewLocalLimiter(throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{
			"openai": {RPM: 30}, // 0.5/sec, burst 30
		},
		AcquireTimeout: time.Second,
	})
	key := throttle.Key{Provider: "openai", Credential: "acct-1"}
	ctx := context.Background()

	var acquired int
	for i := 0; i < 5; i++ {
		lease, err := lim.Acquire(ctx, key, 0)
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		acquired++
		lease.Release()
	}
	if acquired != 5 {
		t.Fatalf("acquired = %d", acquired)
	}
}

func TestLocalLimiter_CooldownFromRetryAfter(t *testing.T) {
	lim := throttle.NewLocalLimiter(throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{
			"openai": {RPM: 1000},
		},
		AcquireTimeout: 200 * time.Millisecond,
	})
	key := throttle.Key{Provider: "openai", Credential: "c"}
	lim.CoolDown(key, 500*time.Millisecond)

	start := time.Now()
	_, err := lim.Acquire(context.Background(), key, 0)
	if err != throttle.ErrAcquireTimeout {
		t.Fatalf("err = %v, want ErrAcquireTimeout", err)
	}
	if time.Since(start) < 150*time.Millisecond {
		t.Fatalf("returned too quickly during cooldown")
	}
}

func TestLocalLimiter_TPMReconcile(t *testing.T) {
	lim := throttle.NewLocalLimiter(throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{
			"openai": {TPM: 10_000},
		},
		AcquireTimeout: time.Second,
	})
	key := throttle.Key{Provider: "openai", Credential: "c"}
	lease, err := lim.Acquire(context.Background(), key, 1000)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	lease.Reconcile(100) // return 900 surplus
	lease.Release()

	// Should still be able to reserve near the full budget.
	lease2, err := lim.Acquire(context.Background(), key, 9000)
	if err != nil {
		t.Fatalf("acquire after reconcile: %v", err)
	}
	lease2.Release()
}

func TestThrottledClient_AcquireTimeout(t *testing.T) {
	inner := &provider.FakeClient{}
	lim := throttle.NewLocalLimiter(throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{
			"openai": {MaxConcurrent: 1},
		},
		AcquireTimeout: 50 * time.Millisecond,
	})
	client := throttle.Wrap(inner, lim, throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{"openai": {MaxConcurrent: 1}},
		AcquireTimeout:   50 * time.Millisecond,
	})

	// Hold the only slot.
	lease, err := lim.Acquire(context.Background(), throttle.Key{Provider: "openai"}, 0)
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	defer lease.Release()

	_, err = client.InvokeModel(context.Background(), provider.Request{
		ProviderKey: "openai",
		Messages:    []provider.Message{{Role: "user", Content: "hi"}},
	})
	failure, ok := provider.AsFailure(err)
	if !ok || failure.Code != provider.FailureCodeRateLimit {
		t.Fatalf("err = %v, want rate_limit failure", err)
	}
}

func TestWrap_NoopWhenDisabled(t *testing.T) {
	inner := &provider.FakeClient{}
	got := throttle.Wrap(inner, throttle.NewLocalLimiter(throttle.Config{}), throttle.Config{})
	if got != inner {
		t.Fatal("expected passthrough when limits disabled")
	}
}

func TestThrottledClient_SharedBudgetEngineAndJudge(t *testing.T) {
	var calls atomic.Int32
	inner := &countingClient{fn: func() {
		calls.Add(1)
		time.Sleep(30 * time.Millisecond)
	}}
	cfg := throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{"openai": {MaxConcurrent: 1}},
		AcquireTimeout:   time.Second,
	}
	lim := throttle.NewLocalLimiter(cfg)
	client := throttle.Wrap(inner, lim, cfg)

	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.InvokeModel(context.Background(), provider.Request{
				ProviderKey:         "openai",
				CredentialReference: "shared",
				Messages:            []provider.Message{{Content: "x"}},
			})
		}()
	}
	wg.Wait()
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Fatalf("expected serialization across shared client; elapsed %s", time.Since(start))
	}
}

type countingClient struct {
	fn func()
}

func (c *countingClient) InvokeModel(context.Context, provider.Request) (provider.Response, error) {
	c.fn()
	return provider.Response{Usage: provider.Usage{TotalTokens: 10}}, nil
}
