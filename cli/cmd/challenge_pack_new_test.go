package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChallengePackNewWritesStarterYAML(t *testing.T) {
	withTempCwd(t, func(dir string) {
		if err := executeCommand(t, []string{"challenge-pack", "new", "refund-handler"}, "http://unused"); err != nil {
			t.Fatalf("challenge-pack new: %v", err)
		}
		path := filepath.Join(dir, "refund-handler.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read scaffold: %v", err)
		}
		body := string(data)
		for _, want := range []string{
			"# refund-handler — agentclash challenge pack",
			"pack:",
			"slug: refund-handler",
			"name: refund-handler",
			"challenges:",
			"exact_match",
			"https://agentclash.dev/docs/challenge-packs",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("scaffold missing %q\n---\n%s", want, body)
			}
		}
	})
}

func TestChallengePackNewNormalizesNoisyNames(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"My Pack!", "my-pack"},
		{"  leading", "leading"},
		{"UPPER_snake-cased", "upper-snake-cased"},
		{"dots.and.slashes/v2", "dots-and-slashes-v2"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			withTempCwd(t, func(dir string) {
				if err := executeCommand(t, []string{"challenge-pack", "new", tc.in}, "http://unused"); err != nil {
					t.Fatalf("challenge-pack new %q: %v", tc.in, err)
				}
				if _, err := os.Stat(filepath.Join(dir, tc.want+".yaml")); err != nil {
					t.Fatalf("expected %s.yaml: %v", tc.want, err)
				}
			})
		})
	}
}

func TestChallengePackNewRejectsPunctuationOnlyName(t *testing.T) {
	withTempCwd(t, func(_ string) {
		err := executeCommand(t, []string{"challenge-pack", "new", "!!!"}, "http://unused")
		if err == nil {
			t.Fatal("expected error for punctuation-only name, got nil")
		}
		if !strings.Contains(err.Error(), "empty slug") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestChallengePackNewRefusesToOverwriteWithoutForce(t *testing.T) {
	withTempCwd(t, func(dir string) {
		if err := executeCommand(t, []string{"challenge-pack", "new", "twice"}, "http://unused"); err != nil {
			t.Fatalf("first new: %v", err)
		}
		originalBytes, _ := os.ReadFile(filepath.Join(dir, "twice.yaml"))
		if err := os.WriteFile(filepath.Join(dir, "twice.yaml"), []byte("# hand-edited by the user\n"), 0644); err != nil {
			t.Fatalf("simulate user edit: %v", err)
		}

		err := executeCommand(t, []string{"challenge-pack", "new", "twice"}, "http://unused")
		if err == nil {
			t.Fatal("expected error overwriting without --force, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("unexpected error: %v", err)
		}
		current, _ := os.ReadFile(filepath.Join(dir, "twice.yaml"))
		if string(current) != "# hand-edited by the user\n" {
			t.Fatalf("file was overwritten without --force: %s", current)
		}
		_ = originalBytes

		if err := executeCommand(t, []string{"challenge-pack", "new", "twice", "--force"}, "http://unused"); err != nil {
			t.Fatalf("--force new: %v", err)
		}
		final, _ := os.ReadFile(filepath.Join(dir, "twice.yaml"))
		if !strings.Contains(string(final), "pack:") {
			t.Fatalf("--force should rewrite scaffold, got: %s", final)
		}
	})
}

func TestChallengePackCheckIsAliasForValidate(t *testing.T) {
	// check must hit the same backend endpoint as validate — the only
	// observable difference is the command verb.
	withTempCwd(t, func(dir string) {
		packPath := filepath.Join(dir, "pack.yaml")
		if err := os.WriteFile(packPath, []byte("pack:\n  slug: x\n"), 0644); err != nil {
			t.Fatalf("write pack: %v", err)
		}

		var validateHits int
		srv := fakeAPI(t, map[string]http.HandlerFunc{
			"POST /v1/workspaces/ws-1/challenge-packs/validate": func(w http.ResponseWriter, r *http.Request) {
				validateHits++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{"valid":true}`))
			},
		})
		defer srv.Close()

		t.Setenv("AGENTCLASH_TOKEN", "test-tok")
		if err := executeCommand(t, []string{"challenge-pack", "check", packPath, "-w", "ws-1"}, srv.URL); err != nil {
			t.Fatalf("check: %v", err)
		}
		if validateHits != 1 {
			t.Fatalf("check should call /validate once; got %d calls", validateHits)
		}
	})
}
