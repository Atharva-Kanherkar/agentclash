package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agentclash/agentclash/runtime/evalset"
	"github.com/google/uuid"
)

func TestExpandEvalSetEndpoint(t *testing.T) {
	workspaceID := uuid.New()
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{})

	manifest := map[string]any{
		"schema": evalset.SchemaV1,
		"name":   "nightly",
		"packs":  []string{"catalog/a", "catalog/b"},
		"agents": []map[string]string{
			{"deployment": "d1"},
			{"deployment": "d2"},
			{"deployment": "d3"},
		},
		"repeats": 5,
	}
	body, _ := json.Marshal(map[string]any{
		"workspace_id": workspaceID.String(),
		"manifest":     manifest,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/eval-sets/expand", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), callerContextKey{}, Caller{UserID: uuid.New()}))
	rr := httptest.NewRecorder()
	expandEvalSetHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), manager).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var report evalset.ExpansionReport
	if err := json.Unmarshal(rr.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Count != 30 {
		t.Fatalf("count = %d, want 30", report.Count)
	}
	if report.Combinations[0].MatrixKey == "" {
		t.Fatal("expected matrix keys")
	}
}

func TestExpandEvalSetEndpointRejectsOverCap(t *testing.T) {
	workspaceID := uuid.New()
	manager := NewEvalSetManager(allowWorkspaceAuthorizer{})
	manifest := map[string]any{
		"schema":  evalset.SchemaV1,
		"name":    "big",
		"packs":   []string{"a", "b"},
		"agents":  []map[string]string{{"deployment": "x"}, {"deployment": "y"}},
		"repeats": 100,
	}
	body, _ := json.Marshal(map[string]any{
		"workspace_id":     workspaceID.String(),
		"manifest":         manifest,
		"max_combinations": 10,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/eval-sets/expand", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), callerContextKey{}, Caller{UserID: uuid.New()}))
	rr := httptest.NewRecorder()
	expandEvalSetHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), manager).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}
