package worker

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/agentclash/agentclash/runtime/provider/throttle"
)

// loadProviderThrottleConfigFromEnv reads PROVIDER_* knobs.
// All defaults are 0 / off so unset envs preserve today's behavior.
//
//	PROVIDER_MAX_CONCURRENT_<PROVIDER>
//	PROVIDER_RPM_<PROVIDER>
//	PROVIDER_TPM_<PROVIDER>
//	PROVIDER_ACQUIRE_TIMEOUT (default 2m)
func loadProviderThrottleConfigFromEnv() (throttle.Config, error) {
	acquireTimeout, err := durationEnvOrDefault("PROVIDER_ACQUIRE_TIMEOUT", 2*time.Minute)
	if err != nil {
		return throttle.Config{}, err
	}

	limits := make(map[string]throttle.Limits)
	for _, env := range os.Environ() {
		key, value, ok := strings.Cut(env, "=")
		if !ok || value == "" {
			continue
		}
		var prefix string
		var field string
		switch {
		case strings.HasPrefix(key, "PROVIDER_MAX_CONCURRENT_"):
			prefix = "PROVIDER_MAX_CONCURRENT_"
			field = "concurrent"
		case strings.HasPrefix(key, "PROVIDER_RPM_"):
			prefix = "PROVIDER_RPM_"
			field = "rpm"
		case strings.HasPrefix(key, "PROVIDER_TPM_"):
			prefix = "PROVIDER_TPM_"
			field = "tpm"
		default:
			continue
		}
		name := strings.ToLower(strings.TrimPrefix(key, prefix))
		if name == "" {
			continue
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n < 0 {
			return throttle.Config{}, fmt.Errorf("%w: %s must be a non-negative integer", ErrInvalidConfig, key)
		}
		lim := limits[name]
		switch field {
		case "concurrent":
			lim.MaxConcurrent = int(n)
		case "rpm":
			lim.RPM = int(n)
		case "tpm":
			lim.TPM = n
		}
		limits[name] = lim
	}

	enabled := make(map[string]throttle.Limits)
	for name, lim := range limits {
		if lim.Enabled() {
			enabled[name] = lim
		}
	}

	return throttle.Config{
		LimitsByProvider: enabled,
		AcquireTimeout:   acquireTimeout,
	}, nil
}
