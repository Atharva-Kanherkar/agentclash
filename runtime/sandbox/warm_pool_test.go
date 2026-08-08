package sandbox

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type countingProvider struct {
	creates atomic.Int32
	inner   FakeProvider
}

func (p *countingProvider) Create(ctx context.Context, request CreateRequest) (Session, error) {
	p.creates.Add(1)
	return p.inner.Create(ctx, request)
}

func TestWarmPool_CheckoutAndReplenish(t *testing.T) {
	inner := &countingProvider{}
	metrics := &RecordingMetrics{}
	now := time.Now()
	pool := WrapWarmPool(inner, WarmPoolConfig{
		Size:        2,
		TTL:         time.Hour,
		FillTimeout: time.Second,
		Metrics:     metrics,
		Clock:       func() time.Time { return now },
	})
	if pool == nil {
		t.Fatal("expected warm pool")
	}
	defer pool.Close(context.Background())

	req := CreateRequest{
		RunID:      uuid.New(),
		RunAgentID: uuid.New(),
		TemplateID: "tmpl-a",
		ToolPolicy: ToolPolicy{AllowShell: true},
	}
	pool.EnsureWarm(context.Background(), req)
	if got := inner.creates.Load(); got != 2 {
		t.Fatalf("warm fills = %d, want 2", got)
	}
	if metrics.PoolFills.Load() != 2 {
		t.Fatalf("fill metric = %d, want 2", metrics.PoolFills.Load())
	}

	session, err := pool.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if metrics.PoolHits.Load() != 1 {
		t.Fatalf("hits = %d, want 1", metrics.PoolHits.Load())
	}
	_ = session.Destroy(context.Background())

	// Filler should replenish asynchronously.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if pool.poolLen(PoolKey(req.TemplateID, req.ToolPolicy)) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := pool.poolLen(PoolKey(req.TemplateID, req.ToolPolicy)); got < 2 {
		t.Fatalf("pool len after replenish = %d, want 2 (creates=%d)", got, inner.creates.Load())
	}
}

func TestWarmPool_IdleExpiry(t *testing.T) {
	inner := &countingProvider{}
	metrics := &RecordingMetrics{}
	now := time.Now()
	clock := func() time.Time { return now }
	pool := WrapWarmPool(inner, WarmPoolConfig{
		Size:    1,
		TTL:     time.Minute,
		Metrics: metrics,
		Clock:   clock,
	})
	defer pool.Close(context.Background())

	req := CreateRequest{TemplateID: "tmpl", ToolPolicy: ToolPolicy{AllowNetwork: true}}
	pool.EnsureWarm(context.Background(), req)
	now = now.Add(2 * time.Minute)
	pool.expireIdle()
	if got := pool.poolLen(PoolKey(req.TemplateID, req.ToolPolicy)); got != 0 {
		t.Fatalf("pool len after expiry = %d, want 0", got)
	}
	if metrics.PoolExpires.Load() == 0 {
		t.Fatal("expected expire metric")
	}
}

func TestWrapWarmPool_Disabled(t *testing.T) {
	if WrapWarmPool(&FakeProvider{}, WarmPoolConfig{Size: 0}) != nil {
		t.Fatal("size 0 should disable warm pool")
	}
}

func TestPoolKeyStable(t *testing.T) {
	a := PoolKey("t", ToolPolicy{AllowShell: true, MaxToolCalls: 3})
	b := PoolKey("t", ToolPolicy{AllowShell: true, MaxToolCalls: 3})
	c := PoolKey("t", ToolPolicy{AllowShell: false, MaxToolCalls: 3})
	if a != b {
		t.Fatalf("unstable key: %s vs %s", a, b)
	}
	if a == c {
		t.Fatal("different policies should differ")
	}
}
