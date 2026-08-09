package kubernetes

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agentclash/agentclash/runtime/sandbox"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type session struct {
	mu                 sync.Mutex
	cluster            cluster
	namespace          string
	podName            string
	policyName         string
	sandboxID          string
	allowShell         bool
	workingDirectory   string
	defaultEnvironment map[string]string
	destroyed          bool
}

func (s *session) ID() string { return s.sandboxID }

func (s *session) UploadFile(ctx context.Context, filePath string, content []byte) error {
	return s.WriteFile(ctx, filePath, content)
}

func (s *session) DownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	return s.ReadFile(ctx, filePath)
}

func (s *session) WriteFile(ctx context.Context, filePath string, content []byte) error {
	if err := s.ensureActive(); err != nil {
		return err
	}
	cleaned := normalizeAbsPath(filePath)
	dir := path.Dir(cleaned)
	if err := s.mkdirp(ctx, dir); err != nil {
		return err
	}
	archive, err := writeTarFile(path.Base(cleaned), content)
	if err != nil {
		return err
	}
	// kubectl cp pattern: tar extract into parent directory.
	result, err := s.execRaw(ctx, []string{"tar", "-xmf", "-", "-C", dir}, bytes.NewReader(archive), nil)
	if err != nil {
		return fmt.Errorf("upload tar: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("upload tar exited %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}

func (s *session) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	if err := s.ensureActive(); err != nil {
		return nil, err
	}
	cleaned := normalizeAbsPath(filePath)
	var stdout bytes.Buffer
	result, err := s.execRaw(ctx, []string{"tar", "-cf", "-", "-C", path.Dir(cleaned), path.Base(cleaned)}, nil, &stdout)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		if strings.Contains(result.Stderr, "No such file") || strings.Contains(result.Stderr, "Cannot stat") {
			return nil, sandbox.ErrFileNotFound
		}
		return nil, fmt.Errorf("download tar exited %d: %s", result.ExitCode, result.Stderr)
	}
	content, err := readTarFile(&stdout)
	if err != nil {
		return nil, sandbox.ErrFileNotFound
	}
	return content, nil
}

func (s *session) ListFiles(ctx context.Context, prefix string) ([]sandbox.FileInfo, error) {
	if err := s.ensureActive(); err != nil {
		return nil, err
	}
	search := strings.TrimSpace(prefix)
	if search == "" {
		search = s.workingDirectory
	}
	search = normalizeAbsPath(search)
	result, err := s.Exec(ctx, sandbox.ExecRequest{
		Command: []string{"find", search, "-type", "f", "-printf", "%p\t%s\n"},
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		if strings.Contains(result.Stderr, "No such file or directory") {
			return nil, sandbox.ErrFileNotFound
		}
		return nil, fmt.Errorf("find exited %d: %s", result.ExitCode, result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) == "" {
		return []sandbox.FileInfo{}, nil
	}
	var items []sandbox.FileInfo
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		size, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		items = append(items, sandbox.FileInfo{Path: strings.TrimSpace(parts[0]), Size: size})
	}
	return items, nil
}

func (s *session) Exec(ctx context.Context, request sandbox.ExecRequest) (sandbox.ExecResult, error) {
	if err := s.ensureActive(); err != nil {
		return sandbox.ExecResult{}, err
	}
	if len(request.Command) == 0 {
		return sandbox.ExecResult{}, fmt.Errorf("exec command is required")
	}
	if !s.allowShell && sandbox.IsShellCommand(request.Command) {
		return sandbox.ExecResult{}, sandbox.ErrShellNotAllowed
	}

	command := request.Command
	execCtx := ctx
	cancel := func() {}
	if request.Timeout > 0 {
		command = append([]string{"timeout", "-k", "2", strconv.Itoa(ceilSeconds(request.Timeout))}, command...)
		execCtx, cancel = context.WithTimeout(ctx, request.Timeout+5*time.Second)
	}
	defer cancel()

	var stdout, stderr bytes.Buffer
	stdoutW := io.Writer(&stdout)
	stderrW := io.Writer(&stderr)
	if request.OnStdout != nil {
		stdoutW = &callbackWriter{cb: request.OnStdout, buf: &stdout}
	}
	if request.OnStderr != nil {
		stderrW = &callbackWriter{cb: request.OnStderr, buf: &stderr}
	}

	workDir := strings.TrimSpace(request.WorkingDirectory)
	if workDir == "" {
		workDir = s.workingDirectory
	}
	env := mergeEnv(s.defaultEnvironment, request.Environment)

	res, err := s.cluster.Exec(execCtx, s.namespace, s.podName, execOptions{
		Command: command,
		Stdout:  stdoutW,
		Stderr:  stderrW,
		Env:     envSlice(env),
		WorkDir: workDir,
	})
	if err != nil {
		if isDeadlineExceeded(err) || isPodDeadline(err) {
			return sandbox.ExecResult{}, fmt.Errorf("sandbox exec timed out: %w", err)
		}
		return sandbox.ExecResult{}, err
	}
	return sandbox.ExecResult{
		ExitCode: res.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func (s *session) Destroy(ctx context.Context) error {
	s.mu.Lock()
	if s.destroyed {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	var result error
	if err := s.cluster.DeletePod(ctx, s.namespace, s.podName); err != nil {
		if apierrors.IsNotFound(err) || isNotFound(err) {
			result = sandbox.ErrSandboxNotFound
		} else {
			return err
		}
	}
	if err := s.cluster.DeleteNetworkPolicy(ctx, s.namespace, s.policyName); err != nil && !apierrors.IsNotFound(err) && !isNotFound(err) {
		return err
	}
	s.markDestroyed()
	return result
}

func (s *session) markDestroyed() {
	s.mu.Lock()
	s.destroyed = true
	s.mu.Unlock()
}

func (s *session) mkdirp(ctx context.Context, dir string) error {
	result, err := s.execRaw(ctx, []string{"mkdir", "-p", dir}, nil, nil)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("mkdir exited %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}

func (s *session) execRaw(ctx context.Context, command []string, stdin io.Reader, stdout *bytes.Buffer) (sandbox.ExecResult, error) {
	var stderr bytes.Buffer
	if stdout == nil {
		stdout = &bytes.Buffer{}
	}
	res, err := s.cluster.Exec(ctx, s.namespace, s.podName, execOptions{
		Command: command,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  &stderr,
		WorkDir: s.workingDirectory,
	})
	if err != nil {
		return sandbox.ExecResult{}, err
	}
	return sandbox.ExecResult{ExitCode: res.ExitCode, Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func (s *session) ensureActive() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.destroyed {
		return sandbox.ErrSessionDestroyed
	}
	return nil
}

type callbackWriter struct {
	cb  func([]byte) error
	buf *bytes.Buffer
}

func (w *callbackWriter) Write(p []byte) (int, error) {
	if w.buf != nil {
		_, _ = w.buf.Write(p)
	}
	if w.cb != nil {
		if err := w.cb(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func normalizeAbsPath(raw string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(raw))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}


func ceilSeconds(d time.Duration) int {
	sec := int(d / time.Second)
	if d%time.Second != 0 {
		sec++
	}
	if sec < 1 {
		sec = 1
	}
	return sec
}

func mergeEnv(base, override map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func envSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

func writeTarFile(name string, content []byte) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readTarFile(r io.Reader) ([]byte, error) {
	tr := tar.NewReader(r)
	hdr, err := tr.Next()
	if err != nil {
		return nil, err
	}
	if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != 0 {
		return nil, fmt.Errorf("unexpected tar entry type %v", hdr.Typeflag)
	}
	return io.ReadAll(tr)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found") || apierrors.IsNotFound(err)
}

func isDeadlineExceeded(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "context deadline"))
}

func isPodDeadline(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "deadlineexceeded")
}
