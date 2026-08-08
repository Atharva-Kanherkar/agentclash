package throttle

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type localBucket struct {
	mu            sync.Mutex
	concurrent    chan struct{}
	rpm           *rate.Limiter
	tpmLimit      int64
	tpmUsed       int64
	tpmWindowStart time.Time
	cooldownUntil time.Time
	limits        Limits
}

// LocalLimiter is an in-process fallback when Redis is unavailable.
type LocalLimiter struct {
	cfg     Config
	mu      sync.Mutex
	buckets map[string]*localBucket
}

// NewLocalLimiter constructs an in-process limiter.
func NewLocalLimiter(cfg Config) *LocalLimiter {
	return &LocalLimiter{
		cfg:     cfg,
		buckets: make(map[string]*localBucket),
	}
}

func (l *LocalLimiter) bucket(key Key) *localBucket {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := key.String()
	if b, ok := l.buckets[id]; ok {
		return b
	}
	limits := l.cfg.limitsFor(key.Provider)
	b := &localBucket{limits: limits, tpmLimit: limits.TPM}
	if limits.MaxConcurrent > 0 {
		b.concurrent = make(chan struct{}, limits.MaxConcurrent)
		for i := 0; i < limits.MaxConcurrent; i++ {
			b.concurrent <- struct{}{}
		}
	}
	if limits.RPM > 0 {
		perSec := rate.Limit(float64(limits.RPM) / 60.0)
		burst := limits.RPM
		if burst < 1 {
			burst = 1
		}
		b.rpm = rate.NewLimiter(perSec, burst)
	}
	l.buckets[id] = b
	return b
}

func (l *LocalLimiter) CoolDown(key Key, d time.Duration) {
	if d <= 0 {
		return
	}
	b := l.bucket(key)
	b.mu.Lock()
	until := time.Now().Add(d)
	if until.After(b.cooldownUntil) {
		b.cooldownUntil = until
	}
	b.mu.Unlock()
}

func (l *LocalLimiter) Acquire(ctx context.Context, key Key, estimatedTokens int64) (Lease, error) {
	limits := l.cfg.limitsFor(key.Provider)
	if !limits.Enabled() {
		return noopLease{}, nil
	}
	b := l.bucket(key)
	ctx, cancel := context.WithTimeout(ctx, l.cfg.acquireTimeout())
	defer cancel()

	for {
		b.mu.Lock()
		cooldown := b.cooldownUntil
		b.mu.Unlock()
		if wait := time.Until(cooldown); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ErrAcquireTimeout
			case <-timer.C:
			}
			continue
		}

		if b.concurrent != nil {
			select {
			case <-b.concurrent:
			case <-ctx.Done():
				return nil, ErrAcquireTimeout
			}
		}

		if b.rpm != nil {
			if err := b.rpm.Wait(ctx); err != nil {
				if b.concurrent != nil {
					b.concurrent <- struct{}{}
				}
				return nil, ErrAcquireTimeout
			}
		}

		reserved := estimatedTokens
		if reserved < 0 {
			reserved = 0
		}
		if b.tpmLimit > 0 && reserved > 0 {
			if !b.reserveTPM(reserved) {
				if b.concurrent != nil {
					b.concurrent <- struct{}{}
				}
				timer := time.NewTimer(20 * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ErrAcquireTimeout
				case <-timer.C:
				}
				continue
			}
		}

		return &localLease{bucket: b, reserved: reserved}, nil
	}
}

func (b *localBucket) reserveTPM(tokens int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if b.tpmWindowStart.IsZero() || now.Sub(b.tpmWindowStart) >= time.Minute {
		b.tpmWindowStart = now
		b.tpmUsed = 0
	}
	if b.tpmUsed+tokens > b.tpmLimit {
		return false
	}
	b.tpmUsed += tokens
	return true
}

func (b *localBucket) returnTPM(tokens int64) {
	if tokens <= 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tpmUsed -= tokens
	if b.tpmUsed < 0 {
		b.tpmUsed = 0
	}
}

type localLease struct {
	bucket   *localBucket
	reserved int64
	once     sync.Once
}

func (l *localLease) Reconcile(actualTokens int64) {
	if l.bucket.tpmLimit <= 0 {
		return
	}
	if actualTokens < 0 {
		actualTokens = 0
	}
	surplus := l.reserved - actualTokens
	if surplus > 0 {
		l.bucket.returnTPM(surplus)
	}
	l.reserved = 0
}

func (l *localLease) Release() {
	l.once.Do(func() {
		if l.bucket.tpmLimit > 0 && l.reserved > 0 {
			l.bucket.returnTPM(l.reserved)
			l.reserved = 0
		}
		if l.bucket.concurrent != nil {
			l.bucket.concurrent <- struct{}{}
		}
	})
}

type noopLease struct{}

func (noopLease) Reconcile(int64) {}
func (noopLease) Release()        {}
