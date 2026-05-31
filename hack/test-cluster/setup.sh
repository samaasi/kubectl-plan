#!/usr/bin/env bash
set -euo pipefail

# Spins up a local Kind cluster with test workloads for integration testing.
# Requirements: kind, kubectl, helm (for optional Prometheus)

CLUSTER_NAME=${CLUSTER_NAME:-kubectl-plan-test}

echo "Creating Kind cluster: ${CLUSTER_NAME}..."
kind create cluster --name ${CLUSTER_NAME} 2>/dev/null || echo "Cluster already exists, continuing..."

echo "Applying test workloads..."
kubectl apply -f testdata/fixtures/deployments.yaml --context kind-${CLUSTER_NAME}
kubectl apply -f testdata/fixtures/services.yaml     --context kind-${CLUSTER_NAME}
kubectl apply -f testdata/fixtures/ingresses.yaml    --context kind-${CLUSTER_NAME}

echo ""
echo "Cluster ready. Run commands with:"
echo "  kubectl plan scale deployment/payment-api --replicas=0 --context kind-${CLUSTER_NAME}"
echo "  kubectl plan doctor --context kind-${CLUSTER_NAME}"
echo ""
echo "To tear down: kind delete cluster --name ${CLUSTER_NAME}"
