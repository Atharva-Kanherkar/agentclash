package cmd

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agentclash/agentclash/cli/internal/config"
)

// evalScenario wires up a fake API server with just enough state to drive
// eval from pack publish through polling to ranking. statusSeq controls the
// run.status the server returns on each GET /v1/runs/<id>, letting tests
// simulate "running, running, completed" cheaply.
type evalScenario struct {
	packVersionID string
	runID         string
	statusSeq     []string
	rankingPayload any
	statusCalls   int32
}

func (s *evalScenario) nextStatus() string {
	i := atomic.AddInt32(&s.statusCalls, 1) - 1
	if int(i) >= len(s.statusSeq) {
		return s.statusSeq[len(s.statusSeq)-1]
	}
	return s.statusSeq[i]
}

// newEvalFakeAPI builds the fake server routes eval touches: publish, run
// create, run GET (for polling), ranking. Every route is keyed by METHOD+PATH
// so unexpected calls surface as 404s.
func newEvalFakeAPI(t *testing.T, wsID string, s *evalScenario) *httptest.Server {
	t.Helper()
	routes := map[string]http.HandlerFunc{
		"POST /v1/workspaces/" + wsID + "/challenge-packs": jsonHandler(200, map[string]any{
			"challenge_pack_id":         "pack-1",
			"challenge_pack_version_id": s.packVersionID,
		}),
		"POST /v1/runs": jsonHandler(200, map[string]any{
			"id":     s.runID,
			"status": "queued",
		}),
		"GET /v1/runs/" + s.runID: func(w http.ResponseWriter, r *http.Request) {
			jsonHandler(200, map[string]any{
				"id":           s.runID,
				"name":         "eval-run",
				"status":       s.nextStatus(),
				"workspace_id": wsID,
				"started_at":   "2026-04-24T10:00:00Z",
				"finished_at":  "2026-04-24T10:00:20Z",
			})(w, r)
		},
		"GET /v1/runs/" + s.runID + "/ranking": jsonHandler(200, s.rankingPayload),
	}
	return fakeAPI(t, routes)
}

// writeLinkedProject drops a .agentclash.yaml with a deployments map so eval
// can resolve --models locally.
func writeLinkedProject(t *testing.T, dir, wsID string, deployments map[string]string) {
	t.Helper()
	cfg := config.ProjectConfig{
		WorkspaceID:   wsID,
		WorkspaceName: "test-ws",
		OrgID:         "org-1",
		Deployments:   deployments,
	}
	if err := config.WriteProjectConfig(dir, cfg); err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

// writePack drops a minimal YAML pack file that the fake /challenge-packs
// endpoint will accept.
func writePack(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("pack:\n  slug: t\n"), 0644); err != nil {
		t.Fatalf("write pack: %v", err)
	}
	return path
}

// fastPoll shrinks the 2s production poll to 1ms so tests finish quickly.
func fastPoll(t *testing.T) {
	t.Helper()
	orig := evalPollInterval
	evalPollInterval = time.Millisecond
	t.Cleanup(func() { evalPollInterval = orig })
}

func TestEvalHappyPathCompletedRun(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{
			"gpt-5":      "dep-a",
			"claude-4.7": "dep-b",
		})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv-1",
			runID:         "run-1",
			statusSeq:     []string{"queued", "running", "completed"},
			rankingPayload: map[string]any{
				"state": "ready",
				"ranking": map[string]any{
					"items": []map[string]any{
						{
							"rank": 1, "label": "claude-4.7", "status": "completed",
							"composite_score": 0.91, "correctness_score": 0.95,
						},
						{
							"rank": 2, "label": "gpt-5", "status": "completed",
							"composite_score": 0.87,
						},
					},
				},
			},
		}
		ts := newEvalFakeAPI(t, "ws-1", scn)
		defer ts.Close()

		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{
			"eval", "pack.yaml",
			"--models", "gpt-5,claude-4.7",
			"-w", "ws-1",
		}, ts.URL)
		if err != nil {
			t.Fatalf("eval happy path: %v", err)
		}
		out := stdout.finish()
		for _, want := range []string{
			"Scorecard",
			"run-1",
			"completed",
			"gpt-5, claude-4.7",
			"claude-4.7",
			"0.91",
			"0.87",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("eval output missing %q\n---\n%s", want, out)
			}
		}
		if scn.statusCalls < 1 {
			t.Fatalf("expected at least one status poll; got %d", scn.statusCalls)
		}
	})
}

func TestEvalFailsFastOnUnknownModelWithDeployLink(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		// The server must NOT be hit — eval must fail locally before
		// publish. We instrument a handler that counts calls.
		hits := int32(0)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&hits, 1)
			w.WriteHeader(500)
		}))
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{
			"eval", "pack.yaml",
			"--models", "gpt-5,o3-mini,claude-4.7",
			"-w", "ws-1",
		}, ts.URL)
		if err == nil {
			t.Fatal("expected error for missing deployments")
		}
		msg := err.Error()
		for _, want := range []string{
			"deployment(s) not found",
			"o3-mini",
			"claude-4.7",
			"available: gpt-5",
			"https://agentclash.dev/workspaces/ws-1/deployments",
		} {
			if !strings.Contains(msg, want) {
				t.Fatalf("error missing %q: %v", want, msg)
			}
		}
		if atomic.LoadInt32(&hits) != 0 {
			t.Fatalf("eval hit backend %d times before validating models; must fail locally", hits)
		}
	})
}

func TestEvalRequiresLinkedProject(t *testing.T) {
	withTempCwd(t, func(dir string) {
		writePack(t, dir, "pack.yaml")
		// No .agentclash.yaml written.

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{
			"eval", "pack.yaml",
			"--models", "gpt-5",
			"-w", "ws-1",
		}, "http://unused")
		if err == nil {
			t.Fatal("expected error when .agentclash.yaml is missing")
		}
		if !strings.Contains(err.Error(), "agentclash link") {
			t.Fatalf("error should point at agentclash link: %v", err)
		}
	})
}

func TestEvalExitsNonZeroOnRunFailure(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv-1", runID: "run-x",
			statusSeq:      []string{"running", "failed"},
			rankingPayload: map[string]any{"ranking": map[string]any{"items": []map[string]any{}}},
		}
		ts := newEvalFakeAPI(t, "ws-1", scn)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1",
		}, ts.URL)
		var exitErr *ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *ExitCodeError, got %T (%v)", err, err)
		}
		if exitErr.Code != 1 {
			t.Fatalf("exit code = %d, want 1", exitErr.Code)
		}
		if !strings.Contains(exitErr.Message, "failed") {
			t.Fatalf("error should mention terminal status: %v", exitErr.Message)
		}
	})
}

func TestEvalAllowPartialMakesPartiallyCompletedPass(t *testing.T) {
	fastPoll(t)
	for _, tc := range []struct {
		name      string
		flag      bool
		wantExit  int // 0 = nil error
		wantError bool
	}{
		{"default exits 1", false, 1, true},
		{"allow-partial exits 0", true, 0, false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			withTempCwd(t, func(dir string) {
				writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
				writePack(t, dir, "pack.yaml")

				scn := &evalScenario{
					packVersionID: "pv", runID: "run-p",
					statusSeq:      []string{"partially_completed"},
					rankingPayload: map[string]any{"ranking": map[string]any{"items": []map[string]any{}}},
				}
				ts := newEvalFakeAPI(t, "ws-1", scn)
				defer ts.Close()

				args := []string{"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1"}
				if tc.flag {
					args = append(args, "--allow-partial")
				}
				t.Setenv("AGENTCLASH_TOKEN", "test-tok")
				err := executeCommand(t, args, ts.URL)
				if tc.wantError {
					var exitErr *ExitCodeError
					if !errors.As(err, &exitErr) {
						t.Fatalf("expected *ExitCodeError, got %T (%v)", err, err)
					}
					if exitErr.Code != tc.wantExit {
						t.Fatalf("exit code = %d, want %d", exitErr.Code, tc.wantExit)
					}
				} else if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			})
		})
	}
}

func TestEvalTimeoutReturnsExitCode2(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv", runID: "run-stuck",
			statusSeq:      []string{"running"}, // never terminal
			rankingPayload: map[string]any{},
		}
		ts := newEvalFakeAPI(t, "ws-1", scn)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1",
			"--timeout", "50ms",
		}, ts.URL)
		var exitErr *ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *ExitCodeError, got %T (%v)", err, err)
		}
		if exitErr.Code != 2 {
			t.Fatalf("exit code = %d, want 2 (timeout)", exitErr.Code)
		}
	})
}

func TestEvalJSONOutputIncludesRunAndRanking(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv-json", runID: "run-json",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{
				"ranking": map[string]any{
					"items": []map[string]any{
						{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.77},
					},
				},
			},
		}
		ts := newEvalFakeAPI(t, "ws-1", scn)
		defer ts.Close()

		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1", "--json",
		}, ts.URL); err != nil {
			t.Fatalf("eval --json: %v", err)
		}
		out := stdout.finish()
		for _, want := range []string{
			"\"run_id\": \"run-json\"",
			"\"pack_version_id\": \"pv-json\"",
			"\"agent_deployment_ids\":",
			"\"ranking\":",
			"\"composite_score\": 0.77",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("--json missing %q\n---\n%s", want, out)
			}
		}
	})
}

func TestNormaliseModelNamesDedupesAndSplits(t *testing.T) {
	got := normaliseModelNames([]string{" gpt-5, claude-4.7 ", "gpt-5", "  "})
	want := []string{"gpt-5", "claude-4.7"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
