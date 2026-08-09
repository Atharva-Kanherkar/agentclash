package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	temporalsdk "go.temporal.io/sdk/client"
)

// fakeDBPinger is a minimal dbPinger double for exercising
// healthzReadyHandler without a real Postgres connection.
type fakeDBPinger struct {
	err error
}

func (f fakeDBPinger) Ping(_ context.Context) error {
	return f.err
}

// fakeTemporalHealthChecker is a minimal temporalHealthChecker double for
// exercising healthzReadyHandler without a real Temporal connection.
type fakeTemporalHealthChecker struct {
	err error
}

func (f fakeTemporalHealthChecker) CheckHealth(_ context.Context, _ *temporalsdk.CheckHealthRequest) (*temporalsdk.CheckHealthResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &temporalsdk.CheckHealthResponse{}, nil
}

func TestHealthzReadyHandler(t *testing.T) {
	errDown := errors.New("connection refused")

	cases := []struct {
		name       string
		db         dbPinger
		temporal   temporalHealthChecker
		wantStatus int
		wantOK     bool
	}{
		{
			name:       "postgres and temporal both reachable",
			db:         fakeDBPinger{},
			temporal:   fakeTemporalHealthChecker{},
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name:       "postgres unreachable",
			db:         fakeDBPinger{err: errDown},
			temporal:   fakeTemporalHealthChecker{},
			wantStatus: http.StatusServiceUnavailable,
			wantOK:     false,
		},
		{
			name:       "temporal unreachable",
			db:         fakeDBPinger{},
			temporal:   fakeTemporalHealthChecker{err: errDown},
			wantStatus: http.StatusServiceUnavailable,
			wantOK:     false,
		},
		{
			name:       "both unreachable",
			db:         fakeDBPinger{err: errDown},
			temporal:   fakeTemporalHealthChecker{err: errDown},
			wantStatus: http.StatusServiceUnavailable,
			wantOK:     false,
		},
		{
			name:       "dependencies not configured",
			db:         nil,
			temporal:   nil,
			wantStatus: http.StatusServiceUnavailable,
			wantOK:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
			recorder := httptest.NewRecorder()

			healthzReadyHandler(tc.db, tc.temporal)(recorder, req)

			if recorder.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, tc.wantStatus)
			}

			var body readyResponse
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("decoding response body: %v", err)
			}
			if body.OK != tc.wantOK {
				t.Errorf("body.OK = %v, want %v", body.OK, tc.wantOK)
			}
			if body.Service != "api-server" {
				t.Errorf("body.Service = %q, want %q", body.Service, "api-server")
			}
			if _, ok := body.Checks["postgres"]; !ok {
				t.Errorf(`body.Checks missing "postgres" key`)
			}
			if _, ok := body.Checks["temporal"]; !ok {
				t.Errorf(`body.Checks missing "temporal" key`)
			}
		})
	}
}

// TestHealthzUnaffectedByReadyRoute guards against a regression where wiring
// the new /healthz/ready route accidentally changes the existing cheap
// liveness endpoint (e.g. by making it depend on the same handler chain).
func TestHealthzUnaffectedByReadyRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	healthzHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body healthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
	if !body.OK {
		t.Errorf("body.OK = %v, want true", body.OK)
	}
}
