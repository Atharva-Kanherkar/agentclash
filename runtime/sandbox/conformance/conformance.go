// Package conformance defines behavioral tests every sandbox.Provider must pass.
package conformance

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agentclash/agentclash/runtime/sandbox"
	"github.com/google/uuid"
)

// Options tunes the suite for providers that need longer boot times.
type Options struct {
	CreateTimeout time.Duration
	SkipIsolation bool // set when the provider cannot enforce network isolation in-process
}

// Run exercises create / files / exec / destroy / sentinel errors against provider.
func Run(t *testing.T, provider sandbox.Provider, opts Options) {
	t.Helper()
	if provider == nil {
		t.Fatal("provider is nil")
	}
	createTimeout := opts.CreateTimeout
	if createTimeout <= 0 {
		createTimeout = 2 * time.Minute
	}

	t.Run("lifecycle_files_exec", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), createTimeout)
		defer cancel()
		session, err := provider.Create(ctx, sandbox.CreateRequest{
			RunID:      uuid.New(),
			RunAgentID: uuid.New(),
			Timeout:    createTimeout,
			ToolPolicy: sandbox.ToolPolicy{AllowShell: true, AllowNetwork: false},
			Filesystem: sandbox.FilesystemSpec{WorkingDirectory: "/workspace"},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer func() {
			if destroyErr := session.Destroy(context.Background()); destroyErr != nil && !errors.Is(destroyErr, sandbox.ErrSandboxNotFound) {
				t.Errorf("Destroy: %v", destroyErr)
			}
		}()

		if session.ID() == "" {
			t.Fatal("session ID empty")
		}
		if err := session.UploadFile(ctx, "/workspace/a.txt", []byte("alpha")); err != nil {
			t.Fatalf("UploadFile: %v", err)
		}
		got, err := session.ReadFile(ctx, "/workspace/a.txt")
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(got) != "alpha" {
			t.Fatalf("ReadFile = %q", got)
		}
		if err := session.WriteFile(ctx, "/workspace/a.txt", []byte("beta")); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		got, err = session.DownloadFile(ctx, "/workspace/a.txt")
		if err != nil {
			t.Fatalf("DownloadFile: %v", err)
		}
		if string(got) != "beta" {
			t.Fatalf("DownloadFile = %q", got)
		}
		files, err := session.ListFiles(ctx, "/workspace")
		if err != nil {
			t.Fatalf("ListFiles: %v", err)
		}
		if len(files) == 0 {
			t.Fatal("ListFiles empty")
		}

		var streamed bytes.Buffer
		result, err := session.Exec(ctx, sandbox.ExecRequest{
			Command:  []string{"echo", "conformance"},
			OnStdout: func(b []byte) error { streamed.Write(b); return nil },
		})
		if err != nil {
			t.Fatalf("Exec: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("exit = %d stderr=%s", result.ExitCode, result.Stderr)
		}
		if !strings.Contains(result.Stdout, "conformance") && !strings.Contains(streamed.String(), "conformance") {
			t.Fatalf("stdout=%q streamed=%q", result.Stdout, streamed.String())
		}
	})

	t.Run("file_not_found", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), createTimeout)
		defer cancel()
		session, err := provider.Create(ctx, sandbox.CreateRequest{
			RunID: uuid.New(), RunAgentID: uuid.New(), Timeout: createTimeout,
			ToolPolicy: sandbox.ToolPolicy{AllowShell: true},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer session.Destroy(context.Background())
		_, err = session.ReadFile(ctx, "/workspace/missing-"+uuid.NewString())
		if !errors.Is(err, sandbox.ErrFileNotFound) {
			t.Fatalf("ReadFile missing = %v, want ErrFileNotFound", err)
		}
	})

	t.Run("shell_not_allowed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), createTimeout)
		defer cancel()
		session, err := provider.Create(ctx, sandbox.CreateRequest{
			RunID: uuid.New(), RunAgentID: uuid.New(), Timeout: createTimeout,
			ToolPolicy: sandbox.ToolPolicy{AllowShell: false},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		defer session.Destroy(context.Background())
		_, err = session.Exec(ctx, sandbox.ExecRequest{Command: []string{"sh", "-c", "echo hi"}})
		if !errors.Is(err, sandbox.ErrShellNotAllowed) {
			t.Fatalf("Exec shell = %v, want ErrShellNotAllowed", err)
		}
	})

	t.Run("destroy_idempotent_after_success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), createTimeout)
		defer cancel()
		session, err := provider.Create(ctx, sandbox.CreateRequest{
			RunID: uuid.New(), RunAgentID: uuid.New(), Timeout: createTimeout,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := session.Destroy(context.Background()); err != nil {
			t.Fatalf("Destroy: %v", err)
		}
		// Second destroy: providers may return nil (session already closed) or
		// ErrSandboxNotFound if the underlying resource is gone.
		if err := session.Destroy(context.Background()); err != nil && !errors.Is(err, sandbox.ErrSandboxNotFound) && !errors.Is(err, sandbox.ErrSessionDestroyed) {
			t.Fatalf("second Destroy: %v", err)
		}
	})
}
