package observability

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config controls the optional Prometheus scrape endpoint (Fleet 14).
// Default-off: Enabled=false leaves process behavior unchanged.
type Config struct {
	Enabled bool
	// Addr is the listen address for the scrape server (e.g. ":9464").
	Addr string
	// StallThreshold is how long a non-terminal eval set may sit without
	// updated_at movement before fleet.set.stalled is emitted.
	StallThreshold time.Duration
	// StallInterval is the reaper poll interval.
	StallInterval time.Duration
	// SSEMaxConnections caps concurrent run-event SSE streams (0 = unlimited).
	SSEMaxConnections int
}

const (
	defaultMetricsAddr     = ":9464"
	defaultStallThreshold  = 30 * time.Minute
	defaultStallInterval   = 5 * time.Minute
	defaultSSEMaxConns     = 0
)

// LoadConfigFromEnv reads METRICS_ENABLED / METRICS_ADDR / FLEET_STALL_* / SSE_MAX_CONNECTIONS.
func LoadConfigFromEnv() Config {
	cfg := Config{
		Enabled:           envBool("METRICS_ENABLED", false),
		Addr:              envOr("METRICS_ADDR", defaultMetricsAddr),
		StallThreshold:    envDuration("FLEET_STALL_THRESHOLD", defaultStallThreshold),
		StallInterval:     envDuration("FLEET_STALL_INTERVAL", defaultStallInterval),
		SSEMaxConnections: envInt("SSE_MAX_CONNECTIONS", defaultSSEMaxConns),
	}
	return cfg
}

func envBool(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return v
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
}

func envInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}
