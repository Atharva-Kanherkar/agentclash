package sandbox

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type trackingProvider struct {
	mu        sync.Mutex
	inFlight  int32
	maxSeen   int32
	block     chan struct{}
	created   int32
	createErr error
}

func (p *trackingProvider) Create(ctx context.Context, request CreateRequest) (Session, error) {
	current := atomic.AddInt32(&p.inFlight, 1)
	for {
		prev := atomic.LoadInt32(&p.maxSeen)
		if current <= prev || atomic.CompareAndSwapInt32(&p.maxSeen, prev, current) {
			break
		}
	}
	defer atomic.AddInt32(&p.inFlight, -1)

	if p.block != nil {
		select {
		case <-p.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if p.createErr != nil {
		return nil, p.createErr
	}
	atomic.AddInt32(&p.created, 1)
	return NewFakeSession("track-" + request.RunAgentID.String()), nil
}

func TestCapacityProvider_SerializesCreates(t *testing.T) {
	inner := &trackingProvider{block: make(chan struct{})}
	metrics := &RecordingMetrics{}
	provider := WrapCapacity(inner, CapacityConfig{
		MaxConcurrent:  2,
		AcquireTimeout: 5 * time.Second,
		Metrics:        metrics,
	})

	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make([]error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session, err := provider.Create(ctx, CreateRequest{
				RunID:      uuid.New(),
				RunAgentID: uuid.New(),
			})
			errs[i] = err
			if err != nil {
				return
			}
			// Release immediately so waiting creates can proceed; the
			// concurrency bound is enforced while Create is in flight.
			_ = session.Destroy(ctx)
		}(i)
	}

	// Let goroutines contend for slots while creates are blocked.
	deadline := time.Now().Add(time.Second)
	for atomic.LoadInt32(&inner.inFlight) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("expected 2 in-flight creates")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&inner.maxSeen); got > 2 {
		t.Fatalf("max in-flight creates = %d, want <= 2", got)
	}
	close(inner.block)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("create[%d]: %v", i, err)
		}
	}
	if got := atomic.LoadInt32(&inner.maxSeen); got > 2 {
		t.Fatalf("max in-flight after unblock = %d, want <= 2", got)
	}
	if metrics.Acquired.Load() != 5 {
		t.Fatalf("acquired metric = %d, want 5", metrics.Acquired.Load())
	}
}

func TestCapacityProvider_AcquireTimeout(t *testing.T) {
	inner := &trackingProvider{block: make(chan struct{})}
	provider := WrapCapacity(inner, CapacityConfig{
		MaxConcurrent:  1,
		AcquireTimeout: 50 * time.Millisecond,
	})

	ctx := context.Background()
	firstCh := make(chan Session, 1)
	errCh := make(chan error, 1)
	go func() {
		session, err := provider.Create(ctx, CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
		if err != nil {
			errCh <- err
			return
		}
		firstCh <- session
	}()

	// Wait until the first Create has acquired the only budget slot and is
	// blocked inside the inner provider.
	deadline := time.Now().Add(time.Second)
	for atomic.LoadInt32(&inner.inFlight) < 1 {
		if time.Now().After(deadline) {
			t.Fatal("first create never entered inner provider")
		}
		time.Sleep(5 * time.Millisecond)
	}

	start := time.Now()
	_, err := provider.Create(ctx, CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
	if !errors.Is(err, ErrCapacityTimeout) {
		t.Fatalf("error = %v, want ErrCapacityTimeout", err)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Fatalf("timed out too quickly: %s", time.Since(start))
	}

	close(inner.block)
	select {
	case err := <-errCh:
		t.Fatalf("first create: %v", err)
	case first := <-firstCh:
		_ = first.Destroy(ctx)
	case <-time.After(time.Second):
		t.Fatal("first create did not finish")
	}
}

func TestCapacityProvider_UnlimitedPassthrough(t *testing.T) {
	inner := &FakeProvider{}
	got := WrapCapacity(inner, CapacityConfig{MaxConcurrent: 0})
	if got != inner {
		t.Fatalf("WrapCapacity with MaxConcurrent=0 should return inner unchanged")
	}
}

func TestCapacityProvider_ReleasesOnCreateError(t *testing.T) {
	inner := &trackingProvider{createErr: errors.New("boom")}
	provider := WrapCapacity(inner, CapacityConfig{
		MaxConcurrent:  1,
		AcquireTimeout: time.Second,
	})
	_, err := provider.Create(context.Background(), CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("error = %v, want boom", err)
	}
	// Slot must be free for the next acquire.
	inner.createErr = nil
	session, err := provider.Create(context.Background(), CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
	if err != nil {
		t.Fatalf("second create after release: %v", err)
	}
	_ = session.Destroy(context.Background())
}

func TestCapacityProvider_DoesNotReleaseOnDestroyError(t *testing.T) {
	inner := &FakeProvider{}
	provider := WrapCapacity(inner, CapacityConfig{
		MaxConcurrent:  1,
		AcquireTimeout: time.Second,
	})

	session, err := provider.Create(context.Background(), CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	underlying := session.(*leasedSession).Session.(*FakeSession)
	underlying.SetDestroyError(errors.New("destroy failed"))

	if err := session.Destroy(context.Background()); err == nil {
		t.Fatal("expected destroy error")
	}

	done := make(chan error, 1)
	go func() {
		_, err := provider.Create(context.Background(), CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
		done <- err
	}()

	time.Sleep(80 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("second create should wait for slot, got err=%v", err)
	default:
	}

	underlying.SetDestroyError(nil)
	if err := session.Destroy(context.Background()); err != nil {
		t.Fatalf("retry destroy: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second create after successful destroy: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("second create did not proceed after destroy retry")
	}
}

func TestCapacityProvider_ReleasesOnSandboxNotFound(t *testing.T) {
	inner := &FakeProvider{}
	provider := WrapCapacity(inner, CapacityConfig{
		MaxConcurrent:  1,
		AcquireTimeout: time.Second,
	})

	session, err := provider.Create(context.Background(), CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	underlying := session.(*leasedSession).Session.(*FakeSession)
	underlying.SetDestroyError(ErrSandboxNotFound)

	if err := session.Destroy(context.Background()); !errors.Is(err, ErrSandboxNotFound) {
		t.Fatalf("destroy err = %v, want ErrSandboxNotFound", err)
	}

	session2, err := provider.Create(context.Background(), CreateRequest{RunID: uuid.New(), RunAgentID: uuid.New()})
	if err != nil {
		t.Fatalf("slot should be released after ErrSandboxNotFound: %v", err)
	}
	_ = session2.Destroy(context.Background())
}

func TestCapacityRetryAfter(t *testing.T) {
	err := NewCapacityTimeoutError(3 * time.Second)
	delay, ok := CapacityRetryAfter(err)
	if !ok || delay != 3*time.Second {
		t.Fatalf("CapacityRetryAfter = %v/%v", delay, ok)
	}
	if !IsCapacityError(err) {
		t.Fatal("expected IsCapacityError")
	}
}
