package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	temporalsdk "go.temporal.io/sdk/client"
)

// Runtime holds the process-wide meter provider and scrape server.
type Runtime struct {
	cfg      Config
	provider *sdkmetric.MeterProvider
	meter    metric.Meter
	fleet    *Fleet
	server   *http.Server
	listener net.Listener
	logger   *slog.Logger
	wg       sync.WaitGroup
}

// Start initializes OTel Prometheus export when cfg.Enabled. Returns a no-op
// Runtime when disabled so callers can always call Fleet()/Close().
func Start(ctx context.Context, cfg Config, logger *slog.Logger, component string) (*Runtime, error) {
	if logger == nil {
		logger = slog.Default()
	}
	rt := &Runtime{cfg: cfg, logger: logger}
	if !cfg.Enabled {
		rt.fleet = NewFleet(nil)
		return rt, nil
	}

	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("prometheus exporter: %w", err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	meter := provider.Meter("github.com/agentclash/agentclash/" + component)

	fleet, err := newFleet(meter)
	if err != nil {
		_ = provider.Shutdown(ctx)
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	addr := cfg.Addr
	if addr == "" {
		addr = defaultMetricsAddr
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		_ = provider.Shutdown(ctx)
		return nil, fmt.Errorf("metrics listen: %w", err)
	}
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	rt.provider = provider
	rt.meter = meter
	rt.fleet = fleet
	rt.server = server
	rt.listener = ln

	rt.wg.Add(1)
	go func() {
		defer rt.wg.Done()
		logger.Info("metrics scrape server listening", "addr", ln.Addr().String(), "component", component)
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server failed", "error", err)
		}
	}()
	return rt, nil
}

// ScrapeAddr returns the bound metrics address (empty when disabled).
func (r *Runtime) ScrapeAddr() string {
	if r == nil || r.listener == nil {
		return ""
	}
	return r.listener.Addr().String()
}

// Fleet returns instrumentation helpers (never nil).
func (r *Runtime) Fleet() *Fleet {
	if r == nil || r.fleet == nil {
		return NewFleet(nil)
	}
	return r.fleet
}

// Meter returns the OTel meter (nil when metrics disabled).
func (r *Runtime) Meter() metric.Meter {
	if r == nil {
		return nil
	}
	return r.meter
}

// TemporalMetricsHandler returns a Temporal client MetricsHandler wired to the
// same OTel meter, or NopHandler when disabled.
func (r *Runtime) TemporalMetricsHandler() temporalsdk.MetricsHandler {
	if r == nil || !r.cfg.Enabled || r.meter == nil {
		return temporalsdk.MetricsNopHandler
	}
	return NewTemporalMetricsHandler(r.meter)
}

// Config returns the loaded config.
func (r *Runtime) Config() Config {
	if r == nil {
		return Config{}
	}
	return r.cfg
}

// Close shuts down the scrape server and meter provider.
func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var first error
	if r.server != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := r.server.Shutdown(shutdownCtx); err != nil && first == nil {
			first = err
		}
		r.wg.Wait()
	}
	if r.provider != nil {
		if err := r.provider.Shutdown(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}
