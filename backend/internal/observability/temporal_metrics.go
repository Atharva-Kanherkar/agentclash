package observability

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	temporalsdk "go.temporal.io/sdk/client"
)

// NewTemporalMetricsHandler bridges Temporal's MetricsHandler to an OTel meter
// so worker slots / task latency share the Prometheus scrape endpoint.
func NewTemporalMetricsHandler(meter metric.Meter) temporalsdk.MetricsHandler {
	if meter == nil {
		return temporalsdk.MetricsNopHandler
	}
	return &otelMetricsHandler{meter: meter}
}

type otelMetricsHandler struct {
	meter metric.Meter
	tags  map[string]string

	mu       sync.Mutex
	counters map[string]metric.Int64Counter
	gauges   map[string]metric.Float64Gauge
	timers   map[string]metric.Float64Histogram
}

func (h *otelMetricsHandler) WithTags(tags map[string]string) temporalsdk.MetricsHandler {
	merged := make(map[string]string, len(h.tags)+len(tags))
	for k, v := range h.tags {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	return &otelMetricsHandler{
		meter:    h.meter,
		tags:     merged,
		counters: h.counters,
		gauges:   h.gauges,
		timers:   h.timers,
	}
}

func (h *otelMetricsHandler) attrs() []attribute.KeyValue {
	if len(h.tags) == 0 {
		return nil
	}
	out := make([]attribute.KeyValue, 0, len(h.tags))
	for k, v := range h.tags {
		out = append(out, attribute.String(k, v))
	}
	return out
}

func (h *otelMetricsHandler) Counter(name string) temporalsdk.MetricsCounter {
	return otelCounter{c: h.getCounter(name), attrs: h.attrs()}
}

func (h *otelMetricsHandler) Gauge(name string) temporalsdk.MetricsGauge {
	return otelGauge{g: h.getGauge(name), attrs: h.attrs()}
}

func (h *otelMetricsHandler) Timer(name string) temporalsdk.MetricsTimer {
	return otelTimer{h: h.getTimer(name), attrs: h.attrs()}
}

func (h *otelMetricsHandler) getCounter(name string) metric.Int64Counter {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.counters == nil {
		h.counters = map[string]metric.Int64Counter{}
	}
	if c, ok := h.counters[name]; ok {
		return c
	}
	c, err := h.meter.Int64Counter("temporal_" + sanitizeMetricName(name))
	if err != nil {
		return nil
	}
	h.counters[name] = c
	return c
}

func (h *otelMetricsHandler) getGauge(name string) metric.Float64Gauge {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.gauges == nil {
		h.gauges = map[string]metric.Float64Gauge{}
	}
	if g, ok := h.gauges[name]; ok {
		return g
	}
	g, err := h.meter.Float64Gauge("temporal_" + sanitizeMetricName(name))
	if err != nil {
		return nil
	}
	h.gauges[name] = g
	return g
}

func (h *otelMetricsHandler) getTimer(name string) metric.Float64Histogram {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.timers == nil {
		h.timers = map[string]metric.Float64Histogram{}
	}
	if t, ok := h.timers[name]; ok {
		return t
	}
	t, err := h.meter.Float64Histogram("temporal_" + sanitizeMetricName(name) + "_seconds")
	if err != nil {
		return nil
	}
	h.timers[name] = t
	return t
}

type otelCounter struct {
	c     metric.Int64Counter
	attrs []attribute.KeyValue
}

func (c otelCounter) Inc(n int64) {
	if c.c == nil {
		return
	}
	c.c.Add(context.Background(), n, metric.WithAttributes(c.attrs...))
}

type otelGauge struct {
	g     metric.Float64Gauge
	attrs []attribute.KeyValue
}

func (g otelGauge) Update(v float64) {
	if g.g == nil {
		return
	}
	g.g.Record(context.Background(), v, metric.WithAttributes(g.attrs...))
}

type otelTimer struct {
	h     metric.Float64Histogram
	attrs []attribute.KeyValue
}

func (t otelTimer) Record(d time.Duration) {
	if t.h == nil {
		return
	}
	t.h.Record(context.Background(), d.Seconds(), metric.WithAttributes(t.attrs...))
}

func sanitizeMetricName(name string) string {
	b := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}
