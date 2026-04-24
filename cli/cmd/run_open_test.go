package cmd

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWebRunURLTransformations(t *testing.T) {
	cases := []struct {
		apiURL string
		runID  string
		want   string
	}{
		{"https://api.agentclash.dev", "run-abc", "https://agentclash.dev/runs/run-abc"},
		{"https://staging-api.agentclash.dev", "r1", "https://staging.agentclash.dev/runs/r1"},
		{"http://localhost:8080", "r1", "http://localhost:3000/runs/r1"},
		{"http://127.0.0.1:8080", "r1", "http://127.0.0.1:3000/runs/r1"},
		{"https://custom.example.com", "r1", "https://custom.example.com/runs/r1"},
		{"garbage", "r1", "https://agentclash.dev/runs/r1"},
		{"", "r1", "https://agentclash.dev/runs/r1"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.apiURL, func(t *testing.T) {
			if got := webRunURL(tc.apiURL, tc.runID); got != tc.want {
				t.Fatalf("webRunURL(%q) = %q, want %q", tc.apiURL, got, tc.want)
			}
		})
	}
}

func TestRunOpenJSONPrintsURLWithoutLaunchingBrowser(t *testing.T) {
	var browserCalls int32
	origOpen := openBrowserFunc
	openBrowserFunc = func(raw string) error {
		atomic.AddInt32(&browserCalls, 1)
		return nil
	}
	defer func() { openBrowserFunc = origOpen }()

	withTempCwd(t, func(_ string) {
		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"run", "open", "r-42", "--json"}, "https://api.agentclash.dev"); err != nil {
			t.Fatalf("run open --json: %v", err)
		}
		out := stdout.finish()
		for _, want := range []string{
			"\"run_id\": \"r-42\"",
			"\"url\": \"https://agentclash.dev/runs/r-42\"",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q in --json output:\n%s", want, out)
			}
		}
		if atomic.LoadInt32(&browserCalls) != 0 {
			t.Fatalf("run open --json must not launch a browser; got %d calls", browserCalls)
		}
	})
}

func TestRunOpenNonInteractiveFallsBackToPrintingURL(t *testing.T) {
	var browserCalls int32
	origOpen := openBrowserFunc
	openBrowserFunc = func(_ string) error {
		atomic.AddInt32(&browserCalls, 1)
		return nil
	}
	defer func() { openBrowserFunc = origOpen }()

	origInteractive := isInteractiveTerminal
	isInteractiveTerminal = func(_ *RunContext) bool { return false }
	defer func() { isInteractiveTerminal = origInteractive }()

	withTempCwd(t, func(_ string) {
		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"run", "open", "r-7"}, "https://api.agentclash.dev"); err != nil {
			t.Fatalf("run open: %v", err)
		}
		out := stdout.finish()
		if !strings.Contains(out, "https://agentclash.dev/runs/r-7") {
			t.Fatalf("non-interactive run open should print URL; got: %q", out)
		}
		if atomic.LoadInt32(&browserCalls) != 0 {
			t.Fatalf("non-interactive run open must not launch browser; got %d calls", browserCalls)
		}
	})
}

func TestRunOpenInteractiveLaunchesBrowserAndPrintsURL(t *testing.T) {
	var gotURL string
	var browserCalls int32
	origOpen := openBrowserFunc
	openBrowserFunc = func(raw string) error {
		gotURL = raw
		atomic.AddInt32(&browserCalls, 1)
		return nil
	}
	defer func() { openBrowserFunc = origOpen }()

	origInteractive := isInteractiveTerminal
	isInteractiveTerminal = func(_ *RunContext) bool { return true }
	defer func() { isInteractiveTerminal = origInteractive }()

	withTempCwd(t, func(_ string) {
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		stdout := captureStdout(t)
		if err := executeCommand(t, []string{"run", "open", "r-42"}, "https://api.agentclash.dev"); err != nil {
			t.Fatalf("run open: %v", err)
		}
		out := stdout.finish()
		if atomic.LoadInt32(&browserCalls) != 1 {
			t.Fatalf("expected 1 browser launch; got %d", browserCalls)
		}
		if gotURL != "https://agentclash.dev/runs/r-42" {
			t.Fatalf("browser URL = %q, want https://agentclash.dev/runs/r-42", gotURL)
		}
		if !strings.Contains(out, "https://agentclash.dev/runs/r-42") {
			t.Fatalf("URL not printed even after browser launch: %q", out)
		}
	})
}

func TestRunShowRendersSummaryWithRanking(t *testing.T) {
	withTempCwd(t, func(_ string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/runs/run-1": jsonHandler(200, map[string]any{
				"id":           "run-1",
				"name":         "refund-eval",
				"status":       "completed",
				"created_at":   "2026-04-24T10:00:00Z",
				"started_at":   "2026-04-24T10:00:05Z",
				"finished_at":  "2026-04-24T10:00:25Z",
				"workspace_id": "ws-1",
			}),
			"GET /v1/runs/run-1/ranking": jsonHandler(200, map[string]any{
				"state": "ready",
				"ranking": map[string]any{
					"items": []map[string]any{
						{
							"rank":              1,
							"label":             "claude-4.7",
							"status":            "completed",
							"composite_score":   0.91,
							"correctness_score": 0.95,
							"reliability_score": 0.90,
							"latency_score":     0.88,
							"cost_score":        0.87,
						},
						{
							"rank":              2,
							"label":             "gpt-5",
							"status":            "completed",
							"composite_score":   0.87,
							"correctness_score": 0.90,
						},
					},
				},
			}),
		})
		defer srv.Close()

		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"run", "show", "run-1"}, srv.URL); err != nil {
			t.Fatalf("run show: %v", err)
		}
		out := stdout.finish()
		for _, want := range []string{
			"Run run-1",
			"refund-eval",
			"completed",
			"20s",
			"claude-4.7",
			"gpt-5",
			"0.91",
			"0.87",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("run show output missing %q\n---\n%s", want, out)
			}
		}
	})
}

func TestRunShowHandlesMissingRankingGracefully(t *testing.T) {
	// Run is still running; ranking endpoint returns 202. run show must
	// print the run block and a soft "not available yet" note — never fail.
	withTempCwd(t, func(_ string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/runs/r-pending": jsonHandler(200, map[string]any{
				"id": "r-pending", "status": "running",
			}),
			"GET /v1/runs/r-pending/ranking": jsonHandler(http.StatusAccepted, map[string]any{
				"state": "pending", "message": "ranking not ready",
			}),
		})
		defer srv.Close()

		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"run", "show", "r-pending"}, srv.URL); err != nil {
			t.Fatalf("run show: %v", err)
		}
		out := stdout.finish()
		if !strings.Contains(out, "r-pending") || !strings.Contains(out, "running") {
			t.Fatalf("missing run block: %s", out)
		}
		if !strings.Contains(out, "Ranking not available yet") {
			t.Fatalf("expected soft 'not available' note; got: %s", out)
		}
	})
}

func TestRunShowFailsOnMissingRun(t *testing.T) {
	withTempCwd(t, func(_ string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/runs/nope": jsonHandler(404, map[string]any{
				"error": map[string]any{"code": "not_found", "message": "run not found"},
			}),
		})
		defer srv.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{"run", "show", "nope"}, srv.URL)
		if err == nil {
			t.Fatal("expected error for missing run")
		}
	})
}
