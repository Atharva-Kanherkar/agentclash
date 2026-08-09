package conformance_test

import (
	"os"
	"testing"
	"time"

	"github.com/agentclash/agentclash/runtime/sandbox"
	"github.com/agentclash/agentclash/runtime/sandbox/conformance"
	"github.com/agentclash/agentclash/runtime/sandbox/docker"
	"github.com/agentclash/agentclash/runtime/sandbox/kubernetes"
)

func TestConformance_FakeProvider(t *testing.T) {
	conformance.Run(t, &sandbox.FakeProvider{}, conformance.Options{})
}

func TestConformance_KubernetesFakeCluster(t *testing.T) {
	provider := kubernetes.NewProviderWithCluster(
		kubernetes.NewFakeCluster(),
		kubernetes.Config{Namespace: "conformance"},
	)
	conformance.Run(t, provider, conformance.Options{CreateTimeout: time.Minute})
}

func TestConformance_Docker(t *testing.T) {
	if testing.Short() {
		t.Skip("docker conformance skipped under -short")
	}
	if os.Getenv("AGENTCLASH_DOCKER_CONFORMANCE") != "1" {
		t.Skip("set AGENTCLASH_DOCKER_CONFORMANCE=1 to run")
	}
	provider, err := docker.NewProvider(docker.Config{})
	if err != nil {
		t.Fatalf("docker provider: %v", err)
	}
	defer provider.Close()
	conformance.Run(t, provider, conformance.Options{CreateTimeout: 3 * time.Minute})
}

func TestConformance_KubernetesCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("kubernetes conformance skipped under -short")
	}
	if os.Getenv("AGENTCLASH_K8S_CONFORMANCE") != "1" {
		t.Skip("set AGENTCLASH_K8S_CONFORMANCE=1 (and a reachable cluster) to run")
	}
	provider, err := kubernetes.NewProvider(kubernetes.Config{
		Kubeconfig: os.Getenv("SANDBOX_K8S_KUBECONFIG"),
		Namespace:  envOr("SANDBOX_K8S_NAMESPACE", "agentclash-sandboxes"),
	})
	if err != nil {
		t.Fatalf("kubernetes provider: %v", err)
	}
	defer provider.Close()
	conformance.Run(t, provider, conformance.Options{CreateTimeout: 5 * time.Minute})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
