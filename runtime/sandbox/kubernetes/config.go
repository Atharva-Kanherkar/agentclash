package kubernetes

import (
	"log/slog"
	"strings"
	"time"
)

const (
	defaultNamespace       = "agentclash-sandboxes"
	defaultImage           = "python:3.12-slim"
	defaultCPURequest      = "100m"
	defaultCPULimit        = "1"
	defaultMemoryRequest   = "128Mi"
	defaultMemoryLimit     = "1Gi"
	defaultReadyTimeout    = 2 * time.Minute
	labelManagedBy         = "agentclash.dev/managed-by"
	labelManagedByValue    = "runtime-sandbox-kubernetes"
	labelRunID             = "agentclash.dev/run-id"
	labelRunAgentID        = "agentclash.dev/run-agent-id"
	labelSandboxID         = "agentclash.dev/sandbox-id"
	defaultWorkingDirectory = "/workspace"
)

// Config controls the Kubernetes sandbox provider.
type Config struct {
	// Kubeconfig is a path to a kubeconfig file. Empty uses in-cluster config,
	// then falls back to the default kubeconfig loading rules.
	Kubeconfig string
	// Namespace is where sandbox pods and NetworkPolicies are created.
	Namespace string
	// DefaultImage is used when CreateRequest.TemplateID is empty / unmapped.
	DefaultImage string
	// ImageMap maps pack template IDs to container images.
	ImageMap map[string]string
	// CPURequest / CPULimit are resource quantities (e.g. "100m", "1").
	CPURequest string
	CPULimit   string
	// MemoryRequest / MemoryLimit are resource quantities (e.g. "128Mi", "1Gi").
	MemoryRequest string
	MemoryLimit   string
	// ReadyTimeout bounds waiting for Pod Ready after Create.
	ReadyTimeout time.Duration
	// RunAsNonRoot enables the non-root security context (UID 65532).
	// Default false so apt-based AdditionalPackages work with common images;
	// set true for hardened clusters.
	RunAsNonRoot bool
	// ServiceAccountName optional pod service account.
	ServiceAccountName string
	// Logger defaults to slog.Default when nil.
	Logger *slog.Logger
	// DNSNamespaceLabels selects the namespace for DNS egress (default kube-system).
	DNSNamespaceLabels map[string]string
	// DNSPodLabelSets are OR'd DNS pod selectors within DNSNamespaceLabels
	// (defaults cover kube-dns and CoreDNS).
	DNSPodLabelSets []map[string]string
}

// DNSPolicyConfig controls DNS egress NetworkPolicy peers.
type DNSPolicyConfig struct {
	NamespaceLabels map[string]string
	PodLabelSets    []map[string]string
}

func defaultDNSPolicyConfig() DNSPolicyConfig {
	return DNSPolicyConfig{
		NamespaceLabels: map[string]string{
			"kubernetes.io/metadata.name": "kube-system",
		},
		PodLabelSets: []map[string]string{
			{"k8s-app": "kube-dns"},
			{"app.kubernetes.io/name": "coredns"},
		},
	}
}

func (c Config) dnsPolicy() DNSPolicyConfig {
	if len(c.DNSNamespaceLabels) == 0 && len(c.DNSPodLabelSets) == 0 {
		return defaultDNSPolicyConfig()
	}
	cfg := DNSPolicyConfig{
		NamespaceLabels: c.DNSNamespaceLabels,
		PodLabelSets:    c.DNSPodLabelSets,
	}
	if len(cfg.NamespaceLabels) == 0 {
		cfg.NamespaceLabels = defaultDNSPolicyConfig().NamespaceLabels
	}
	if len(cfg.PodLabelSets) == 0 {
		cfg.PodLabelSets = defaultDNSPolicyConfig().PodLabelSets
	}
	return cfg
}

func (c Config) logger() *slog.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return slog.Default()
}

func (c Config) namespace() string {
	if strings.TrimSpace(c.Namespace) == "" {
		return defaultNamespace
	}
	return strings.TrimSpace(c.Namespace)
}

func (c Config) defaultImage() string {
	if strings.TrimSpace(c.DefaultImage) == "" {
		return defaultImage
	}
	return strings.TrimSpace(c.DefaultImage)
}

func (c Config) resolveImage(templateID string) string {
	templateID = strings.TrimSpace(templateID)
	if templateID != "" && c.ImageMap != nil {
		if image, ok := c.ImageMap[templateID]; ok && strings.TrimSpace(image) != "" {
			return strings.TrimSpace(image)
		}
		// Treat unknown template IDs as literal image references (same as Docker).
		if strings.Contains(templateID, "/") || strings.Contains(templateID, ":") {
			return templateID
		}
	}
	if templateID != "" && (strings.Contains(templateID, "/") || strings.Contains(templateID, ":")) {
		return templateID
	}
	return c.defaultImage()
}

func (c Config) cpuRequest() string {
	if strings.TrimSpace(c.CPURequest) == "" {
		return defaultCPURequest
	}
	return strings.TrimSpace(c.CPURequest)
}

func (c Config) cpuLimit() string {
	if strings.TrimSpace(c.CPULimit) == "" {
		return defaultCPULimit
	}
	return strings.TrimSpace(c.CPULimit)
}

func (c Config) memoryRequest() string {
	if strings.TrimSpace(c.MemoryRequest) == "" {
		return defaultMemoryRequest
	}
	return strings.TrimSpace(c.MemoryRequest)
}

func (c Config) memoryLimit() string {
	if strings.TrimSpace(c.MemoryLimit) == "" {
		return defaultMemoryLimit
	}
	return strings.TrimSpace(c.MemoryLimit)
}

func (c Config) readyTimeout() time.Duration {
	if c.ReadyTimeout <= 0 {
		return defaultReadyTimeout
	}
	return c.ReadyTimeout
}
