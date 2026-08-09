package observability

import "context"

// SSEGate adapts Fleet SSE gauges to the api.SSEConnectionGate interface.
type SSEGate struct {
	fleet *Fleet
	max   int
}

func NewSSEGate(fleet *Fleet, max int) *SSEGate {
	if fleet == nil {
		fleet = NewFleet(nil)
	}
	return &SSEGate{fleet: fleet, max: max}
}

func (g *SSEGate) TryAcquire(ctx context.Context) bool {
	return g.fleet.AcquireSSE(ctx, g.max)
}

func (g *SSEGate) Release(ctx context.Context) {
	g.fleet.ReleaseSSE(ctx)
}
