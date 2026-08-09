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

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fakeCluster is an in-memory cluster for unit and conformance tests.
type fakeCluster struct {
	mu              sync.Mutex
	pods            map[string]*fakePod // namespace/name
	policies        map[string]*networkingv1.NetworkPolicy
	deletePodErr    error
	deletePolicyErr error
}

type fakePod struct {
	pod       *corev1.Pod
	files     map[string][]byte
	deadline  time.Time
	createdAt time.Time
}

// NewFakeCluster returns an in-memory cluster for unit and conformance tests.
func NewFakeCluster() *fakeCluster {
	return &fakeCluster{
		pods:     make(map[string]*fakePod),
		policies: make(map[string]*networkingv1.NetworkPolicy),
	}
}

// newFakeCluster is an alias kept for tests in this package.
func newFakeCluster() *fakeCluster { return NewFakeCluster() }

func (c *fakeCluster) key(ns, name string) string { return ns + "/" + name }

func (c *fakeCluster) CreatePod(_ context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.key(pod.Namespace, pod.Name)
	if _, ok := c.pods[key]; ok {
		return nil, apierrors.NewAlreadyExists(schema.GroupResource{Resource: "pods"}, pod.Name)
	}
	cloned := pod.DeepCopy()
	cloned.Status.Phase = corev1.PodRunning
	cloned.Status.Conditions = []corev1.PodCondition{{
		Type:   corev1.PodReady,
		Status: corev1.ConditionTrue,
	}}
	fp := &fakePod{
		pod:       cloned,
		files:     map[string][]byte{},
		createdAt: time.Now(),
	}
	if pod.Spec.ActiveDeadlineSeconds != nil {
		fp.deadline = time.Now().Add(time.Duration(*pod.Spec.ActiveDeadlineSeconds) * time.Second)
	}
	c.pods[key] = fp
	return cloned, nil
}

func (c *fakeCluster) WaitPodReady(ctx context.Context, namespace, name string, _ time.Duration) error {
	_, err := c.GetPod(ctx, namespace, name)
	return err
}

func (c *fakeCluster) DeletePod(_ context.Context, namespace, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.deletePodErr != nil {
		return c.deletePodErr
	}
	key := c.key(namespace, name)
	if _, ok := c.pods[key]; !ok {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
	}
	delete(c.pods, key)
	return nil
}

func (c *fakeCluster) GetPod(_ context.Context, namespace, name string) (*corev1.Pod, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fp, ok := c.pods[c.key(namespace, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
	}
	if !fp.deadline.IsZero() && time.Now().After(fp.deadline) {
		fp.pod.Status.Phase = corev1.PodFailed
		fp.pod.Status.Reason = "DeadlineExceeded"
		return nil, fmt.Errorf("pod %s DeadlineExceeded", name)
	}
	return fp.pod.DeepCopy(), nil
}

func (c *fakeCluster) CreateNetworkPolicy(_ context.Context, policy *networkingv1.NetworkPolicy) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policies[c.key(policy.Namespace, policy.Name)] = policy.DeepCopy()
	return nil
}

func (c *fakeCluster) DeleteNetworkPolicy(_ context.Context, namespace, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.deletePolicyErr != nil {
		return c.deletePolicyErr
	}
	key := c.key(namespace, name)
	if _, ok := c.policies[key]; !ok {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "networkpolicies"}, name)
	}
	delete(c.policies, key)
	return nil
}

func (c *fakeCluster) Exec(ctx context.Context, namespace, podName string, opts execOptions) (execResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fp, ok := c.pods[c.key(namespace, podName)]
	if !ok {
		return execResult{}, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, podName)
	}
	if !fp.deadline.IsZero() && time.Now().After(fp.deadline) {
		return execResult{}, fmt.Errorf("pod %s DeadlineExceeded", podName)
	}
	if err := ctx.Err(); err != nil {
		return execResult{}, err
	}
	return c.runCommand(fp, opts)
}

func (c *fakeCluster) runCommand(fp *fakePod, opts execOptions) (execResult, error) {
	cmd := opts.Command
	if len(cmd) == 0 {
		return execResult{ExitCode: 1}, nil
	}
	// Strip timeout wrapper used by session.Exec.
	if cmd[0] == "timeout" && len(cmd) >= 4 {
		cmd = cmd[3:]
	}
	writeOut := func(s string) {
		if opts.Stdout != nil && s != "" {
			_, _ = io.WriteString(opts.Stdout, s)
		}
	}
	writeErr := func(s string) {
		if opts.Stderr != nil && s != "" {
			_, _ = io.WriteString(opts.Stderr, s)
		}
	}

	switch cmd[0] {
	case "mkdir":
		// no-op for in-memory files; parents implied
		return execResult{ExitCode: 0}, nil
	case "tar":
		return c.fakeTar(fp, cmd, opts)
	case "find":
		return c.fakeFind(fp, cmd, opts)
	case "echo":
		writeOut(strings.Join(cmd[1:], " ") + "\n")
		return execResult{ExitCode: 0}, nil
	case "sh", "bash":
		if len(cmd) >= 3 && cmd[1] == "-c" {
			// Minimal support for package install script — succeed no-op.
			if strings.Contains(cmd[2], "apt-get") {
				return execResult{ExitCode: 0}, nil
			}
			if strings.HasPrefix(cmd[2], "echo ") {
				writeOut(strings.TrimPrefix(cmd[2], "echo ") + "\n")
				return execResult{ExitCode: 0}, nil
			}
		}
		writeErr("unsupported shell command in fake cluster\n")
		return execResult{ExitCode: 1}, nil
	case "curl", "wget":
		// Network isolation simulation: deny by default unless allowlisted CIDR
		// appears in a policy with open egress. For unit tests of NetworkPolicy
		// shape we don't execute curl against the fake; return exit 1.
		writeErr("connection refused (fake network deny)\n")
		return execResult{ExitCode: 1}, nil
	default:
		writeErr(fmt.Sprintf("fake cluster: unsupported command %q\n", cmd[0]))
		return execResult{ExitCode: 127}, nil
	}
}

func (c *fakeCluster) fakeTar(fp *fakePod, cmd []string, opts execOptions) (execResult, error) {
	// tar -xmf - -C dir  (extract)
	// tar -cf - -C dir file (create)
	if len(cmd) >= 5 && cmd[1] == "-xmf" {
		dir := cmd[4]
		if opts.Stdin == nil {
			return execResult{ExitCode: 1}, nil
		}
		tr := tar.NewReader(opts.Stdin)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return execResult{ExitCode: 1}, nil
			}
			content, _ := io.ReadAll(tr)
			fp.files[path.Join(dir, hdr.Name)] = content
		}
		return execResult{ExitCode: 0}, nil
	}
	if len(cmd) >= 5 && cmd[1] == "-cf" {
		dir := cmd[4]
		name := cmd[5]
		full := path.Join(dir, name)
		content, ok := fp.files[full]
		if !ok {
			if opts.Stderr != nil {
				_, _ = io.WriteString(opts.Stderr, "Cannot stat: No such file or directory\n")
			}
			return execResult{ExitCode: 2}, nil
		}
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))})
		_, _ = tw.Write(content)
		_ = tw.Close()
		if opts.Stdout != nil {
			_, _ = opts.Stdout.Write(buf.Bytes())
		}
		return execResult{ExitCode: 0}, nil
	}
	return execResult{ExitCode: 1}, nil
}

func (c *fakeCluster) fakeFind(fp *fakePod, cmd []string, opts execOptions) (execResult, error) {
	prefix := "/"
	if len(cmd) > 1 {
		prefix = cmd[1]
	}
	var b strings.Builder
	foundDir := false
	for p, content := range fp.files {
		if strings.HasPrefix(p, prefix) {
			foundDir = true
			b.WriteString(p)
			b.WriteByte('\t')
			b.WriteString(strconv.Itoa(len(content)))
			b.WriteByte('\n')
		}
	}
	// Empty prefix match with no files is ok; missing root dir with no files under it:
	if !foundDir && prefix != "/" && prefix != defaultWorkingDirectory {
		// Still OK if directory conceptually exists (we don't track dirs).
	}
	if opts.Stdout != nil {
		_, _ = io.WriteString(opts.Stdout, b.String())
	}
	return execResult{ExitCode: 0}, nil
}

// NetworkPolicy returns a stored policy (tests).
func (c *fakeCluster) NetworkPolicy(namespace, name string) *networkingv1.NetworkPolicy {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.policies[c.key(namespace, name)]
}

// Pod returns a stored pod (tests).
func (c *fakeCluster) Pod(namespace, name string) *corev1.Pod {
	c.mu.Lock()
	defer c.mu.Unlock()
	fp, ok := c.pods[c.key(namespace, name)]
	if !ok {
		return nil
	}
	return fp.pod.DeepCopy()
}