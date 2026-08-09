package api

import (
	"context"
	"net/http"

	temporalsdk "go.temporal.io/sdk/client"
)

type healthResponse struct {
	OK      bool   `json:"ok"`
	Service string `json:"service"`
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		OK:      true,
		Service: "api-server",
	})
}

// dbPinger is the subset of the Postgres connection pool that the readiness
// probe depends on. *pgxpool.Pool satisfies this.
type dbPinger interface {
	Ping(ctx context.Context) error
}

// temporalHealthChecker is the subset of the Temporal client that the
// readiness probe depends on. temporalsdk.Client satisfies this.
type temporalHealthChecker interface {
	CheckHealth(ctx context.Context, request *temporalsdk.CheckHealthRequest) (*temporalsdk.CheckHealthResponse, error)
}

// readyResponse reports overall readiness plus a per-dependency breakdown,
// so an unhealthy instance is diagnosable from the response body alone.
type readyResponse struct {
	OK      bool              `json:"ok"`
	Service string            `json:"service"`
	Checks  map[string]string `json:"checks"`
}

// healthzReadyHandler reports whether the API server's dependencies
// (Postgres, Temporal) are reachable. Unlike healthzHandler, this touches
// the network on every call, so it's meant for load-balancer/orchestrator
// readiness checks rather than a cheap liveness ping — the plain /healthz
// stays untouched for that purpose.
//
// db or temporal may be nil (e.g. a test router built without them); that
// is reported as "not configured" rather than a panic.
func healthzReadyHandler(db dbPinger, temporal temporalHealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		checks := make(map[string]string, 2)
		ready := true

		if db == nil {
			checks["postgres"] = "not configured"
			ready = false
		} else if err := db.Ping(ctx); err != nil {
			checks["postgres"] = "unreachable: " + err.Error()
			ready = false
		} else {
			checks["postgres"] = "ok"
		}

		if temporal == nil {
			checks["temporal"] = "not configured"
			ready = false
		} else if _, err := temporal.CheckHealth(ctx, &temporalsdk.CheckHealthRequest{}); err != nil {
			checks["temporal"] = "unreachable: " + err.Error()
			ready = false
		} else {
			checks["temporal"] = "ok"
		}

		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}

		writeJSON(w, status, readyResponse{
			OK:      ready,
			Service: "api-server",
			Checks:  checks,
		})
	}
}
