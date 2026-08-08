package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// WarmPoolConfig configures an optional per-worker warm sandbox pool.
//
// v1 is intentionally per-process: replicas do not share warm sandboxes.
// Cross-replica sharing belongs with a future control-plane pool if needed.
type WarmPoolConfig struct {
	// Size is the target number of warm sandboxes per pool key. 0 disables.
	Size int
	// TTL is how long an unused warm sandbox may idle before Destroy. Default 10m.
	TTL time.Duration
	// FillTimeout bounds each background Create used to replenish the pool.
	FillTimeout time.Duration
	// Metrics receives pool hit/miss/fill/expire counters (optional).
	Metrics Metrics
	// Logger defaults to slog.Default when nil.
	Logger *slog.Logger
	// Clock overrides time.Now for tests.
	Clock func() time.Time
}

// WarmPool is a Provider decorator that checks out pre-created sessions keyed by
// (TemplateID, ToolPolicy hash) before falling through to the inner provider.
type WarmPool struct {
	inner       Provider
	size        int
	ttl         time.Duration
	fillTimeout time.Duration
	metrics     Metrics
	logger      *slog.Logger
	clock       func() time.Time

	mu      sync.Mutex
	pools   map[string][]*warmEntry
	stopCh  chan struct{}
	stopped bool
	wg      sync.WaitGroup
}

type warmEntry struct {
	session   Session
	request   CreateRequest
	createdAt time.Time
}

// WrapWarmPool returns inner unchanged when Size <= 0.
func WrapWarmPool(inner Provider, cfg WarmPoolConfig) *WarmPool {
	if inner == nil {
		inner = UnconfiguredProvider{}
	}
	if cfg.Size <= 0 {
		return nil
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	fillTimeout := cfg.FillTimeout
	if fillTimeout <= 0 {
		fillTimeout = 2 * time.Minute
	}
	metrics := cfg.Metrics
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	return &WarmPool{
		inner:       inner,
		size:        cfg.Size,
		ttl:         ttl,
		fillTimeout: fillTimeout,
		metrics:     metrics,
		logger:      logger,
		clock:       clock,
		pools:       make(map[string][]*warmEntry),
		stopCh:      make(chan struct{}),
	}
}

// Start begins the idle-expiry loop. Safe to call once.
func (p *WarmPool) Start() {
	if p == nil {
		return
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-p.stopCh:
				return
			case <-ticker.C:
				p.expireIdle()
			}
		}
	}()
}

// Close stops the filler/expiry loops and destroys remaining warm sessions.
func (p *WarmPool) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return nil
	}
	p.stopped = true
	close(p.stopCh)
	entries := make([]*warmEntry, 0)
	for key, list := range p.pools {
		entries = append(entries, list...)
		delete(p.pools, key)
	}
	p.mu.Unlock()
	p.wg.Wait()

	var firstErr error
	for _, entry := range entries {
		if err := entry.session.Destroy(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		p.metrics.WarmPoolExpire()
	}
	return firstErr
}

func (p *WarmPool) Create(ctx context.Context, request CreateRequest) (Session, error) {
	if p == nil {
		return nil, ErrProviderNotConfigured
	}
	key := PoolKey(request.TemplateID, request.ToolPolicy)
	if session, ok := p.checkout(key); ok {
		p.metrics.WarmPoolHit()
		p.scheduleFill(key, request)
		return session, nil
	}
	p.metrics.WarmPoolMiss()
	session, err := p.inner.Create(ctx, request)
	if err != nil {
		return nil, err
	}
	p.scheduleFill(key, request)
	return session, nil
}

// EnsureWarm pre-fills the pool for request's key up to Size (best-effort).
func (p *WarmPool) EnsureWarm(ctx context.Context, request CreateRequest) {
	if p == nil {
		return
	}
	key := PoolKey(request.TemplateID, request.ToolPolicy)
	for p.poolLen(key) < p.size {
		if err := ctx.Err(); err != nil {
			return
		}
		if !p.fillOne(ctx, key, request) {
			return
		}
	}
}

func (p *WarmPool) checkout(key string) (Session, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	list := p.pools[key]
	for len(list) > 0 {
		entry := list[0]
		list = list[1:]
		p.pools[key] = list
		if p.clock().Sub(entry.createdAt) > p.ttl {
			go p.destroyQuiet(entry.session)
			p.metrics.WarmPoolExpire()
			continue
		}
		return entry.session, true
	}
	return nil, false
}

func (p *WarmPool) scheduleFill(key string, request CreateRequest) {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), p.fillTimeout)
		defer cancel()
		for p.poolLen(key) < p.size {
			select {
			case <-p.stopCh:
				return
			default:
			}
			if !p.fillOne(ctx, key, request) {
				return
			}
		}
	}()
}

func (p *WarmPool) fillOne(ctx context.Context, key string, request CreateRequest) bool {
	session, err := p.inner.Create(ctx, cloneCreateRequest(request))
	if err != nil {
		p.logger.Warn("warm pool fill failed", "pool_key", key, "error", err)
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		go p.destroyQuiet(session)
		return false
	}
	if len(p.pools[key]) >= p.size {
		go p.destroyQuiet(session)
		return false
	}
	p.pools[key] = append(p.pools[key], &warmEntry{
		session:   session,
		request:   cloneCreateRequest(request),
		createdAt: p.clock(),
	})
	p.metrics.WarmPoolFill()
	return true
}

func (p *WarmPool) poolLen(key string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pools[key])
}

func (p *WarmPool) expireIdle() {
	now := p.clock()
	var expired []Session
	p.mu.Lock()
	for key, list := range p.pools {
		kept := list[:0]
		for _, entry := range list {
			if now.Sub(entry.createdAt) > p.ttl {
				expired = append(expired, entry.session)
				continue
			}
			kept = append(kept, entry)
		}
		if len(kept) == 0 {
			delete(p.pools, key)
		} else {
			p.pools[key] = kept
		}
	}
	p.mu.Unlock()
	for _, session := range expired {
		p.destroyQuiet(session)
		p.metrics.WarmPoolExpire()
	}
}

func (p *WarmPool) destroyQuiet(session Session) {
	if session == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = session.Destroy(ctx)
}

// PoolKey returns a stable key for (templateID, tool policy).
func PoolKey(templateID string, policy ToolPolicy) string {
	payload, err := json.Marshal(policy)
	if err != nil {
		payload = []byte("{}")
	}
	sum := sha256.Sum256(payload)
	return templateID + ":" + hex.EncodeToString(sum[:8])
}
