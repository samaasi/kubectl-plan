# kubectl-plan

> **Terraform has `plan`. Kubernetes should too.**

[![CI](https://github.com/samaasi/kubectl-plan/actions/workflows/ci.yml/badge.svg)](https://github.com/samaasi/kubectl-plan/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/samaasi/kubectl-plan)](https://goreportcard.com/report/github.com/samaasi/kubectl-plan)
[![Coverage](https://codecov.io/gh/samaasi/kubectl-plan/branch/master/graph/badge.svg)](https://codecov.io/gh/samaasi/kubectl-plan)
[![License](https://img.shields.io/github/license/samaasi/kubectl-plan)](LICENSE)
[![Release](https://img.shields.io/github/v/release/samaasi/kubectl-plan)](https://github.com/samaasi/kubectl-plan/releases/latest)

`kubectl-plan` is an **operational decision support CLI plugin** for Kubernetes. It answers the pre-flight question engineers never had a tool for:

**"What will happen if I perform this operation?"**

```
Prometheus tells you what is happening right now.
Grafana shows you what happened over time.
kubectl-plan answers: "Is it safe to do this right now?"
```

---

## Table of Contents

- [Why kubectl-plan?](#why-kubectl-plan)
- [Sample Output](#sample-output)
- [Installation](#installation)
- [Quickstart](#quickstart)
- [Usage](#usage)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

---

## Why kubectl-plan?

Traditional observability tools are retrospective. They cannot evaluate prospective changes.

| Tool | Question answered |
|---|---|
| Prometheus | "What is the current error rate of payment-api?" |
| Grafana | "What was the traffic pattern over the last 7 days?" |
| Jaeger / Tempo | "Which services were called during this request?" |
| kubectl-graph | "What does the dependency graph look like?" |
| **kubectl-plan** | **"Is it safe to scale payment-api to zero right now?"** |

No existing tool answers the last question. That gap is where this project lives.

**Key capabilities:**
- **Confidence-based decision support** — builds a dependency graph from label selectors, Ingress routing, env var references, and owner references, then computes an uncertainty index so you know exactly what is unknown
- **Outage prevention** — pre-flight checks for the commands engineers run without thinking: `scale`, `restart`, `delete`
- **Auditable scores** — every number is deterministic and inspectable with `kubectl plan why`
- **Readiness diagnostics** — `kubectl plan doctor` tells you exactly what instrumentation is missing and what it would improve
- **Read-only by design** — never mutates your cluster through v0.4 (see [SECURITY.md](SECURITY.md))

---

## Sample Output

```
ACTION:     scale deployment/payment-api --replicas=0  [namespace: production]

RISK SCORE:       8.7 / 10  ████████░░  HIGH
CONFIDENCE:        94%      █████████░  (topology + Prometheus)
UNCERTAINTY:       LOW      (well-instrumented service)

DEPENDENTS:
  ├─ checkout-service   DIRECT    [99%]
  │     Evidence: 18,234 req/24h · destination_service=payment-api · Prometheus
  │
  ├─ billing-service    DIRECT    [99%]
  │     Evidence: 4,102 req/24h  · destination_service=payment-api · Prometheus
  │
  └─ invoice-worker     INDIRECT  [70%]
        Evidence: env.PAYMENT_URL matches service DNS · no traffic observed
       ~Uncertain: no Prometheus confirmation

UNKNOWN BLAST RADIUS:
  ⚠ Cannot detect: Kafka consumers, external HTTP clients, Consul-registered services
  ℹ Run `kubectl plan doctor` to see what instrumentation would increase confidence.

RISK CONTRIBUTORS:
  +3.0  production-payments namespace   [criticality: CRITICAL]
  +2.4  Ingress exposed (external traffic)
  +1.8  3 confirmed direct consumers
  +1.5  Cross-namespace impact
  ─────
  = 8.7 / 10

RECOMMENDATION:
  ⚠ Do not proceed during peak traffic.
  → kubectl plan why deployment payment-api   for full scoring breakdown.
```

---

## Installation

### Via Krew (Recommended)

```bash
kubectl krew install plan
kubectl plan --help
```

### Pre-built Binary

Download the latest binary from the [Releases](https://github.com/samaasi/kubectl-plan/releases/latest) page:

```bash
# Linux / macOS
curl -Lo kubectl-plan \
  https://github.com/samaasi/kubectl-plan/releases/latest/download/kubectl-plan_linux_amd64
chmod +x kubectl-plan
sudo mv kubectl-plan /usr/local/bin/
```

Windows: download `kubectl-plan_windows_amd64.exe`, rename to `kubectl-plan.exe`, and place it in a directory on `%PATH%`.

### From Source

Requires Go `>= 1.22` and a configured `kubeconfig`.

```bash
git clone https://github.com/samaasi/kubectl-plan.git
cd kubectl-plan
go build -o kubectl-plan ./cmd/kubectl-plan
sudo mv kubectl-plan /usr/local/bin/
```

---

## Quickstart

### Path A — You have a running cluster

```bash
# Apply read-only RBAC (once per cluster)
kubectl apply -f deploy/rbac/clusterrole.yaml
kubectl apply -f deploy/rbac/clusterrolebinding.yaml

# Diagnose your environment first
kubectl plan doctor

# Analyse any workload
kubectl plan scale deployment/<your-deployment> --replicas=0 -n <namespace>
kubectl plan why deployment/<your-deployment> -n <namespace>
```

> **Not sure which deployment to try?** `kubectl get deployments -A` lists everything.

### Path B — No cluster yet (local Kind)

Requires [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) and Docker.

```bash
# Spin up a local cluster pre-loaded with test workloads
./hack/test-cluster/setup.sh

kubectl plan doctor --context kind-kubectl-plan-test

kubectl plan scale deployment/payment-api --replicas=0 \
  --context kind-kubectl-plan-test -n production

kubectl plan why deployment/payment-api \
  --context kind-kubectl-plan-test -n production
```

The test workloads (`payment-api`, `checkout-service`, `billing-service`) are pre-wired with env var references and cross-namespace dependencies so you see a realistic dependency graph on the first run.

### What to expect

`kubectl plan doctor` reports the confidence level of your environment:

```
DATA SOURCES:
  ✓ Kubernetes API    reachable · N resources scanned
  ✗ Prometheus        not found — topology-only scoring active

ESTIMATED ANALYSIS CONFIDENCE:
  52%  █████░░░░░

TO IMPROVE CONFIDENCE:
  → Integrate Prometheus data source (v0.2)
```

Topology-only mode (no Prometheus) is fully functional for v0.1 — you get dependency graph analysis, risk scoring, and recommendations. Prometheus adds real traffic evidence in v0.2.

---

## Usage

All commands require a working `kubeconfig`. By default they target the current context and namespace.

### Global flags

| Flag | Default | Description |
|---|---|---|
| `-n`, `--namespace` | current context | Target namespace |
| `--context` | current context | Override kubeconfig context |
| `-o`, `--output` | `terminal` | Output format: `terminal` \| `json` |
| `--all-namespaces` | false | Include cross-namespace dependency scanning |
| `--ascii` | false | Disable unicode box drawing |
| `--no-color` | false | Disable ANSI color (also respects `NO_COLOR` env) |

---

### `kubectl plan scale`

Analyse risk before scaling a workload.

```bash
kubectl plan scale deployment/payment-api --replicas=0
kubectl plan scale deployment/payment-api --replicas=5 -n production
kubectl plan scale deployment/payment-api --replicas=0 --output json
```

**Checks:** direct and indirect dependents, HPA presence, PodDisruptionBudget constraints, cross-namespace impact, namespace criticality.

---

### `kubectl plan restart`

Analyse risk before a rolling restart — the command engineers run "without thinking".

```bash
kubectl plan restart deployment/payment-api
kubectl plan restart statefulset/postgres -n data
```

Rolling restarts cascade. `kubectl plan restart` surfaces the blast radius before the pods start terminating.

---

### `kubectl plan why`

Inspect the full, auditable scoring breakdown for any workload.

```bash
kubectl plan why deployment/payment-api
kubectl plan why deployment/payment-api -n production
```

Every scoring rule, its weight, its computed value, and its contribution to the final score — nothing is hidden.

---

### `kubectl plan doctor`

Diagnose why confidence scores are low and get actionable improvement steps.

```bash
kubectl plan doctor
kubectl plan doctor --namespace production
kubectl plan doctor --output json
```

**Checks:** Kubernetes API reachability, Prometheus availability, service mesh detection, OpenTelemetry collector presence, namespace criticality profile, historical record count.

---

## Configuration

### Namespace Criticality Profiles

By default, any namespace containing `prod` is treated as `HIGH` criticality. Override with a YAML profile:

```bash
cp config/criticality.example.yaml ~/.kubectl-plan/criticality.yaml
```

```yaml
# ~/.kubectl-plan/criticality.yaml
profiles:
  - namespace: production-payments
    level: CRITICAL
  - namespace: production-checkout
    level: HIGH
  - namespace: staging
    level: LOW
```

See [docs/criticality-profiles.md](docs/criticality-profiles.md) for the full level reference and score multipliers.

### RBAC — Minimum Required Permissions

`kubectl-plan` is **read-only**. Apply the bundled ClusterRole:

```bash
kubectl apply -f deploy/rbac/clusterrole.yaml
kubectl apply -f deploy/rbac/clusterrolebinding.yaml
```

See [docs/installation.md](docs/installation.md) for the full permission matrix.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    kubectl-plan                      │
│                    (CLI plugin)                      │
└───────────────────┬─────────────────────────────────┘
                    │
          ┌─────────▼──────────┐
          │   Command Layer     │  Verb + resource parsing
          │   (Cobra CLI)       │  Flag normalization
          └─────────┬───────────┘
                    │
          ┌─────────▼───────────────┐
          │   Analysis Engine        │  Orchestrates pipeline
          │   internal/analysis/     │  Returns AnalysisResult
          └─────────┬───────────────┘
                    │
        ┌───────────▼──────────────────────────┐
        │        Dependency Engine              │
        │        internal/dependency/           │
        │  Builds confidence-weighted graph     │
        │  Tags every edge with Evidence        │
        │  Computes per-edge uncertainty        │
        └───────────┬──────────────────────────┘
                    │
   ┌────────────────▼───────────────────────────┐
   │           Data Source Adapters              │
   │  ┌────────────┐  ┌──────────────────────┐  │
   │  │ Kubernetes │  │   Prometheus         │  │
   │  │    API     │  │   (optional)         │  │
   │  └────────────┘  └──────────────────────┘  │
   │  ┌────────────┐  ┌──────────────────────┐  │
   │  │  History   │  │  Criticality Profile │  │
   │  │  Store     │  │  (~/.kubectl-plan/)  │  │
   │  └────────────┘  └──────────────────────┘  │
   └────────────────┬───────────────────────────┘
                    │
          ┌─────────▼──────────────┐
          │   Risk Scorer           │  Deterministic weighted rules
          │   + Uncertainty Scorer  │  Separate axis from risk
          │   internal/risk/        │
          └─────────┬───────────────┘
                    │
          ┌─────────▼──────────┐
          │  Output Renderer    │  Terminal / JSON / CI
          └────────────────────┘
```

**Dependency resolution (v0.1 — K8s API only):**

| Step | Method | Confidence |
|---|---|---|
| 1 | `ownerReferences` | 1.00 |
| 2 | Service label selectors → pod labels | 0.95 |
| 3 | Ingress backends → matched Services | 0.95 |
| 4 | NetworkPolicy ingress selectors | 0.80 |
| 5 | Env var values matching service name / cluster DNS | 0.70 |
| 6 | DNS pattern matching in string values | 0.65 |
| 7 | ConfigMap/Secret volume mounts | 0.60 |
| 8 | CronJob URL pattern matching | 0.50 |

**Risk scoring formula:**

```
risk_score = Σ (rule_weight × rule_value) / Σ active_rule_weights × 10
```

Fully deterministic. No ML. Reproducible given the same cluster state. Full documentation in [docs/risk-model.md](docs/risk-model.md).

---

## Roadmap

### ✅ v0.1 — Core _(current)_

> A working `kubectl plan` plugin that delivers risk output in seconds with zero external dependencies.

| Capability | Status |
|---|---|
| `kubectl plan scale` | ✅ Shipped |
| `kubectl plan restart` | ✅ Shipped |
| `kubectl plan why` | ✅ Shipped |
| `kubectl plan doctor` | ✅ Shipped |
| `kubectl plan delete` | 🔜 In progress |
| Dependency engine (K8s API — 8 resolution steps) | ✅ Shipped |
| Risk scoring (deterministic weighted rules) | ✅ Shipped |
| Uncertainty score (separate axis from risk) | ✅ Shipped |
| Namespace criticality profiles | ✅ Shipped |
| Terminal / JSON output renderer | ✅ Shipped |
| RBAC manifests (read-only ClusterRole) | ✅ Shipped |
| GoReleaser multi-platform distribution | ✅ Shipped |

---

### 🔄 v0.2 — Observability Integration

> Replace topological inference with real traffic evidence from Prometheus.

- Auto-discover Prometheus in cluster
- Named PromQL builders for traffic, error rate, P99 latency
- Evidence enrichment: upgrade topology edges with observed traffic (confidence → 0.99)
- Discover Prometheus-only dependencies invisible to topology analysis
- Graceful degradation: topology-only mode when Prometheus is absent

---

### 🔄 v0.3 — GitOps Integration

> Shift risk analysis left into PR workflows and manifest diffs.

- `kubectl plan manifest ./k8s/` — diff manifests vs live cluster, run analysis per changed resource
- ArgoCD PreSync resource hook + PR comment posting
- GitHub Actions integration (`kubectl-plan/action@v1`)
- Flux notification provider

---

### 🔄 v0.4 — Historical Impact Memory

> Stop inferring. Start remembering.

- Append-only local history store (`~/.kubectl-plan/history.jsonl`)
- `kubectl plan history deployment/payment-api` — surface past operations on same target
- Historical evidence in risk output: "Previous scale 3→1 caused +32% latency"

---

### 🔄 v1.0 — Stable + Admission Controller _(opt-in)_

> Enforce risk thresholds at the API server level for teams that require it.

- `ValidatingAdmissionWebhook` server with configurable risk threshold
- cert-manager integration for TLS
- Stability guarantee: API compatibility from this release forward

---

## Contributing

Contributions of all kinds are welcome — bug reports, documentation, test fixtures, and new features.

Read [CONTRIBUTING.md](CONTRIBUTING.md) for dev setup, the test suite, and the branch workflow (`develop` → `master`).

---

## Security

`kubectl-plan` is **read-only through v0.4**. It never creates, patches, or deletes any Kubernetes resource.

See [SECURITY.md](SECURITY.md) for the full security policy and how to report vulnerabilities.

---

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.
