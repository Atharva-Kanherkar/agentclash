package throttle_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agentclash/agentclash/runtime/provider"
	"github.com/agentclash/agentclash/runtime/provider/throttle"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newMiniRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestRedisLimiter_SharedRPM(t *testing.T) {
	// miniredis — safe under -short; models two worker processes sharing Redis.
	rdb := newMiniRedis(t)
	cfg := throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{"openai": {RPM: 5}},
		AcquireTimeout:   80 * time.Millisecond,
	}
	a := throttle.NewRedisLimiter(rdb, cfg)
	b := throttle.NewRedisLimiter(rdb, cfg)
	key := throttle.Key{Provider: "openai", Credential: "shared"}

	var okCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		lim := a
		if i%2 == 1 {
			lim = b
		}
		go func(lim *throttle.RedisLimiter) {
			defer wg.Done()
			lease, err := lim.Acquire(context.Background(), key, 0)
			if err != nil {
				return
			}
			okCount.Add(1)
			lease.Release()
		}(lim)
	}
	wg.Wait()

	got := okCount.Load()
	if got != 5 {
		t.Fatalf("shared RPM successes = %d, want 5 (two workers must not exceed budget)", got)
	}
}

func TestRedisLimiter_SharedCooldown(t *testing.T) {
	rdb := newMiniRedis(t)
	cfg := throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{"openai": {RPM: 1000}},
		AcquireTimeout:   150 * time.Millisecond,
	}
	a := throttle.NewRedisLimiter(rdb, cfg)
	b := throttle.NewRedisLimiter(rdb, cfg)
	key := throttle.Key{Provider: "openai", Credential: "c"}

	// Simulate 429 Retry-After from worker A.
	a.CoolDown(key, 500*time.Millisecond)

	_, err := b.Acquire(context.Background(), key, 0)
	if err != throttle.ErrAcquireTimeout {
		t.Fatalf("worker B acquire during cooldown: %v, want timeout", err)
	}
}

func TestRedisLimiter_TPMReconcile(t *testing.T) {
	rdb := newMiniRedis(t)
	cfg := throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{"openai": {TPM: 10_000}},
		AcquireTimeout:   time.Second,
	}
	lim := throttle.NewRedisLimiter(rdb, cfg)
	key := throttle.Key{Provider: "openai", Credential: "c"}

	lease, err := lim.Acquire(context.Background(), key, 1000)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	lease.Reconcile(100)
	lease.Release()

	lease2, err := lim.Acquire(context.Background(), key, 9000)
	if err != nil {
		t.Fatalf("acquire after reconcile: %v", err)
	}
	lease2.Release()
}

func TestThrottledClient_FeedsCooldownOn429(t *testing.T) {
	inner := &provider.FakeClient{}
	inner.Err = provider.Failure{
		ProviderKey: "openai",
		Code:        provider.FailureCodeRateLimit,
		Message:     "slow down",
		Retryable:   true,
		RetryAfter:  300 * time.Millisecond,
	}

	cfg := throttle.Config{
		LimitsByProvider: map[string]throttle.Limits{"openai": {RPM: 1000}},
		AcquireTimeout:   100 * time.Millisecond,
	}
	lim := throttle.NewLocalLimiter(cfg)
	client := throttle.Wrap(inner, lim, cfg)

	_, err := client.InvokeModel(context.Background(), provider.Request{ProviderKey: "openai"})
	if err == nil {
		t.Fatal("expected rate limit error")
	}

	start := time.Now()
	_, err = lim.Acquire(context.Background(), throttle.Key{Provider: "openai"}, 0)
	if err != throttle.ErrAcquireTimeout {
		t.Fatalf("acquire during cooldown: %v", err)
	}
	if time.Since(start) < 80*time.Millisecond {
		t.Fatal("cooldown too short")
	}
}
