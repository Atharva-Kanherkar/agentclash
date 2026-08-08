package kubernetes

import (
	"context"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// cluster is the Kubernetes API surface used by the provider. Tests inject fakes.
type cluster interface {
	CreatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error)
	WaitPodReady(ctx context.Context, namespace, name string, timeout time.Duration) error
	DeletePod(ctx context.Context, namespace, name string) error
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)

	CreateNetworkPolicy(ctx context.Context, policy *networkingv1.NetworkPolicy) error
	DeleteNetworkPolicy(ctx context.Context, namespace, name string) error

	Exec(ctx context.Context, namespace, podName string, opts execOptions) (execResult, error)
}

type execOptions struct {
	Command []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Env     []string
	WorkDir string
}

type execResult struct {
	ExitCode int
}
