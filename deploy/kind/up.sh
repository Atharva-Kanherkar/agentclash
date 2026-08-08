#!/usr/bin/env bash
# Kind rehearsal for the AgentClash Helm chart (Fleet 12 / #1209).
# Prerequisites: kind, kubectl, helm, docker. Optional: keda installed in cluster.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-agentclash}"
NAMESPACE="${NAMESPACE:-agentclash}"
RELEASE="${RELEASE:-agentclash}"

echo "==> ensuring kind cluster ${CLUSTER_NAME}"
if ! kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  kind create cluster --name "${CLUSTER_NAME}"
fi
kubectl cluster-info --context "kind-${CLUSTER_NAME}" >/dev/null

echo "==> creating namespace ${NAMESPACE}"
kubectl get ns "${NAMESPACE}" >/dev/null 2>&1 || kubectl create ns "${NAMESPACE}"

echo "==> helm lint"
helm lint "${ROOT}/deploy/helm/agentclash"

echo "==> helm upgrade --install (external Temporal/Postgres expected via secret)"
# Dev secret placeholders — replace before a real eval-set run.
kubectl -n "${NAMESPACE}" create secret generic "${RELEASE}-secrets" \
  --from-literal=DATABASE_URL="${DATABASE_URL:-postgres://agentclash:agentclash@postgres:5432/agentclash?sslmode=disable}" \
  --from-literal=REDIS_URL="${REDIS_URL:-redis://redis:6379/0}" \
  --from-literal=AUTH_MODE=dev \
  --dry-run=client -o yaml | kubectl apply -f -

helm upgrade --install "${RELEASE}" "${ROOT}/deploy/helm/agentclash" \
  --namespace "${NAMESPACE}" \
  --set secrets.existingSecret="${RELEASE}-secrets" \
  --set secrets.create=false \
  --set image.tag="${IMAGE_TAG:-latest}" \
  --set keda.enabled="${KEDA_ENABLED:-false}" \
  --set sandbox.provider="${SANDBOX_PROVIDER:-kubernetes}" \
  --wait --timeout 5m || true

echo "==> pods"
kubectl -n "${NAMESPACE}" get pods,deploy,svc

cat <<EOF

Kind rehearsal scaffold complete.
To finish an end-to-end eval set you still need:
  - Temporal frontend reachable at values.external.temporalAddress
  - Postgres + Redis matching the secret
  - KEDA installed if KEDA_ENABLED=true (scale 0→N test)
  - Images loaded into kind: kind load docker-image <api> <worker> --name ${CLUSTER_NAME}

See docs/deployment/self-host-kubernetes.md
EOF
