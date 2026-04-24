package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/agentclash/agentclash/cli/internal/config"
)

// withTempCwd runs fn inside a fresh temp directory that becomes the process
// working dir, with every AGENTCLASH_* env var that could inject a default
// scrubbed. Mirrors what a real user has when they cd into a fresh project
// and run `agentclash link` from an empty shell.
func withTempCwd(t *testing.T, fn func(dir string)) {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// Isolate user config / env. Without this, a dev running the tests with
	// ~/.config/agentclash/config.yaml containing a real workspace id makes
	// link skip the picker and hit a URL the fake server doesn't know.
	xdg := filepath.Join(dir, ".xdg")
	if err := os.MkdirAll(xdg, 0o755); err != nil {
		t.Fatalf("mkdir xdg: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("AGENTCLASH_API_URL", "")
	t.Setenv("AGENTCLASH_WORKSPACE", "")
	t.Setenv("AGENTCLASH_ORG", "")
	t.Setenv("AGENTCLASH_DEV", "")
	t.Setenv("AGENTCLASH_DEV_USER_ID", "")
	t.Setenv("CI", "")

	fn(dir)
}

// readProjectYAML parses .agentclash.yaml written by link.
func readProjectYAML(t *testing.T, dir string) config.ProjectConfig {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, config.ProjectConfigFile))
	if err != nil {
		t.Fatalf("read project config: %v", err)
	}
	var cfg config.ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal project config: %v", err)
	}
	return cfg
}

// lockPickerForTest swaps the package picker for the duration of the test.
// Passing nil restores the default. Also suppresses isInteractiveTerminal to
// match the picker state — tests cover the branching logic directly via the
// explicit pickers.
func lockPickerForTest(t *testing.T, p interactivePicker, interactive bool) {
	t.Helper()
	origPicker := newInteractivePicker
	origInteractive := isInteractiveTerminal
	newInteractivePicker = func() interactivePicker { return p }
	isInteractiveTerminal = func(_ *RunContext) bool { return interactive }
	t.Cleanup(func() {
		newInteractivePicker = origPicker
		isInteractiveTerminal = origInteractive
	})
}

// fakeDeploymentsResponse is the minimal shape link's fetchDeployments reads.
func fakeDeploymentsResponse(items ...map[string]any) map[string]any {
	return map[string]any{"items": items}
}

func TestLinkHappyPathSingleWorkspaceSingleOrg(t *testing.T) {
	withTempCwd(t, func(dir string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/organizations": jsonHandler(200, map[string]any{
				"items": []map[string]any{
					{"id": "org-1", "name": "Personal", "slug": "personal"},
				},
			}),
			"GET /v1/organizations/org-1/workspaces": jsonHandler(200, map[string]any{
				"items": []map[string]any{
					{"id": "ws-1", "name": "raj-personal", "slug": "raj-personal"},
				},
			}),
			"GET /v1/workspaces/ws-1/details": jsonHandler(200, map[string]any{
				"id":              "ws-1",
				"name":            "raj-personal",
				"organization_id": "org-1",
			}),
			"GET /v1/workspaces/ws-1/agent-deployments": jsonHandler(200, fakeDeploymentsResponse(
				map[string]any{"id": "dep-a", "name": "gpt-5"},
				map[string]any{"id": "dep-b", "name": "claude-4.7"},
				map[string]any{"id": "dep-c", "name": "gemini-3"},
			)),
		})
		defer srv.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"link"}, srv.URL); err != nil {
			t.Fatalf("link: %v", err)
		}

		got := readProjectYAML(t, dir)
		if got.WorkspaceID != "ws-1" {
			t.Fatalf("workspace_id = %q, want ws-1", got.WorkspaceID)
		}
		if got.WorkspaceName != "raj-personal" {
			t.Fatalf("workspace_name = %q, want raj-personal", got.WorkspaceName)
		}
		if got.OrgID != "org-1" {
			t.Fatalf("org_id = %q, want org-1", got.OrgID)
		}
		wantDeployments := map[string]string{"gpt-5": "dep-a", "claude-4.7": "dep-b", "gemini-3": "dep-c"}
		if len(got.Deployments) != len(wantDeployments) {
			t.Fatalf("deployments = %v, want %v", got.Deployments, wantDeployments)
		}
		for k, v := range wantDeployments {
			if got.Deployments[k] != v {
				t.Fatalf("deployments[%q] = %q, want %q", k, got.Deployments[k], v)
			}
		}

		agentsMd, err := os.ReadFile(filepath.Join(dir, AgentsMdFile))
		if err != nil {
			t.Fatalf("read AGENTS.md: %v", err)
		}
		for _, want := range []string{"raj-personal", "gpt-5", "claude-4.7", "gemini-3"} {
			if !strings.Contains(string(agentsMd), want) {
				t.Fatalf("AGENTS.md missing %q\n---\n%s", want, agentsMd)
			}
		}
	})
}

func TestLinkWorkspaceFlagSkipsPicker(t *testing.T) {
	// Passing -w should bypass the org-list/workspace-list round trip
	// entirely. We assert by NOT registering those handlers: any hit would
	// return 404 and the test would fail.
	withTempCwd(t, func(dir string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/workspaces/ws-explicit/details": jsonHandler(200, map[string]any{
				"id": "ws-explicit", "name": "chosen", "organization_id": "org-x",
			}),
			"GET /v1/workspaces/ws-explicit/agent-deployments": jsonHandler(200, fakeDeploymentsResponse()),
		})
		defer srv.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"link", "-w", "ws-explicit"}, srv.URL); err != nil {
			t.Fatalf("link -w: %v", err)
		}
		got := readProjectYAML(t, dir)
		if got.WorkspaceID != "ws-explicit" {
			t.Fatalf("workspace_id = %q, want ws-explicit", got.WorkspaceID)
		}
	})
}

func TestLinkMultiWorkspaceNonInteractiveErrorsWithoutFlag(t *testing.T) {
	withTempCwd(t, func(dir string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/organizations": jsonHandler(200, map[string]any{
				"items": []map[string]any{
					{"id": "org-1", "name": "A"},
					{"id": "org-2", "name": "B"},
				},
			}),
			"GET /v1/organizations/org-1/workspaces": jsonHandler(200, map[string]any{
				"items": []map[string]any{{"id": "ws-1", "name": "one"}},
			}),
			"GET /v1/organizations/org-2/workspaces": jsonHandler(200, map[string]any{
				"items": []map[string]any{{"id": "ws-2", "name": "two"}},
			}),
		})
		defer srv.Close()

		lockPickerForTest(t, &errPicker{err: nil}, false) // non-interactive

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		err := executeCommand(t, []string{"link"}, srv.URL)
		if err == nil {
			t.Fatal("expected error in non-interactive multi-workspace state, got nil")
		}
		if !strings.Contains(err.Error(), "workspaces") {
			t.Fatalf("expected workspace-count error, got: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(dir, config.ProjectConfigFile)); statErr == nil {
			t.Fatalf(".agentclash.yaml was written despite failure — link must bail before any filesystem side effect")
		}
		if _, statErr := os.Stat(filepath.Join(dir, AgentsMdFile)); statErr == nil {
			t.Fatal("AGENTS.md was written despite failure")
		}
	})
}

func TestLinkZeroDeploymentsStillWritesConfigWithWarning(t *testing.T) {
	withTempCwd(t, func(dir string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/workspaces/ws-empty/details": jsonHandler(200, map[string]any{
				"id": "ws-empty", "name": "empty", "organization_id": "org-x",
			}),
			"GET /v1/workspaces/ws-empty/agent-deployments": jsonHandler(200, fakeDeploymentsResponse()),
		})
		defer srv.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"link", "-w", "ws-empty"}, srv.URL); err != nil {
			t.Fatalf("link: %v", err)
		}
		got := readProjectYAML(t, dir)
		if got.WorkspaceID != "ws-empty" {
			t.Fatalf("workspace_id = %q, want ws-empty", got.WorkspaceID)
		}
		if len(got.Deployments) != 0 {
			t.Fatalf("deployments = %v, want empty", got.Deployments)
		}

		agentsMd, err := os.ReadFile(filepath.Join(dir, AgentsMdFile))
		if err != nil {
			t.Fatalf("read AGENTS.md: %v", err)
		}
		if !strings.Contains(string(agentsMd), "none yet") {
			t.Fatalf("AGENTS.md should call out zero deployments: %s", agentsMd)
		}
		if !strings.Contains(string(agentsMd), "https://agentclash.dev/workspaces/ws-empty/deployments") {
			t.Fatalf("AGENTS.md should link to the deploy URL: %s", agentsMd)
		}
	})
}

func TestLinkIsReadOnly(t *testing.T) {
	// Any write method (POST/PATCH/PUT/DELETE) hitting the fake server is
	// a bug — link must be strictly read-only. We enforce that by wrapping
	// the handler and blowing up on non-GET.
	withTempCwd(t, func(_ string) {
		var writes int32
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/workspaces/ws-ro/details": func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					atomic.AddInt32(&writes, 1)
				}
				jsonHandler(200, map[string]any{"id": "ws-ro", "name": "ro", "organization_id": "o"})(w, r)
			},
			"GET /v1/workspaces/ws-ro/agent-deployments": func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					atomic.AddInt32(&writes, 1)
				}
				jsonHandler(200, fakeDeploymentsResponse(map[string]any{"id": "d", "name": "m"}))(w, r)
			},
		})
		defer srv.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"link", "-w", "ws-ro"}, srv.URL); err != nil {
			t.Fatalf("link: %v", err)
		}
		if atomic.LoadInt32(&writes) != 0 {
			t.Fatalf("link made %d non-GET requests — must be read-only", writes)
		}
	})
}

func TestLinkStructuredOutputShapesJSON(t *testing.T) {
	withTempCwd(t, func(_ string) {
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"GET /v1/workspaces/ws-j/details": jsonHandler(200, map[string]any{
				"id": "ws-j", "name": "jsn", "organization_id": "o",
			}),
			"GET /v1/workspaces/ws-j/agent-deployments": jsonHandler(200, fakeDeploymentsResponse(
				map[string]any{"id": "d1", "name": "gpt-5"},
			)),
		})
		defer srv.Close()

		stdout := captureStdout(t)
		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"link", "-w", "ws-j", "--json"}, srv.URL); err != nil {
			t.Fatalf("link: %v", err)
		}
		out := stdout.finish()
		for _, want := range []string{
			"\"workspace_id\": \"ws-j\"",
			"\"workspace_name\": \"jsn\"",
			"\"org_id\": \"o\"",
			"\"gpt-5\": \"d1\"",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("--json output missing %q\n---\n%s", want, out)
			}
		}
	})
}
