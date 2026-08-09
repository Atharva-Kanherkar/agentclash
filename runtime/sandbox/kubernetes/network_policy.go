package kubernetes

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// AllowlistEntry is a parsed CIDR/port allowlist item.
type AllowlistEntry struct {
	CIDR  string
	Ports []int32
	// Hostname is set when the entry is a DNS name. Standard NetworkPolicy
	// cannot express FQDNs; callers must use an egress proxy (documented).
	Hostname string
}

// ParseAllowlist parses CreateRequest.NetworkAllowlist entries.
//
// Accepted forms:
//   - "10.0.0.0/8"
//   - "10.0.0.0/8:443"
//   - "192.168.1.1" (host → /32)
//   - "example.com" (hostname — recorded but not enforceable via NetworkPolicy)
func ParseAllowlist(entries []string) ([]AllowlistEntry, []string /*hostnames*/, error) {
	var parsed []AllowlistEntry
	var hostnames []string
	for _, raw := range entries {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		host, port, hasPort := strings.Cut(raw, ":")
		host = strings.TrimSpace(host)
		if host == "" {
			return nil, nil, fmt.Errorf("invalid network allowlist entry %q", raw)
		}
		if ip := net.ParseIP(host); ip != nil {
			cidr := host + "/32"
			if ip.To4() == nil {
				cidr = host + "/128"
			}
			entry := AllowlistEntry{CIDR: cidr}
			if hasPort {
				p, err := strconv.Atoi(strings.TrimSpace(port))
				if err != nil || p <= 0 || p > 65535 {
					return nil, nil, fmt.Errorf("invalid port in allowlist entry %q", raw)
				}
				entry.Ports = []int32{int32(p)}
			}
			parsed = append(parsed, entry)
			continue
		}
		if _, _, err := net.ParseCIDR(host); err == nil {
			entry := AllowlistEntry{CIDR: host}
			if hasPort {
				p, err := strconv.Atoi(strings.TrimSpace(port))
				if err != nil || p <= 0 || p > 65535 {
					return nil, nil, fmt.Errorf("invalid port in allowlist entry %q", raw)
				}
				entry.Ports = []int32{int32(p)}
			}
			parsed = append(parsed, entry)
			continue
		}
		// DNS name — not expressible in standard NetworkPolicy.
		hostname := raw
		if hasPort {
			hostname = host
		}
		hostnames = append(hostnames, hostname)
		parsed = append(parsed, AllowlistEntry{Hostname: hostname})
	}
	return parsed, hostnames, nil
}

func buildNetworkPolicy(namespace, name, sandboxID string, allowNetwork bool, allowlist []string, dns DNSPolicyConfig) (*networkingv1.NetworkPolicy, []string, error) {
	entries, hostnames, err := ParseAllowlist(allowlist)
	if err != nil {
		return nil, nil, err
	}

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				labelManagedBy:  labelManagedByValue,
				labelSandboxID:  sandboxID,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{labelSandboxID: sandboxID},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			// Default-deny: empty Ingress; Egress only what we add below.
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
		},
	}

	if !allowNetwork {
		policy.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{}
		return policy, hostnames, nil
	}

	// Allow DNS only to cluster DNS pods in kube-system so CIDR allowlists
	// remain usable with name resolution.
	dnsUDP := corev1.ProtocolUDP
	dnsTCP := corev1.ProtocolTCP
	dnsPort := intstr.FromInt32(53)
	dnsPorts := []networkingv1.NetworkPolicyPort{
		{Protocol: &dnsUDP, Port: &dnsPort},
		{Protocol: &dnsTCP, Port: &dnsPort},
	}
	egress := make([]networkingv1.NetworkPolicyEgressRule, 0, len(dns.PodLabelSets))
	for _, podLabels := range dns.PodLabelSets {
		egress = append(egress, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: dns.NamespaceLabels},
				PodSelector:       &metav1.LabelSelector{MatchLabels: podLabels},
			}},
			Ports: dnsPorts,
		})
	}

	for _, entry := range entries {
		if entry.CIDR == "" {
			continue
		}
		cidr := entry.CIDR
		peer := networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{CIDR: cidr},
		}
		rule := networkingv1.NetworkPolicyEgressRule{To: []networkingv1.NetworkPolicyPeer{peer}}
		if len(entry.Ports) > 0 {
			proto := corev1.ProtocolTCP
			for _, p := range entry.Ports {
				port := intstr.FromInt32(p)
				rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
					Protocol: &proto,
					Port:     &port,
				})
			}
		}
		egress = append(egress, rule)
	}

	// If allowNetwork with empty allowlist → allow all egress (explicit open).
	if len(allowlist) == 0 {
		egress = []networkingv1.NetworkPolicyEgressRule{{}}
	}

	policy.Spec.Egress = egress
	return policy, hostnames, nil
}
