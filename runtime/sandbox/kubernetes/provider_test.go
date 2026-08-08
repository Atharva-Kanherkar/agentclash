package kubernetes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agentclash/agentclash/runtime/sandbox"
	"github.com/google/uuid"
	networkingv1 "k8s.io/api/networking/v1"
)

func asSession(t *testing.T, s sandbox.Session) *session {
	t.Helper()
	out, ok := s.(*session)
	if !ok {
		t.Fatalf("session type = %T, want *session", s)
	}
	return out
}

func TestProvider_CreateExecDestroy(t *testing.T) {
	cl := newFakeCluster()
	provider := NewProviderWithCluster(cl, Config{Namespace: "test-ns"})
	ctx := context.Background()

	sess, err := provider.Create(ctx, sandbox.CreateRequest{
		RunID:      uuid.New(),
		RunAgentID: uuid.New(),
		Timeout:    time.Minute,
		ToolPolicy: sandbox.ToolPolicy{AllowShell: true},
		Filesystem: sandbox.FilesystemSpec{WorkingDirectory: "/workspace"},
		Labels:     map[string]string{"case": "1"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := sess.WriteFile(ctx, "/workspace/hello.txt", []byte("hi")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := sess.ReadFile(ctx, "/workspace/hello.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hi" {
		t.Fatalf("ReadFile = %q", got)
	}

	result, err := sess.Exec(ctx, sandbox.ExecRequest{Command: []string{"echo", "ok"}})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.ExitCode != 0 || result.Stdout != "ok\n" {
		t.Fatalf("Exec result = %+v", result)
	}

	if err := sess.Destroy(ctx); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if err := sess.Destroy(ctx); err != nil {
		t.Fatalf("second Destroy: %v", err)
	}
}

func TestProvider_NetworkPolicyDefaultDeny(t *testing.T) {
	cl := newFakeCluster()
	provider := NewProviderWithCluster(cl, Config{Namespace: "net"})
	sess, err := provider.Create(context.Background(), sandbox.CreateRequest{
		RunID:      uuid.New(),
		RunAgentID: uuid.New(),
		ToolPolicy: sandbox.ToolPolicy{AllowNetwork: false},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sess.Destroy(context.Background())

	policy := cl.NetworkPolicy("net", asSession(t, sess).policyName)
	if policy == nil {
		t.Fatal("expected network policy")
	}
	if len(policy.Spec.Egress) != 0 {
		t.Fatalf("default-deny egress rules = %d, want 0", len(policy.Spec.Egress))
	}
	if len(policy.Spec.PolicyTypes) != 2 {
		t.Fatalf("policy types = %v", policy.Spec.PolicyTypes)
	}
}

func TestProvider_NetworkPolicyAllowlistCIDR(t *testing.T) {
	cl := newFakeCluster()
	provider := NewProviderWithCluster(cl, Config{Namespace: "net"})
	sess, err := provider.Create(context.Background(), sandbox.CreateRequest{
		RunID:            uuid.New(),
		RunAgentID:       uuid.New(),
		ToolPolicy:       sandbox.ToolPolicy{AllowNetwork: true},
		NetworkAllowlist: []string{"10.0.0.0/8:443", "example.com"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sess.Destroy(context.Background())

	policy := cl.NetworkPolicy("net", asSession(t, sess).policyName)
	if policy == nil {
		t.Fatal("expected policy")
	}
	if !hasCIDREgress(policy, "10.0.0.0/8") {
		t.Fatalf("missing CIDR egress: %#v", policy.Spec.Egress)
	}
}

func TestProvider_ActiveDeadline(t *testing.T) {
	cl := newFakeCluster()
	provider := NewProviderWithCluster(cl, Config{Namespace: "dl"})
	sess, err := provider.Create(context.Background(), sandbox.CreateRequest{
		RunID:      uuid.New(),
		RunAgentID: uuid.New(),
		Timeout:    time.Second,
		ToolPolicy: sandbox.ToolPolicy{AllowShell: true},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sess.Destroy(context.Background())

	cl.mu.Lock()
	for _, fp := range cl.pods {
		fp.deadline = time.Now().Add(-time.Second)
	}
	cl.mu.Unlock()

	_, err = sess.Exec(context.Background(), sandbox.ExecRequest{Command: []string{"echo", "x"}})
	if err == nil {
		t.Fatal("expected deadline error")
	}
}

func TestProvider_DestroyNotFound(t *testing.T) {
	cl := newFakeCluster()
	provider := NewProviderWithCluster(cl, Config{Namespace: "nf"})
	sess, err := provider.Create(context.Background(), sandbox.CreateRequest{
		RunID: uuid.New(), RunAgentID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	inner := asSession(t, sess)
	_ = cl.DeletePod(context.Background(), inner.namespace, inner.podName)
	err = sess.Destroy(context.Background())
	if !errors.Is(err, sandbox.ErrSandboxNotFound) {
		t.Fatalf("Destroy = %v, want ErrSandboxNotFound", err)
	}
}

func TestParseAllowlist(t *testing.T) {
	entries, hosts, err := ParseAllowlist([]string{"10.0.0.0/8:443", "1.2.3.4", "api.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 || hosts[0] != "api.example.com" {
		t.Fatalf("hosts = %v", hosts)
	}
	if entries[0].CIDR != "10.0.0.0/8" || entries[0].Ports[0] != 443 {
		t.Fatalf("entry0 = %+v", entries[0])
	}
	if entries[1].CIDR != "1.2.3.4/32" {
		t.Fatalf("entry1 = %+v", entries[1])
	}
}

func hasCIDREgress(policy *networkingv1.NetworkPolicy, cidr string) bool {
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.IPBlock != nil && peer.IPBlock.CIDR == cidr {
				return true
			}
		}
	}
	return false
}
