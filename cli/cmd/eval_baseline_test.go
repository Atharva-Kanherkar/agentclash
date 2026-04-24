package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// baselineFakeState records what happened against the fake baselines
// endpoints so assertions can distinguish "auto-save on first run" from
// "skipped because --no-baseline".
type baselineFakeState struct {
	getCalls    int32
	postCalls   int32
	postedName  string
	postedBody  map[string]any
	getResponse func() (int, any) // server responds with (status, body)
}

// newEvalWithBaselineFakeAPI extends newEvalFakeAPI's routes with the
// baseline GET/POST handlers. `getResponse` lets each test control whether
// the GET returns 200 with a stored ranking or 404.
func newEvalWithBaselineFakeAPI(t *testing.T, wsID string, scn *evalScenario, state *baselineFakeState) *httptest.Server {
	t.Helper()
	routes := map[string]http.HandlerFunc{
		"POST /v1/workspaces/" + wsID + "/challenge-packs": jsonHandler(200, map[string]any{
			"challenge_pack_id":         "pack-1",
			"challenge_pack_version_id": scn.packVersionID,
		}),
		"POST /v1/runs": jsonHandler(200, map[string]any{
			"id":     scn.runID,
			"status": "queued",
		}),
		"GET /v1/runs/" + scn.runID: func(w http.ResponseWriter, r *http.Request) {
			jsonHandler(200, map[string]any{
				"id":           scn.runID,
				"name":         "eval-run",
				"status":       scn.nextStatus(),
				"workspace_id": wsID,
				"started_at":   "2026-04-24T10:00:00Z",
				"finished_at":  "2026-04-24T10:00:20Z",
			})(w, r)
		},
		"GET /v1/runs/" + scn.runID + "/ranking": jsonHandler(200, scn.rankingPayload),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}
		// Baseline routes — the test owns state via closures.
		if strings.HasPrefix(r.URL.Path, "/v1/workspaces/"+wsID+"/baselines/") {
			name := strings.TrimPrefix(r.URL.Path, "/v1/workspaces/"+wsID+"/baselines/")
			switch r.Method {
			case http.MethodGet:
				atomic.AddInt32(&state.getCalls, 1)
				status := 404
				var body any = map[string]any{"error": map[string]any{"code": "baseline_not_found"}}
				if state.getResponse != nil {
					status, body = state.getResponse()
				}
				jsonHandler(status, body)(w, r)
			case http.MethodPost:
				atomic.AddInt32(&state.postCalls, 1)
				state.postedName = name
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &state.postedBody)
				jsonHandler(200, map[string]any{
					"id":              "baseline-id",
					"workspace_id":    wsID,
					"name":            name,
					"pack_version_id": scn.packVersionID,
					"run_id":          scn.runID,
				})(w, r)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}
		t.Logf("unhandled: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found"}}`))
	}))
	return srv
}

// baselineGetSuccess helps tests whose GET should return 200 with a stored
// ranking snapshot. Returns the `state.getResponse` shape expected above.
func baselineGetSuccess(snapshot map[string]any) func() (int, any) {
	return func() (int, any) {
		return 200, map[string]any{
			"id":                 "baseline-id",
			"name":               "main",
			"pack_version_id":    "pv-baseline",
			"run_id":             "run-baseline",
			"scorecard_snapshot": snapshot,
		}
	}
}

func baselineGetNotFound() func() (int, any) {
	return func() (int, any) {
		return 404, map[string]any{"error": map[string]any{"code": "baseline_not_found"}}
	}
}

func TestEvalAutoSavesMainOnFirstRun(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv-auto", runID: "run-auto",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{
				"ranking": map[string]any{
					"items": []map[string]any{
						{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.85},
					},
				},
			},
		}
		state := &baselineFakeState{getResponse: baselineGetNotFound()}
		ts := newEvalWithBaselineFakeAPI(t, "ws-1", scn, state)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1"}, ts.URL)
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		if atomic.LoadInt32(&state.getCalls) != 1 {
			t.Fatalf("expected 1 GET to /baselines/main, got %d", state.getCalls)
		}
		if atomic.LoadInt32(&state.postCalls) != 1 {
			t.Fatalf("expected 1 POST to /baselines/main (auto-save), got %d", state.postCalls)
		}
		if state.postedName != "main" {
			t.Fatalf("posted name = %q, want main", state.postedName)
		}
		if state.postedBody["run_id"] != "run-auto" {
			t.Fatalf("baseline body run_id = %v, want run-auto", state.postedBody["run_id"])
		}
		if state.postedBody["pack_version_id"] != "pv-auto" {
			t.Fatalf("baseline body pack_version_id = %v, want pv-auto", state.postedBody["pack_version_id"])
		}
		if _, ok := state.postedBody["scorecard_snapshot"]; !ok {
			t.Fatalf("baseline body missing scorecard_snapshot")
		}
	})
}

func TestEvalDetectsStatusRegressionVsBaseline(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{
			"gpt-5":      "dep-a",
			"claude-4.7": "dep-b",
		})
		writePack(t, dir, "pack.yaml")

		baselineSnap := map[string]any{
			"ranking": map[string]any{
				"items": []map[string]any{
					{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.9},
					{"rank": 2, "label": "claude-4.7", "status": "completed", "composite_score": 0.85},
				},
			},
		}
		scn := &evalScenario{
			packVersionID: "pv-r", runID: "run-r",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{
				"ranking": map[string]any{
					"items": []map[string]any{
						{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.88},
						{"rank": 2, "label": "claude-4.7", "status": "failed", "composite_score": 0.3},
					},
				},
			},
		}
		state := &baselineFakeState{getResponse: baselineGetSuccess(baselineSnap)}
		ts := newEvalWithBaselineFakeAPI(t, "ws-1", scn, state)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{"eval", "pack.yaml", "--models", "gpt-5,claude-4.7", "-w", "ws-1"}, ts.URL)
		var exitErr *ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *ExitCodeError with regression; got %T (%v)", err, err)
		}
		if exitErr.Code != 1 {
			t.Fatalf("exit = %d, want 1", exitErr.Code)
		}
		if !strings.Contains(exitErr.Message, "REGRESSION_DETECTED") {
			t.Fatalf("exit message should name REGRESSION_DETECTED: %v", exitErr.Message)
		}
		if !strings.Contains(exitErr.Message, "claude-4.7") {
			t.Fatalf("regression should name the affected agent: %v", exitErr.Message)
		}
		// Existing baseline present + run completed cleanly: no auto-save;
		// user wanted to compare, not to overwrite.
		if atomic.LoadInt32(&state.postCalls) != 0 {
			t.Fatalf("must not overwrite an existing baseline during a default compare; got %d POSTs", state.postCalls)
		}
	})
}

func TestEvalPassesWhenNewRunMatchesBaseline(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		baselineSnap := map[string]any{
			"ranking": map[string]any{
				"items": []map[string]any{
					{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.88},
				},
			},
		}
		scn := &evalScenario{
			packVersionID: "pv-ok", runID: "run-ok",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{
				"ranking": map[string]any{
					"items": []map[string]any{
						{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.9},
					},
				},
			},
		}
		state := &baselineFakeState{getResponse: baselineGetSuccess(baselineSnap)}
		ts := newEvalWithBaselineFakeAPI(t, "ws-1", scn, state)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1"}, ts.URL); err != nil {
			t.Fatalf("eval: %v", err)
		}
	})
}

func TestEvalNoBaselineFlagSkipsBothSaveAndCompare(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv-nb", runID: "run-nb",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{
				"ranking": map[string]any{
					"items": []map[string]any{
						{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.9},
					},
				},
			},
		}
		state := &baselineFakeState{getResponse: baselineGetNotFound()}
		ts := newEvalWithBaselineFakeAPI(t, "ws-1", scn, state)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1", "--no-baseline",
		}, ts.URL); err != nil {
			t.Fatalf("eval: %v", err)
		}
		if atomic.LoadInt32(&state.getCalls) != 0 {
			t.Fatalf("--no-baseline must skip the GET; got %d", state.getCalls)
		}
		if atomic.LoadInt32(&state.postCalls) != 0 {
			t.Fatalf("--no-baseline must skip the POST; got %d", state.postCalls)
		}
	})
}

func TestEvalSaveAsOverridesDefaultName(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv-sv", runID: "run-sv",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{
				"ranking": map[string]any{
					"items": []map[string]any{
						{"rank": 1, "label": "gpt-5", "status": "completed", "composite_score": 0.9},
					},
				},
			},
		}
		state := &baselineFakeState{getResponse: baselineGetNotFound()}
		ts := newEvalWithBaselineFakeAPI(t, "ws-1", scn, state)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1", "--save-as", "production",
		}, ts.URL); err != nil {
			t.Fatalf("eval: %v", err)
		}
		if state.postedName != "production" {
			t.Fatalf("posted name = %q, want production", state.postedName)
		}
	})
}

func TestEvalCompareToMissingExplicitBaselineFails(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		scn := &evalScenario{
			packVersionID: "pv-c", runID: "run-c",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{},
		}
		state := &baselineFakeState{getResponse: baselineGetNotFound()}
		ts := newEvalWithBaselineFakeAPI(t, "ws-1", scn, state)
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1", "--compare-to", "missing-name",
		}, ts.URL)
		var exitErr *ExitCodeError
		if !errors.As(err, &exitErr) {
			t.Fatalf("expected *ExitCodeError for missing baseline, got %T (%v)", err, err)
		}
		if exitErr.Code != 1 {
			t.Fatalf("exit = %d, want 1", exitErr.Code)
		}
		if !strings.Contains(exitErr.Message, "missing-name") {
			t.Fatalf("error should name the missing baseline: %v", exitErr.Message)
		}
		// Must fail before publishing a pack or creating a run — the
		// backend baseline GET is the first network call.
		if atomic.LoadInt32(&scn.statusCalls) != 0 {
			t.Fatalf("should fail before polling any run status; got %d polls", scn.statusCalls)
		}
	})
}

func TestEvalSaveAsAndNoBaselineAreMutuallyExclusive(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("no network call should be made — flag validation must fail first")
			w.WriteHeader(500)
		}))
		defer ts.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1",
			"--save-as", "x", "--no-baseline",
		}, ts.URL)
		if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("expected mutually-exclusive error, got: %v", err)
		}
	})
}

// buildRankingPayload constructs the map shape the CLI sees after JSON
// decoding — `items` as `[]any` with each element a `map[string]any`. A
// Go-typed `[]map[string]any` literal would not survive the real wire
// format and would spuriously pass/fail the regression math.
func buildRankingPayload(items ...map[string]any) map[string]any {
	asAny := make([]any, len(items))
	for i, it := range items {
		asAny[i] = it
	}
	return map[string]any{
		"ranking": map[string]any{"items": asAny},
	}
}

func TestEvalAbsentAgentInNewRunCountsAsRegression(t *testing.T) {
	// Baseline had gpt-5 completed; new run never registered gpt-5 at
	// all (e.g., someone renamed the deployment). The binary rule is
	// "silent disappearance is worse than a loud failure".
	baseline := buildRankingPayload(
		map[string]any{"label": "gpt-5", "status": "completed", "composite_score": 0.9},
	)
	current := buildRankingPayload(
		map[string]any{"label": "claude-4.7", "status": "completed", "composite_score": 0.8},
	)
	regs := computeRegressions(baseline, current)
	if len(regs) != 1 {
		t.Fatalf("expected 1 regression for absent agent, got %d: %+v", len(regs), regs)
	}
	if regs[0].Agent != "gpt-5" || regs[0].NewStatus != "absent" {
		t.Fatalf("unexpected regression shape: %+v", regs[0])
	}
}

func TestEvalBaselineWithPartialStatusNotTreatedAsRegressionSource(t *testing.T) {
	// A baseline that itself wasn't fully clean shouldn't fire
	// regressions for agents that were already degraded — otherwise
	// users who bookmarked a known-bad state get spurious verdicts.
	baseline := buildRankingPayload(
		map[string]any{"label": "gpt-5", "status": "failed", "composite_score": 0.3},
		map[string]any{"label": "claude-4.7", "status": "completed", "composite_score": 0.9},
	)
	current := buildRankingPayload(
		map[string]any{"label": "gpt-5", "status": "failed", "composite_score": 0.25},
		map[string]any{"label": "claude-4.7", "status": "completed", "composite_score": 0.88},
	)
	regs := computeRegressions(baseline, current)
	if len(regs) != 0 {
		t.Fatalf("expected 0 regressions (baseline degraded entry ignored), got %d: %+v", len(regs), regs)
	}
}

func TestEvalJSONOutputIncludesBaselineBlock(t *testing.T) {
	fastPoll(t)
	withTempCwd(t, func(dir string) {
		writeLinkedProject(t, dir, "ws-1", map[string]string{"gpt-5": "dep-a"})
		writePack(t, dir, "pack.yaml")

		baselineSnap := map[string]any{
			"ranking": map[string]any{
				"items": []map[string]any{
					{"label": "gpt-5", "status": "completed", "composite_score": 0.9},
				},
			},
		}
		scn := &evalScenario{
			packVersionID: "pv-j", runID: "run-j",
			statusSeq: []string{"completed"},
			rankingPayload: map[string]any{
				"ranking": map[string]any{
					"items": []map[string]any{
						{"label": "gpt-5", "status": "failed", "composite_score": 0.2},
					},
				},
			},
		}
		state := &baselineFakeState{getResponse: baselineGetSuccess(baselineSnap)}
		ts := newEvalWithBaselineFakeAPI(t, "ws-1", scn, state)
		defer ts.Close()

		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		// Expected to fail (regression), but --json emits the scorecard
		// first and the exit error is surfaced separately.
		_ = executeCommand(t, []string{
			"eval", "pack.yaml", "--models", "gpt-5", "-w", "ws-1", "--json",
		}, ts.URL)
		out := stdout.finish()
		for _, want := range []string{
			"\"compared_to\": \"main\"",
			"\"compared\": true",
			"\"regression_count\": 1",
			"\"regressions\":",
			"\"agent\": \"gpt-5\"",
			"\"new_status\": \"failed\"",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("--json output missing %q\n---\n%s", want, out)
			}
		}
	})
}
