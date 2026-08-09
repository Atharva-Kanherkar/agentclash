package observability

import (
	"sync"
	"testing"

	"go.opentelemetry.io/otel/metric/noop"
)

func TestTemporalMetricsHandlerWithTagsSharesCacheMutex(t *testing.T) {
	handler := NewTemporalMetricsHandler(noop.Meter{})
	a := handler.WithTags(map[string]string{"worker_type": "activity"})
	b := handler.WithTags(map[string]string{"worker_type": "workflow"})

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			a.Counter("temporal_worker_task_slots_available").Inc(1)
			a.Gauge("fleet_fanout_inflight").Update(1)
			a.Timer("temporal_workflow_task_schedule_to_start_latency").Record(0)
		}()
		go func() {
			defer wg.Done()
			b.Counter("temporal_worker_task_slots_available").Inc(1)
			b.Gauge("fleet_fanout_cap").Update(2)
			b.Timer("temporal_workflow_task_schedule_to_start_latency").Record(0)
		}()
	}
	wg.Wait()
}
