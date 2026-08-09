package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Capacity / account-limit errors. Activities map these to a retryable Temporal
// failure class with NextRetryDelay (see backend wrapActivityError).
var (
	ErrCapacityTimeout = errors.New("sandbox capacity acquire timed out")
	ErrAccountLimit    = errors.New("sandbox provider account concurrency limit")
)

// CapacityError carries an optional Retry-After style delay for Temporal.
type CapacityError struct {
	Err        error
	RetryAfter time.Duration
}

func (e *CapacityError) Error() string {
	if e == nil || e.Err == nil {
		return ErrCapacityTimeout.Error()
	}
	return e.Err.Error()
}

func (e *CapacityError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewCapacityTimeoutError returns ErrCapacityTimeout with a suggested backoff.
func NewCapacityTimeoutError(retryAfter time.Duration) error {
	if retryAfter <= 0 {
		retryAfter = 5 * time.Second
	}
	return &CapacityError{Err: ErrCapacityTimeout, RetryAfter: retryAfter}
}

// NewAccountLimitError returns ErrAccountLimit with a suggested backoff.
func NewAccountLimitError(retryAfter time.Duration, detail string) error {
	if retryAfter <= 0 {
		retryAfter = 15 * time.Second
	}
	err := ErrAccountLimit
	if detail != "" {
		err = fmt.Errorf("%w: %s", ErrAccountLimit, detail)
	}
	return &CapacityError{Err: err, RetryAfter: retryAfter}
}

// CapacityRetryAfter extracts a Retry-After style delay from err when present.
func CapacityRetryAfter(err error) (time.Duration, bool) {
	var capacityErr *CapacityError
	if errors.As(err, &capacityErr) && capacityErr.RetryAfter > 0 {
		return capacityErr.RetryAfter, true
	}
	return 0, false
}

// IsCapacityError reports whether err is a capacity timeout or account limit.
func IsCapacityError(err error) bool {
	return errors.Is(err, ErrCapacityTimeout) || errors.Is(err, ErrAccountLimit)
}

// Budget is a concurrency semaphore for live sandboxes. Implementations must be
// safe for concurrent use. Release is idempotent when called via sync.Once from
// the capacity session wrapper.
type Budget interface {
	Acquire(ctx context.Context) (release func(), err error)
}

// LocalBudget is an in-process FIFO token pool (buffered channel).
type LocalBudget struct {
	tokens         chan struct{}
	acquireTimeout time.Duration
}

// NewLocalBudget returns a budget with max concurrent tokens. max must be > 0.
func NewLocalBudget(max int, acquireTimeout time.Duration) *LocalBudget {
	if max <= 0 {
		panic("sandbox: LocalBudget max must be > 0")
	}
	if acquireTimeout <= 0 {
		acquireTimeout = 5 * time.Minute
	}
	tokens := make(chan struct{}, max)
	for i := 0; i < max; i++ {
		tokens <- struct{}{}
	}
	return &LocalBudget{tokens: tokens, acquireTimeout: acquireTimeout}
}

func (b *LocalBudget) Acquire(ctx context.Context) (func(), error) {
	ctx, cancel := context.WithTimeout(ctx, b.acquireTimeout)
	defer cancel()

	select {
	case <-b.tokens:
		var once sync.Once
		return func() {
			once.Do(func() {
				b.tokens <- struct{}{}
			})
		}, nil
	case <-ctx.Done():
		return nil, NewCapacityTimeoutError(5 * time.Second)
	}
}

// CapacityConfig configures the capacity Provider decorator.
type CapacityConfig struct {
	// MaxConcurrent bounds live sandboxes. 0 disables the decorator (passthrough).
	MaxConcurrent int
	// AcquireTimeout bounds how long Create waits for a slot. Default 5m.
	AcquireTimeout time.Duration
	// Budget overrides the default LocalBudget (e.g. Redis-backed).
	Budget Budget
	// Metrics receives capacity counters (optional).
	Metrics Metrics
}

// CapacityProvider wraps an inner Provider with a concurrency budget.
type CapacityProvider struct {
	inner   Provider
	budget  Budget
	metrics Metrics
}

// WrapCapacity returns inner unchanged when MaxConcurrent <= 0; otherwise a
// CapacityProvider using Budget or a LocalBudget.
func WrapCapacity(inner Provider, cfg CapacityConfig) Provider {
	if inner == nil {
		return UnconfiguredProvider{}
	}
	if cfg.MaxConcurrent <= 0 {
		return inner
	}
	budget := cfg.Budget
	if budget == nil {
		budget = NewLocalBudget(cfg.MaxConcurrent, cfg.AcquireTimeout)
	}
	metrics := cfg.Metrics
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &CapacityProvider{inner: inner, budget: budget, metrics: metrics}
}

func (p *CapacityProvider) Create(ctx context.Context, request CreateRequest) (Session, error) {
	p.metrics.CapacityAcquireAttempt()
	waitStarted := time.Now()
	release, err := p.budget.Acquire(ctx)
	if err != nil {
		p.metrics.CapacityTimeout()
		return nil, err
	}
	if time.Since(waitStarted) > time.Millisecond {
		p.metrics.CapacityWait(time.Since(waitStarted))
	}
	p.metrics.CapacityAcquired()

	session, err := p.inner.Create(ctx, request)
	if err != nil {
		release()
		p.metrics.CapacityReleased()
		return nil, err
	}
	return &leasedSession{Session: session, release: release, metrics: p.metrics}, nil
}

type leasedSession struct {
	Session
	release func()
	metrics Metrics
	once    sync.Once
}

func (s *leasedSession) Destroy(ctx context.Context) error {
	err := s.Session.Destroy(ctx)
	if err != nil && !errors.Is(err, ErrSandboxNotFound) {
		return err
	}
	s.once.Do(func() {
		if s.release != nil {
			s.release()
		}
		if s.metrics != nil {
			s.metrics.CapacityReleased()
		}
	})
	return err
}

// NewLeaseID returns a unique member id for Redis-backed budgets.
func NewLeaseID() string {
	return uuid.NewString()
}
