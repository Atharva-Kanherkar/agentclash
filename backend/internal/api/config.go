package api

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	defaultBindAddress    = ":8080"
	defaultDatabaseURL    = "postgres://agentclash:agentclash@localhost:5432/agentclash?sslmode=disable"
	defaultTemporalTarget = "localhost:7233"
	defaultNamespace      = "default"
	defaultShutdownTime   = 10 * time.Second
)

var ErrInvalidConfig = errors.New("invalid api server config")

type Config struct {
	BindAddress       string
	DatabaseURL       string
	TemporalAddress   string
	TemporalNamespace string
	ShutdownTimeout   time.Duration
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		BindAddress:       envOrDefault("API_SERVER_BIND_ADDRESS", defaultBindAddress),
		DatabaseURL:       envOrDefault("DATABASE_URL", defaultDatabaseURL),
		TemporalAddress:   envOrDefault("TEMPORAL_HOST_PORT", defaultTemporalTarget),
		TemporalNamespace: envOrDefault("TEMPORAL_NAMESPACE", defaultNamespace),
		ShutdownTimeout:   defaultShutdownTime,
	}

	if cfg.BindAddress == "" {
		return Config{}, fmt.Errorf("%w: API_SERVER_BIND_ADDRESS is required", ErrInvalidConfig)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("%w: DATABASE_URL is required", ErrInvalidConfig)
	}
	if cfg.TemporalAddress == "" {
		return Config{}, fmt.Errorf("%w: TEMPORAL_HOST_PORT is required", ErrInvalidConfig)
	}
	if cfg.TemporalNamespace == "" {
		return Config{}, fmt.Errorf("%w: TEMPORAL_NAMESPACE is required", ErrInvalidConfig)
	}

	return cfg, nil
}

func envOrDefault(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}
