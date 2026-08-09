// Package kubernetes implements runtime/sandbox.Provider using Kubernetes Pods.
//
// Network isolation uses standard NetworkPolicy (default-deny egress). DNS
// hostnames in NetworkAllowlist cannot be expressed as NetworkPolicy peers;
// use CIDR/port entries or an egress proxy (see docs/deployment/k8s-sandbox.md).
//
// Reaper-friendly labels on every pod/policy:
//   - agentclash.dev/managed-by=runtime-sandbox-kubernetes
//   - agentclash.dev/run-id
//   - agentclash.dev/run-agent-id
//   - agentclash.dev/sandbox-id
package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/agentclash/agentclash/runtime/maputil"
	"github.com/agentclash/agentclash/runtime/sandbox"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const packageInstallTimeout = 120 * time.Second

// Provider creates Kubernetes-backed sandbox sessions.
type Provider struct {
	cluster cluster
	config  Config
	owns    bool
	logger  *slog.Logger
}

// NewProvider builds a provider using kubeconfig / in-cluster config.
func NewProvider(config Config) (*Provider, error) {
	cl, err := newClientCluster(config)
	if err != nil {
		return nil, err
	}
	return &Provider{cluster: cl, config: config, owns: true, logger: config.logger()}, nil
}

// NewProviderWithCluster injects a cluster implementation (tests).
func NewProviderWithCluster(cl cluster, config Config) *Provider {
	return &Provider{cluster: cl, config: config, owns: false, logger: config.logger()}
}

// Close is a no-op for the Kubernetes client (shared REST config).
func (p *Provider) Close() error { return nil }

func (p *Provider) Create(ctx context.Context, request sandbox.CreateRequest) (sandbox.Session, error) {
	startedAt := time.Now()
	sandboxID := uuid.NewString()
	podName := podNameFor(request.RunAgentID, sandboxID)
	namespace := p.config.namespace()
	image := p.config.resolveImage(request.TemplateID)

	labels := map[string]string{
		labelManagedBy:  labelManagedByValue,
		labelRunID:      request.RunID.String(),
		labelRunAgentID: request.RunAgentID.String(),
		labelSandboxID:  sandboxID,
	}
	for k, v := range request.Labels {
		if strings.HasPrefix(k, "agentclash.dev/") {
			continue // reserved
		}
		labels[k] = v
	}

	var deadline *int64
	if request.Timeout > 0 {
		seconds := int64(request.Timeout.Round(time.Second) / time.Second)
		if seconds < 1 {
			seconds = 1
		}
		deadline = &seconds
	}

	allowNetwork := request.ToolPolicy.AllowNetwork
	policy, hostnames, err := buildNetworkPolicy(namespace, podName+"-egress", sandboxID, allowNetwork, request.NetworkAllowlist, p.config.dnsPolicy())
	if err != nil {
		return nil, err
	}
	for _, host := range hostnames {
		p.logger.Warn("kubernetes sandbox: DNS allowlist entry ignored by NetworkPolicy; use CIDR or an egress proxy",
			"hostname", host, "sandbox_id", sandboxID)
	}
	if err := p.cluster.CreateNetworkPolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("create network policy: %w", err)
	}

	automountServiceAccountToken := false
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                corev1.RestartPolicyNever,
			ActiveDeadlineSeconds:        deadline,
			ServiceAccountName:           p.config.ServiceAccountName,
			AutomountServiceAccountToken: &automountServiceAccountToken,
			Containers: []corev1.Container{{
				Name:            "sandbox",
				Image:           image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"sleep", "infinity"},
				WorkingDir:      workingDir(request),
				Env:             envVars(request.EnvVars),
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(p.config.cpuRequest()),
						corev1.ResourceMemory: resource.MustParse(p.config.memoryRequest()),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(p.config.cpuLimit()),
						corev1.ResourceMemory: resource.MustParse(p.config.memoryLimit()),
					},
				},
				SecurityContext: containerSecurityContext(p.config.RunAsNonRoot),
			}},
			SecurityContext: podSecurityContext(p.config.RunAsNonRoot),
		},
	}

	created, err := p.cluster.CreatePod(ctx, pod)
	if err != nil {
		_ = p.cluster.DeleteNetworkPolicy(context.WithoutCancel(ctx), namespace, policy.Name)
		p.logger.Error("kubernetes sandbox create failed", "error", err, "duration", time.Since(startedAt))
		return nil, err
	}

	if err := p.cluster.WaitPodReady(ctx, namespace, created.Name, p.config.readyTimeout()); err != nil {
		_ = p.cluster.DeletePod(context.WithoutCancel(ctx), namespace, created.Name)
		_ = p.cluster.DeleteNetworkPolicy(context.WithoutCancel(ctx), namespace, policy.Name)
		return nil, fmt.Errorf("wait for pod ready: %w", err)
	}

	sess := &session{
		cluster:            p.cluster,
		namespace:          namespace,
		podName:            created.Name,
		policyName:         policy.Name,
		sandboxID:          sandboxID,
		allowShell:         request.ToolPolicy.AllowShell,
		workingDirectory:   workingDir(request),
		defaultEnvironment: maputil.CloneStringMap(request.EnvVars),
	}

	if len(request.AdditionalPackages) > 0 {
		if err := p.installAdditionalPackages(ctx, sess, request.AdditionalPackages); err != nil {
			_ = sess.Destroy(context.WithoutCancel(ctx))
			return nil, err
		}
	}

	p.logger.Info("kubernetes sandbox created",
		"sandbox_id", sandboxID, "pod", created.Name, "namespace", namespace,
		"image", image, "duration", time.Since(startedAt))
	return sess, nil
}

func (p *Provider) installAdditionalPackages(ctx context.Context, sess *session, packages []string) error {
	installCmd := "apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends " + strings.Join(packages, " ")
	result, err := sess.Exec(ctx, sandbox.ExecRequest{
		Command: []string{"sh", "-c", installCmd},
		Timeout: packageInstallTimeout,
	})
	if err != nil {
		return fmt.Errorf("additional packages install: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("additional packages install failed: exit=%d stderr=%s", result.ExitCode, result.Stderr)
	}
	return nil
}

func workingDir(request sandbox.CreateRequest) string {
	if strings.TrimSpace(request.Filesystem.WorkingDirectory) != "" {
		return strings.TrimSpace(request.Filesystem.WorkingDirectory)
	}
	return defaultWorkingDirectory
}

func envVars(vars map[string]string) []corev1.EnvVar {
	if len(vars) == 0 {
		return nil
	}
	out := make([]corev1.EnvVar, 0, len(vars))
	for k, v := range vars {
		out = append(out, corev1.EnvVar{Name: k, Value: v})
	}
	return out
}

func podNameFor(runAgentID uuid.UUID, sandboxID string) string {
	// DNS-1123: max 63 chars. Prefix + short ids.
	shortAgent := strings.ReplaceAll(runAgentID.String(), "-", "")[:8]
	shortSand := strings.ReplaceAll(sandboxID, "-", "")[:8]
	return fmt.Sprintf("ac-sbx-%s-%s", shortAgent, shortSand)
}

func podSecurityContext(runAsNonRoot bool) *corev1.PodSecurityContext {
	seccomp := corev1.SeccompProfileTypeRuntimeDefault
	ctx := &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{Type: seccomp},
	}
	if runAsNonRoot {
		uid := int64(65532)
		ctx.RunAsNonRoot = boolPtr(true)
		ctx.RunAsUser = &uid
		ctx.RunAsGroup = &uid
	}
	return ctx
}

func containerSecurityContext(runAsNonRoot bool) *corev1.SecurityContext {
	allowPriv := false
	ctx := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPriv,
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}
	if runAsNonRoot {
		ctx.RunAsNonRoot = boolPtr(true)
		uid := int64(65532)
		ctx.RunAsUser = &uid
	}
	return ctx
}

func boolPtr(v bool) *bool { return &v }
