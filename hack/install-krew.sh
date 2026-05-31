#!/usr/bin/env bash
set -euo pipefail

# Installs kubectl-plan as a krew plugin from the local build.
# Requires: kubectl-krew, go

BINARY=kubectl-plan

echo "Building kubectl-plan..."
go build -o ${BINARY} ./cmd/kubectl-plan

echo "Installing into krew plugin path..."
KREW_ROOT="${KREW_ROOT:-${HOME}/.krew}"
mkdir -p "${KREW_ROOT}/bin"
cp ${BINARY} "${KREW_ROOT}/bin/kubectl-${BINARY}"

echo "Done. Run: kubectl plan --help"
