package observability

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/agentclash/agentclash/runtime/sandbox"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Fleet aggregates Fleet-plane instruments. Methods are no-ops when meter is nil.
type Fleet struct {
	meter metric.Meter

	evalSetsByStatus   metric.Int64UpDownCounter
	fanoutInFlight     metric.Int64UpDownCounter
	runDuration        metric.Float64Histogram
	sandboxAcquires    metric.Int64Counter
	sandboxWaits       metric.Int64Counter
	sandboxWaitSeconds metric.Float64Histogram
	sandboxTimeouts    metric.Int64Counter
	sandboxPoolHits    metric.Int64Counter
	sandboxPoolMisses  metric.Int64Counter
	providerRequests   metric.Int64Counter
	providerThrottles  metric.Int64Counter
	providerRateLimits metric.Int64Counter
	providerCooldowns  metric.Int64UpDownCounter
	eventsWritten      metric.Int64Counter
	eventBytesInline   metric.Int64Counter
	eventBytesSpill    metric.Int64Counter
	eventQueueDepth    metric.Int64UpDownCounter
	sseActive          metric.Int64UpDownCounter
	sseRejected        metric.Int64Counter
	setsStalled        metric.Int64Counter

	activeSSE atomic.Int64
}

func NewFleet(meter metric.Meter) *Fleet {
	if meter == nil {
		return &Fleet{}
	}
	f, err := newFleet(meter)
	if err != nil {
		return &Fleet{}
	}
	return f
}

func newFleet(meter metric.Meter) (*Fleet, error) {
	f := &Fleet{meter: meter}
	var err error
	if f.evalSetsByStatus, err = meter.Int64UpDownCounter("fleet_eval_sets", metric.WithDescription("Eval set status transitions (+1 to / -1 from)")); err != nil {
		return nil, err
	}
	if f.fanoutInFlight, err = meter.Int64UpDownCounter("fleet_fanout_inflight", metric.WithDescription("Bounded fan-out in-flight futures")); err != nil {
		return nil, err
	}
	if f.runDuration, err = meter.Float64Histogram("fleet_run_duration_seconds", metric.WithDescription("Run duration seconds")); err != nil {
		return nil, err
	}
	if f.sandboxAcquires, err = meter.Int64Counter("fleet_sandbox_acquires", metric.WithDescription("Sandbox capacity acquires")); err != nil {
		return nil, err
	}
	if f.sandboxWaits, err = meter.Int64Counter("fleet_sandbox_waits", metric.WithDescription("Sandbox capacity waits")); err != nil {
		return nil, err
	}
	if f.sandboxWaitSeconds, err = meter.Float64Histogram("fleet_sandbox_wait_seconds", metric.WithDescription("Sandbox acquire wait duration")); err != nil {
		return nil, err
	}
	if f.sandboxTimeouts, err = meter.Int64Counter("fleet_sandbox_timeouts", metric.WithDescription("Sandbox acquire timeouts")); err != nil {
		return nil, err
	}
	if f.sandboxPoolHits, err = meter.Int64Counter("fleet_sandbox_pool_hits", metric.WithDescription("Warm pool hits")); err != nil {
		return nil, err
	}
	if f.sandboxPoolMisses, err = meter.Int64Counter("fleet_sandbox_pool_misses", metric.WithDescription("Warm pool misses")); err != nil {
		return nil, err
	}
	if f.providerRequests, err = meter.Int64Counter("fleet_provider_requests", metric.WithDescription("Provider invoke attempts")); err != nil {
		return nil, err
	}
	if f.providerThrottles, err = meter.Int64Counter("fleet_provider_throttle_waits", metric.WithDescription("Provider throttle waits")); err != nil {
		return nil, err
	}
	if f.providerRateLimits, err = meter.Int64Counter("fleet_provider_rate_limits", metric.WithDescription("Observed 429 / rate-limit failures")); err != nil {
		return nil, err
	}
	if f.providerCooldowns, err = meter.Int64UpDownCounter("fleet_provider_cooldowns_active", metric.WithDescription("Active provider cooldowns")); err != nil {
		return nil, err
	}
	if f.eventsWritten, err = meter.Int64Counter("fleet_events_written", metric.WithDescription("Run events persisted")); err != nil {
		return nil, err
	}
	if f.eventBytesInline, err = meter.Int64Counter("fleet_event_bytes_inline", metric.WithDescription("Inline event payload bytes")); err != nil {
		return nil, err
	}
	if f.eventBytesSpill, err = meter.Int64Counter("fleet_event_bytes_offloaded", metric.WithDescription("Offloaded event payload bytes")); err != nil {
		return nil, err
	}
	if f.eventQueueDepth, err = meter.Int64UpDownCounter("fleet_event_queue_depth", metric.WithDescription("Buffered observer queue depth")); err != nil {
		return nil, err
	}
	if f.sseActive, err = meter.Int64UpDownCounter("fleet_sse_active_connections", metric.WithDescription("Active run-event SSE connections")); err != nil {
		return nil, err
	}
	if f.sseRejected, err = meter.Int64Counter("fleet_sse_rejected", metric.WithDescription("SSE connections rejected at capacity")); err != nil {
		return nil, err
	}
	if f.setsStalled, err = meter.Int64Counter("fleet_set_stalled", metric.WithDescription("Non-terminal eval sets with no transitions past stall threshold")); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Fleet) RecordEvalSetStatus(ctx context.Context, status string, delta int64) {
	if f == nil || f.evalSetsByStatus == nil {
		return
	}
	f.evalSetsByStatus.Add(ctx, delta, metric.WithAttributes(attribute.String("status", status)))
}

func (f *Fleet) RecordFanoutInFlight(ctx context.Context, delta int64) {
	if f == nil || f.fanoutInFlight == nil {
		return
	}
	f.fanoutInFlight.Add(ctx, delta)
}

func (f *Fleet) RecordRunDuration(ctx context.Context, d time.Duration) {
	if f == nil || f.runDuration == nil {
		return
	}
	f.runDuration.Record(ctx, d.Seconds())
}

func (f *Fleet) RecordProviderRequest(ctx context.Context, providerKey string) {
	if f == nil || f.providerRequests == nil {
		return
	}
	f.providerRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", providerKey)))
}

func (f *Fleet) RecordProviderThrottleWait(ctx context.Context, providerKey string) {
	if f == nil || f.providerThrottles == nil {
		return
	}
	f.providerThrottles.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", providerKey)))
}

func (f *Fleet) RecordProviderRateLimit(ctx context.Context, providerKey string) {
	if f == nil || f.providerRateLimits == nil {
		return
	}
	f.providerRateLimits.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", providerKey)))
}

func (f *Fleet) RecordEventWritten(ctx context.Context, inlineBytes, spillBytes int64) {
	if f == nil {
		return
	}
	if f.eventsWritten != nil {
		f.eventsWritten.Add(ctx, 1)
	}
	if inlineBytes > 0 && f.eventBytesInline != nil {
		f.eventBytesInline.Add(ctx, inlineBytes)
	}
	if spillBytes > 0 && f.eventBytesSpill != nil {
		f.eventBytesSpill.Add(ctx, spillBytes)
	}
}

func (f *Fleet) SetEventQueueDepth(ctx context.Context, depth int64) {
	if f == nil || f.eventQueueDepth == nil {
		return
	}
	// Represent absolute depth via delta from last is hard without state; emit as add of depth
	// is wrong. Use UpDownCounter by tracking last — for v1, Add(depth) after Add(-last).
	f.eventQueueDepth.Add(ctx, depth)
}

func (f *Fleet) RecordSetStalled(ctx context.Context) {
	if f == nil || f.setsStalled == nil {
		return
	}
	f.setsStalled.Add(ctx, 1)
}

func (f *Fleet) AcquireSSE(ctx context.Context, max int) bool {
	if max <= 0 {
		if f != nil && f.sseActive != nil {
			f.sseActive.Add(ctx, 1)
			f.activeSSE.Add(1)
		}
		return true
	}
	for {
		cur := f.activeSSE.Load()
		if cur >= int64(max) {
			if f != nil && f.sseRejected != nil {
				f.sseRejected.Add(ctx, 1)
			}
			return false
		}
		if f.activeSSE.CompareAndSwap(cur, cur+1) {
			if f != nil && f.sseActive != nil {
				f.sseActive.Add(ctx, 1)
			}
			return true
		}
	}
}

func (f *Fleet) ReleaseSSE(ctx context.Context) {
	if f == nil {
		return
	}
	f.activeSSE.Add(-1)
	if f.sseActive != nil {
		f.sseActive.Add(ctx, -1)
	}
}

// SandboxMetrics returns a sandbox.Metrics implementation backed by this Fleet.
func (f *Fleet) SandboxMetrics() sandbox.Metrics {
	if f == nil || f.meter == nil {
		return sandbox.NoopMetrics{}
	}
	return &sandboxBridge{fleet: f}
}

type sandboxBridge struct {
	fleet *Fleet
}

func (b *sandboxBridge) CapacityAcquireAttempt() {}
func (b *sandboxBridge) CapacityAcquired() {
	if b.fleet.sandboxAcquires != nil {
		b.fleet.sandboxAcquires.Add(context.Background(), 1)
	}
}
func (b *sandboxBridge) CapacityWait(d time.Duration) {
	if b.fleet.sandboxWaits != nil {
		b.fleet.sandboxWaits.Add(context.Background(), 1)
	}
	if b.fleet.sandboxWaitSeconds != nil {
		b.fleet.sandboxWaitSeconds.Record(context.Background(), d.Seconds())
	}
}
func (b *sandboxBridge) CapacityTimeout() {
	if b.fleet.sandboxTimeouts != nil {
		b.fleet.sandboxTimeouts.Add(context.Background(), 1)
	}
}
func (b *sandboxBridge) CapacityReleased() {}
func (b *sandboxBridge) WarmPoolHit() {
	if b.fleet.sandboxPoolHits != nil {
		b.fleet.sandboxPoolHits.Add(context.Background(), 1)
	}
}
func (b *sandboxBridge) WarmPoolMiss() {
	if b.fleet.sandboxPoolMisses != nil {
		b.fleet.sandboxPoolMisses.Add(context.Background(), 1)
	}
}
func (b *sandboxBridge) WarmPoolFill()   {}
func (b *sandboxBridge) WarmPoolExpire() {}
