package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentclash/agentclash/backend/internal/pubsub"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type rejectingSSEGate struct{}

func (rejectingSSEGate) TryAcquire(context.Context) bool { return false }
func (rejectingSSEGate) Release(context.Context)         {}

func TestSSECapacityExceededReturns503(t *testing.T) {
	prev := configuredSSEGate
	ConfigureSSEConnectionGate(rejectingSSEGate{})
	t.Cleanup(func() { ConfigureSSEConnectionGate(prev) })

	runID := uuid.New()
	auth := &capturingSSEAuthenticator{caller: Caller{UserID: uuid.New()}}
	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID.String()+"/events/stream", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("runID", runID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	streamRunEventsHandler(discardLogger(), auth, &fakeSSERunReadService{}, pubsub.NoopSubscriber{}).ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
}
