// Package throttle provides distributed outbound LLM rate limiting keyed by
// (provider, credential-ref). Defaults are off (zero limits = unlimited).
package throttle

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Key identifies a provider account budget.
type Key struct {
	Provider   string
	Credential string
}

func (k Key) String() string {
	provider := strings.ToLower(strings.TrimSpace(k.Provider))
	cred := strings.TrimSpace(k.Credential)
	if cred == "" {
		cred = "_"
	}
	return provider + ":" + cred
}

// Limits are per-key budgets. Zero means unlimited for that dimension.
type Limits struct {
	MaxConcurrent int
	RPM           int
	TPM           int64
}

// Enabled reports whether any dimension is capped.
func (l Limits) Enabled() bool {
	return l.MaxConcurrent > 0 || l.RPM > 0 || l.TPM > 0
}

// Config configures the throttle layer.
type Config struct {
	// LimitsByProvider maps provider key (e.g. "openai") to limits.
	LimitsByProvider map[string]Limits
	// AcquireTimeout bounds waiting for a slot (default 2m).
	AcquireTimeout time.Duration
	// EstimateTokens estimates pre-call TPM reservation; optional.
	EstimateTokens func(messagesChars int) int64
}

func (c Config) acquireTimeout() time.Duration {
	if c.AcquireTimeout <= 0 {
		return 2 * time.Minute
	}
	return c.AcquireTimeout
}

func (c Config) limitsFor(provider string) Limits {
	if c.LimitsByProvider == nil {
		return Limits{}
	}
	return c.LimitsByProvider[strings.ToLower(strings.TrimSpace(provider))]
}

func (c Config) estimate(chars int) int64 {
	if c.EstimateTokens != nil {
		return c.EstimateTokens(chars)
	}
	// ~4 chars/token heuristic; floor 1 when there is any content.
	if chars <= 0 {
		return 1
	}
	est := int64(chars / 4)
	if est < 1 {
		est = 1
	}
	return est
}

// Lease is held for the duration of one provider call.
type Lease interface {
	// Reconcile adjusts the TPM reservation using actual usage (may return surplus).
	Reconcile(actualTokens int64)
	// Release frees concurrency + any remaining TPM reservation.
	Release()
}

// Limiter is the shared outbound budget.
type Limiter interface {
	// Acquire waits for capacity. estimatedTokens is reserved against TPM.
	Acquire(ctx context.Context, key Key, estimatedTokens int64) (Lease, error)
	// CoolDown blocks new acquisitions for the key until now+d (from Retry-After).
	CoolDown(key Key, d time.Duration)
}

// ErrAcquireTimeout is returned when Acquire exceeds the wait budget.
var ErrAcquireTimeout = fmt.Errorf("provider throttle acquire timed out")
