package sandbox

import (
	"sync/atomic"
	"time"
)

// Metrics is a Fleet 14-facing hook for sandbox capacity / warm-pool counters.
// Implementations may be no-ops until OTel wiring lands.
type Metrics interface {
	CapacityAcquireAttempt()
	CapacityAcquired()
	CapacityWait(d time.Duration)
	CapacityTimeout()
	CapacityReleased()
	WarmPoolHit()
	WarmPoolMiss()
	WarmPoolFill()
	WarmPoolExpire()
}

// NoopMetrics discards all metric events.
type NoopMetrics struct{}

func (NoopMetrics) CapacityAcquireAttempt()    {}
func (NoopMetrics) CapacityAcquired()          {}
func (NoopMetrics) CapacityWait(time.Duration) {}
func (NoopMetrics) CapacityTimeout()           {}
func (NoopMetrics) CapacityReleased()          {}
func (NoopMetrics) WarmPoolHit()               {}
func (NoopMetrics) WarmPoolMiss()              {}
func (NoopMetrics) WarmPoolFill()              {}
func (NoopMetrics) WarmPoolExpire()            {}

// RecordingMetrics stores counters for tests (goroutine-safe).
type RecordingMetrics struct {
	AcquireAttempts atomic.Int64
	Acquired        atomic.Int64
	Waits           atomic.Int64
	Timeouts        atomic.Int64
	Released        atomic.Int64
	PoolHits        atomic.Int64
	PoolMisses      atomic.Int64
	PoolFills       atomic.Int64
	PoolExpires     atomic.Int64
}

func (m *RecordingMetrics) CapacityAcquireAttempt() { m.AcquireAttempts.Add(1) }
func (m *RecordingMetrics) CapacityAcquired()       { m.Acquired.Add(1) }
func (m *RecordingMetrics) CapacityWait(time.Duration) {
	m.Waits.Add(1)
}
func (m *RecordingMetrics) CapacityTimeout()  { m.Timeouts.Add(1) }
func (m *RecordingMetrics) CapacityReleased() { m.Released.Add(1) }
func (m *RecordingMetrics) WarmPoolHit()      { m.PoolHits.Add(1) }
func (m *RecordingMetrics) WarmPoolMiss()     { m.PoolMisses.Add(1) }
func (m *RecordingMetrics) WarmPoolFill()     { m.PoolFills.Add(1) }
func (m *RecordingMetrics) WarmPoolExpire()   { m.PoolExpires.Add(1) }
