package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type clientCluster struct {
	clientset kubernetes.Interface
	restConfig *rest.Config
}

func newClientCluster(config Config) (*clientCluster, error) {
	restConfig, err := loadRESTConfig(config.Kubeconfig)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}
	return &clientCluster{clientset: cs, restConfig: restConfig}, nil
}

func loadRESTConfig(kubeconfigPath string) (*rest.Config, error) {
	if strings.TrimSpace(kubeconfigPath) != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
}

func (c *clientCluster) CreatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
}

func (c *clientCluster) WaitPodReady(ctx context.Context, namespace, name string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + name,
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	if ready, err := c.podReady(ctx, namespace, name); err != nil {
		return err
	} else if ready {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for pod %s/%s ready: %w", namespace, name, ctx.Err())
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch closed waiting for pod %s/%s", namespace, name)
			}
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			if pod.Status.Phase == corev1.PodFailed {
				return fmt.Errorf("pod %s/%s failed: %s", namespace, name, pod.Status.Reason)
			}
			if event.Type == watch.Deleted {
				return fmt.Errorf("pod %s/%s deleted before ready", namespace, name)
			}
			if isPodReady(pod) {
				return nil
			}
		}
	}
}

func (c *clientCluster) podReady(ctx context.Context, namespace, name string) (bool, error) {
	pod, err := c.GetPod(ctx, namespace, name)
	if err != nil {
		return false, err
	}
	return isPodReady(pod), nil
}

func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (c *clientCluster) DeletePod(ctx context.Context, namespace, name string) error {
	propagation := metav1.DeletePropagationForeground
	err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if apierrors.IsNotFound(err) {
		return err
	}
	return err
}

func (c *clientCluster) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *clientCluster) CreateNetworkPolicy(ctx context.Context, policy *networkingv1.NetworkPolicy) error {
	_, err := c.clientset.NetworkingV1().NetworkPolicies(policy.Namespace).Create(ctx, policy, metav1.CreateOptions{})
	return err
}

func (c *clientCluster) DeleteNetworkPolicy(ctx context.Context, namespace, name string) error {
	err := c.clientset.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return err
	}
	return err
}

func (c *clientCluster) Exec(ctx context.Context, namespace, podName string, opts execOptions) (execResult, error) {
	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	command := opts.Command
	// remotecommand has no working-dir / env fields; wrap once when needed.
	needWrap := (strings.TrimSpace(opts.WorkDir) != "" && opts.WorkDir != "/") || len(opts.Env) > 0
	if needWrap {
		var prefix []string
		for _, e := range opts.Env {
			// e is KEY=VAL
			if eq := strings.IndexByte(e, '='); eq > 0 {
				prefix = append(prefix, "export "+shellQuote(e[:eq])+"="+shellQuote(e[eq+1:]))
			}
		}
		body := shellJoin(command)
		if strings.TrimSpace(opts.WorkDir) != "" && opts.WorkDir != "/" {
			body = "cd " + shellQuote(opts.WorkDir) + " && " + body
		}
		if len(prefix) > 0 {
			body = strings.Join(prefix, "; ") + "; " + body
		}
		command = []string{"sh", "-c", body}
	}

	execOpts := &corev1.PodExecOptions{
		Container: "sandbox",
		Command:   command,
		Stdin:     opts.Stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}
	req.VersionedParams(execOpts, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(c.restConfig, "POST", req.URL())
	if err != nil {
		return execResult{}, err
	}

	var stdout, stderr io.Writer = opts.Stdout, opts.Stderr
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	// Capture stderr for exit diagnosis when callers pass buffers.
	var stderrBuf bytes.Buffer
	stderr = io.MultiWriter(stderr, &stderrBuf)

	streamErr := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  opts.Stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	})
	// SPDY executor returns nil on success even when remote exit != 0; exit codes
	// require the CodeExitError unwrap below.
	if streamErr != nil {
		if code, ok := extractExitCode(streamErr); ok {
			return execResult{ExitCode: code}, nil
		}
		if pod, getErr := c.GetPod(ctx, namespace, podName); getErr == nil && pod.Status.Reason == "DeadlineExceeded" {
			return execResult{}, fmt.Errorf("pod DeadlineExceeded: %w", streamErr)
		}
		return execResult{}, streamErr
	}
	return execResult{ExitCode: 0}, nil
}

func extractExitCode(err error) (int, bool) {
	type exitCoder interface {
		ExitStatus() int
	}
	if e, ok := err.(exitCoder); ok {
		return e.ExitStatus(), true
	}
	// remotecommand.CodeExitError
	type codeExit interface {
		error
		ExitStatus() int
	}
	if e, ok := err.(codeExit); ok {
		return e.ExitStatus(), true
	}
	if err != nil && strings.Contains(err.Error(), "command terminated with exit code") {
		// Fallback parse: "... exit code N"
		fields := strings.Fields(err.Error())
		if n := len(fields); n > 0 {
			var code int
			if _, scanErr := fmt.Sscanf(fields[n-1], "%d", &code); scanErr == nil {
				return code, true
			}
		}
	}
	return 0, false
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func shellJoin(command []string) string {
	parts := make([]string, len(command))
	for i, c := range command {
		parts[i] = shellQuote(c)
	}
	return strings.Join(parts, " ")
}
