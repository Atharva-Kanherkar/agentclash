# Kubernetes sandbox provider (Fleet 4)

`SANDBOX_PROVIDER=kubernetes` runs each sandbox as a **Pod** (not a Job) with a
companion **NetworkPolicy**. This is the self-hosted compute path for fleets
that bring their own cluster (kind, k3s, EKS, GKE, …).

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `SANDBOX_PROVIDER` | `unconfigured` | Set to `kubernetes` |
| `SANDBOX_K8S_KUBECONFIG` | _(empty)_ | Kubeconfig path; empty → in-cluster, then default kubeconfig |
| `SANDBOX_K8S_NAMESPACE` | `agentclash-sandboxes` | Namespace for pods + policies |
| `SANDBOX_K8S_DEFAULT_IMAGE` | `python:3.12-slim` | Image when template unmapped |
| `SANDBOX_K8S_IMAGE_MAP` | _(empty)_ | `templateID=image,template2=image2` |
| `SANDBOX_K8S_CPU_REQUEST` / `_LIMIT` | `100m` / `1` | Resource quantities |
| `SANDBOX_K8S_MEMORY_REQUEST` / `_LIMIT` | `128Mi` / `1Gi` | Resource quantities |
| `SANDBOX_K8S_RUN_AS_NON_ROOT` | `false` | When `true`, UID 65532 + seccomp RuntimeDefault |
| `SANDBOX_K8S_SERVICE_ACCOUNT` | _(empty)_ | Optional pod service account |

Capacity middleware from Fleet 3 wraps this provider when
`SANDBOX_MAX_CONCURRENT > 0`.

## Network isolation

Standard Kubernetes `NetworkPolicy` (not Cilium):

- **Default deny** ingress + egress when `AllowNetwork=false`.
- When `AllowNetwork=true` with an empty allowlist → allow all egress.
- When `AllowNetwork=true` with allowlist entries:
  - CIDR / `ip` / `cidr:port` → egress peers
  - DNS hostnames → **ignored by NetworkPolicy** (logged). Use an egress proxy
    or resolve to CIDRs. inspect_k8s_sandbox uses Cilium `toFQDNs` for this;
    we deliberately stay on portable NetworkPolicy for v1.

DNS to the cluster DNS service is always allowed when network is enabled so
CIDR allowlists remain usable with name resolution.

## Timeouts

`CreateRequest.Timeout` maps to Pod `activeDeadlineSeconds`. When the deadline
fires, the pod becomes `Failed` / `DeadlineExceeded` and session ops surface a
timeout-class error (same activity retry posture as other sandbox failures).

## Reaper-friendly labels

Every pod and NetworkPolicy carries:

- `agentclash.dev/managed-by=runtime-sandbox-kubernetes`
- `agentclash.dev/run-id`
- `agentclash.dev/run-agent-id`
- `agentclash.dev/sandbox-id`

Destroy deletes the pod (foreground) and its NetworkPolicy. A namespace-scoped
controller can also garbage-collect by `managed-by` after cancellation.

## Conformance

```bash
cd runtime
go test ./sandbox/conformance -run Fake -count=1
AGENTCLASH_DOCKER_CONFORMANCE=1 go test ./sandbox/conformance -run Docker -count=1
AGENTCLASH_K8S_CONFORMANCE=1 SANDBOX_K8S_NAMESPACE=agentclash-sandboxes \
  go test ./sandbox/conformance -run KubernetesCluster -count=1
```

## kind smoke (optional / nightly)

```bash
kind create cluster --name agentclash-sandbox
kubectl create namespace agentclash-sandboxes
# install a CNI that enforces NetworkPolicy (kind's default does)
export SANDBOX_PROVIDER=kubernetes
export SANDBOX_K8S_NAMESPACE=agentclash-sandboxes
# run worker against the kind kubeconfig, then a pack case
```

`kind` is not required for unit tests; the in-memory fake cluster covers
provider shape, NetworkPolicy construction, and conformance.
