package observability

import (
	"context"
	"time"
)

// ThrottleMetrics implements throttle.Metrics using Fleet instruments.
type ThrottleMetrics struct {
	fleet *Fleet
}

func NewThrottleMetrics(fleet *Fleet) *ThrottleMetrics {
	if fleet == nil {
		fleet = NewFleet(nil)
	}
	return &ThrottleMetrics{fleet: fleet}
}

func (m *ThrottleMetrics) Request(providerKey string) {
	m.fleet.RecordProviderRequest(context.Background(), providerKey)
}

func (m *ThrottleMetrics) ThrottleWait(providerKey string) {
	m.fleet.RecordProviderThrottleWait(context.Background(), providerKey)
}

func (m *ThrottleMetrics) RateLimit(providerKey string) {
	m.fleet.RecordProviderRateLimit(context.Background(), providerKey)
}

func (m *ThrottleMetrics) Cooldown(providerKey string, d time.Duration) {
	if d <= 0 {
		return
	}
	m.fleet.RecordProviderCooldown(context.Background(), providerKey, 1)
	time.AfterFunc(d, func() {
		m.fleet.RecordProviderCooldown(context.Background(), providerKey, -1)
	})
}
