#!/usr/bin/env bash
set -euo pipefail

# Build kubectl-plan with full version metadata injected.
# Usage: ./hack/build.sh [version]
# Example: ./hack/build.sh v0.1.0

VERSION=${1:-dev}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILT=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-X github.com/samaasi/kubectl-plan/pkg/version.Version=${VERSION} \
         -X github.com/samaasi/kubectl-plan/pkg/version.Commit=${COMMIT} \
         -X github.com/samaasi/kubectl-plan/pkg/version.BuildDate=${BUILT}"

echo "Building kubectl-plan ${VERSION} (${COMMIT})..."
go build -ldflags "${LDFLAGS}" -o kubectl-plan ./cmd/kubectl-plan
echo "Done: ./kubectl-plan"
