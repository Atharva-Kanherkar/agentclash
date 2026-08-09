package redisbudget_test

import (
	"context"
	"testing"
	"time"

	"github.com/agentclash/agentclash/backend/internal/sandbox/redisbudget"
	"github.com/agentclash/agentclash/runtime/sandbox"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisBudget_SharedAcrossClients(t *testing.T) {
	if testing.Short() {
		t.Skip("redis budget integration skipped under -short")
	}

	mr := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	clientB := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = clientA.Close()
		_ = clientB.Close()
	})

	budgetA := redisbudget.New(redisbudget.Config{
		Client:         clientA,
		MaxConcurrent:  2,
		AcquireTimeout: time.Second,
		PollInterval:   10 * time.Millisecond,
	})
	budgetB := redisbudget.New(redisbudget.Config{
		Client:         clientB,
		MaxConcurrent:  2,
		AcquireTimeout: time.Second,
		PollInterval:   10 * time.Millisecond,
	})

	ctx := context.Background()
	release1, err := budgetA.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire A1: %v", err)
	}
	release2, err := budgetB.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire B1: %v", err)
	}

	type acquireResult struct {
		release func()
		err     error
	}
	resultCh := make(chan acquireResult, 1)
	go func() {
		release, err := budgetA.Acquire(ctx)
		resultCh <- acquireResult{release: release, err: err}
	}()

	time.Sleep(80 * time.Millisecond)
	select {
	case got := <-resultCh:
		t.Fatalf("third acquire should still be waiting, got err=%v", got.err)
	default:
	}

	release1()
	got := <-resultCh
	if got.err != nil {
		t.Fatalf("third acquire after release: %v", got.err)
	}
	got.release()
	release2()

	// Exhaust then timeout.
	r1, err := budgetA.Acquire(ctx)
	if err != nil {
		t.Fatalf("re-acquire A: %v", err)
	}
	r2, err := budgetB.Acquire(ctx)
	if err != nil {
		t.Fatalf("re-acquire B: %v", err)
	}
	_, err = budgetA.Acquire(ctx)
	if !sandbox.IsCapacityError(err) {
		t.Fatalf("expected capacity timeout, got %v", err)
	}
	r1()
	r2()
}

func TestRedisBudget_RenewsHeldSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("redis budget integration skipped under -short")
	}

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	budget := redisbudget.New(redisbudget.Config{
		Client:         client,
		MaxConcurrent:  1,
		AcquireTimeout: time.Second,
		SlotTTL:        200 * time.Millisecond,
		RenewInterval:  50 * time.Millisecond,
		PollInterval:   10 * time.Millisecond,
		LeaseID:        func() string { return "lease-renew" },
	})

	ctx := context.Background()
	release, err := budget.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	mr.FastForward(150 * time.Millisecond)

	release2, err := budget.Acquire(ctx)
	if err == nil {
		release2()
		t.Fatal("expected capacity full while lease is held and renewed")
	}
	if !sandbox.IsCapacityError(err) {
		t.Fatalf("expected capacity error, got %v", err)
	}

	release()
	time.Sleep(20 * time.Millisecond)

	release3, err := budget.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	release3()
}

func TestRedisBudget_SlotExpiresWithoutRenewal(t *testing.T) {
	if testing.Short() {
		t.Skip("redis budget integration skipped under -short")
	}

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	key := "agentclash:sandbox:capacity:expire"
	member := "stale-lease"
	now := time.Now().Unix()
	expired := time.Now().Add(-time.Minute).Unix()
	if err := client.ZAdd(context.Background(), key, redis.Z{Score: float64(expired), Member: member}).Err(); err != nil {
		t.Fatalf("seed zset: %v", err)
	}
	_ = now

	budget := redisbudget.New(redisbudget.Config{
		Client:         client,
		MaxConcurrent:  1,
		AcquireTimeout: time.Second,
		SlotTTL:        time.Hour,
		Key:            key,
		PollInterval:   10 * time.Millisecond,
		LeaseID:        func() string { return "fresh-lease" },
	})

	release, err := budget.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire should prune expired member: %v", err)
	}
	release()
}
