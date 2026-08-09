package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEvalsetSubmitDryRunExpand(t *testing.T) {
	t.Setenv("AGENTCLASH_TOKEN", "test-tok")
	pack := uuid.New().String()
	agent := uuid.New().String()
	var createHit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/eval-sets/expand":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":        "nightly",
				"count":       2,
				"pack_count":  1,
				"agent_count": 1,
				"repeats":     2,
				"combinations": []map[string]any{
					{"matrix_key": pack + "/" + agent + "/1", "pack_ref": pack, "agent_ref": agent, "repeat": 1},
					{"matrix_key": pack + "/" + agent + "/2", "pack_ref": pack, "agent_ref": agent, "repeat": 2},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/eval-sets":
			createHit = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "sweep.yaml")
	manifest := "schema: evalset/v1\nname: nightly\npacks:\n  - " + pack + "\nagents:\n  - deployment: " + agent + "\nrepeats: 2\n"
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := captureStdout(t)
	err := executeCommand(t, []string{"evalset", "submit", path, "--dry-run", "--json", "-w", "ws-1"}, srv.URL)
	out := stdout.finish()
	if err != nil {
		t.Fatalf("submit dry-run: %v\n%s", err, out)
	}
	if createHit {
		t.Fatal("create must not be called on --dry-run")
	}
	if !strings.Contains(out, `"count"`) {
		t.Fatalf("stdout missing count: %s", out)
	}
}

func TestEvalsetLocalFlagRejected(t *testing.T) {
	t.Setenv("AGENTCLASH_TOKEN", "test-tok")
	dir := t.TempDir()
	path := filepath.Join(dir, "sweep.yaml")
	_ = os.WriteFile(path, []byte("schema: evalset/v1\nname: x\npacks: [a]\nagents: [{deployment: b}]\n"), 0o644)
	err := executeCommand(t, []string{"evalset", "submit", path, "--local", "-w", "ws-1"}, "http://127.0.0.1:9")
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *cliError
	if !errors.As(err, &ce) || ce.Code != "local_not_available" {
		t.Fatalf("err = %v", err)
	}
}

func TestEvalsetCancel(t *testing.T) {
	t.Setenv("AGENTCLASH_TOKEN", "test-tok")
	id := uuid.New().String()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/eval-sets/"+id+"/cancel" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "cancelled"})
	}))
	defer srv.Close()
	stdout := captureStdout(t)
	err := executeCommand(t, []string{"evalset", "cancel", id, "--json", "-w", "ws-1"}, srv.URL)
	out := stdout.finish()
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if !strings.Contains(out, "cancelled") {
		t.Fatalf("stdout = %s", out)
	}
}

func TestEvalsetStatusJSONStableFields(t *testing.T) {
	t.Setenv("AGENTCLASH_TOKEN", "test-tok")
	id := uuid.New().String()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"eval_set": map[string]any{
				"id":                id,
				"name":              "nightly",
				"status":            "completed",
				"combination_count": 2,
			},
			"eval_session_ids": []string{uuid.New().String()},
			"result": map[string]any{
				"session_count": 1,
				"run_count":     2,
				"aggregate": map[string]any{
					"sessions": 1,
					"runs":     2,
					"combinations": []map[string]any{
						{"matrix_key": "p/a/1", "pack_ref": "p", "status": "completed"},
					},
					"per_pack": map[string]any{"p": 1},
				},
			},
		})
	}))
	defer srv.Close()
	stdout := captureStdout(t)
	err := executeCommand(t, []string{"evalset", "status", id, "--json", "-w", "ws-1"}, srv.URL)
	out := stdout.finish()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"eval_set", "eval_session_ids", "result"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing stable field %q in %s", key, out)
		}
	}
}

func TestEvalsetReportCSV(t *testing.T) {
	t.Setenv("AGENTCLASH_TOKEN", "test-tok")
	id := uuid.New().String()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"eval_set": map[string]any{"id": id, "status": "completed", "name": "n", "combination_count": 1},
			"result": map[string]any{
				"aggregate": map[string]any{
					"sessions": 1,
					"runs":     1,
					"combinations": []map[string]any{
						{"matrix_key": "p/a/1", "pack_ref": "p", "status": "completed"},
					},
				},
			},
		})
	}))
	defer srv.Close()
	stdout := captureStdout(t)
	err := executeCommand(t, []string{"evalset", "report", id, "--format", "csv", "-w", "ws-1"}, srv.URL)
	out := stdout.finish()
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.Contains(out, "matrix_key,pack_ref,status") {
		t.Fatalf("csv header missing: %s", out)
	}
	if !strings.Contains(out, "p/a/1,p,completed") {
		t.Fatalf("csv row missing: %s", out)
	}
}

func TestEvalsetFailedExitCode(t *testing.T) {
	t.Setenv("AGENTCLASH_TOKEN", "test-tok")
	id := uuid.New().String()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"eval_set": map[string]any{"id": id, "status": "failed", "name": "n", "combination_count": 1},
			"result":   map[string]any{},
		})
	}))
	defer srv.Close()
	err := executeCommand(t, []string{"evalset", "status", id, "--json", "-w", "ws-1"}, srv.URL)
	if err == nil {
		t.Fatal("expected exit error for failed set")
	}
	var ee *ExitCodeError
	if !errors.As(err, &ee) || ee.Code != evalsetExitFailed {
		t.Fatalf("err = %v", err)
	}
}

func TestEvalsetInitWritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agentclash.evalset.yaml")
	err := executeCommand(t, []string{"evalset", "init", path}, "http://127.0.0.1:9")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "schema: evalset/v1") {
		t.Fatalf("content = %s", raw)
	}
}
