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
		if pool.poolLen(PoolKey(req)) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := pool.poolLen(PoolKey(req)); got < 2 {
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
	if got := pool.poolLen(PoolKey(req)); got != 0 {
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
	policy := ToolPolicy{AllowShell: true, MaxToolCalls: 3}
	a := PoolKey(CreateRequest{TemplateID: "t", ToolPolicy: policy})
	b := PoolKey(CreateRequest{TemplateID: "t", ToolPolicy: policy})
	c := PoolKey(CreateRequest{TemplateID: "t", ToolPolicy: ToolPolicy{AllowShell: false, MaxToolCalls: 3}})
	if a != b {
		t.Fatalf("unstable key: %s vs %s", a, b)
	}
	if a == c {
		t.Fatal("different policies should differ")
	}
}

func TestPoolKey_IgnoresRunIdentity(t *testing.T) {
	base := CreateRequest{
		RunID:      uuid.New(),
		RunAgentID: uuid.New(),
		TemplateID: "tmpl",
		ToolPolicy: ToolPolicy{AllowShell: true},
		EnvVars:    map[string]string{"K": "V"},
	}
	other := base
	other.RunID = uuid.New()
	other.RunAgentID = uuid.New()
	if PoolKey(base) != PoolKey(other) {
		t.Fatal("RunID/RunAgentID should not affect pool key")
	}
}

func TestPoolKey_IncludesSandboxConfig(t *testing.T) {
	base := CreateRequest{
		TemplateID: "tmpl",
		ToolPolicy: ToolPolicy{AllowShell: true},
	}
	withEnv := base
	withEnv.EnvVars = map[string]string{"FOO": "bar"}
	if PoolKey(base) == PoolKey(withEnv) {
		t.Fatal("env vars should affect pool key")
	}

	withPackages := base
	withPackages.AdditionalPackages = []string{"git"}
	if PoolKey(base) == PoolKey(withPackages) {
		t.Fatal("additional packages should affect pool key")
	}

	withLabels := base
	withLabels.Labels = map[string]string{"team": "a"}
	if PoolKey(base) == PoolKey(withLabels) {
		t.Fatal("labels should affect pool key")
	}
}

func TestWarmPool_CheckoutMismatchDefense(t *testing.T) {
	inner := &countingProvider{}
	pool := WrapWarmPool(inner, WarmPoolConfig{
		Size:        1,
		TTL:         time.Hour,
		FillTimeout: time.Second,
	})
	defer pool.Close(context.Background())

	base := CreateRequest{
		TemplateID: "tmpl",
		ToolPolicy: ToolPolicy{AllowShell: true},
	}
	withEnv := base
	withEnv.EnvVars = map[string]string{"DIFF": "1"}

	pool.EnsureWarm(context.Background(), base)
	if got := inner.creates.Load(); got != 1 {
		t.Fatalf("warm fills = %d, want 1", got)
	}

	session, err := pool.Create(context.Background(), withEnv)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if session == nil {
		t.Fatal("expected new session on mismatch")
	}
	if inner.creates.Load() != 2 {
		t.Fatalf("mismatch should miss pool and create fresh session, creates=%d", inner.creates.Load())
	}
	_ = session.Destroy(context.Background())
}
