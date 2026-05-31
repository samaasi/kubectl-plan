# kubectl-plan

> **Terraform has `plan`. Kubernetes should too.**

[![Go Report Card](https://goreportcard.com/badge/github.com/samaasi/kubectl-plan)](https://goreportcard.com/report/github.com/samaasi/kubectl-plan)
[![GoDoc](https://godoc.org/github.com/samaasi/kubectl-plan?status.svg)](https://godoc.org/github.com/samaasi/kubectl-plan)
[![License](https://img.shields.io/github/license/samaasi/kubectl-plan)](LICENSE)

`kubectl-plan` is an **operational decision support CLI plugin** for Kubernetes. It bridges the gap between observability (which tells you what *happened*) and execution (which acts without foresight), answering the ultimate pre-flight question:

**"What will happen if I perform this operation?"**

---

## Why `kubectl-plan`?

Traditional observability tools (Prometheus, Grafana, Jaeger) instrument the past and present. They tell you the error rate *right now* or latency over the *last 7 days*. They cannot evaluate prospective changes. 

`kubectl-plan` evaluates changes **prospective-first**:

```
understand ──> decide ──> act
```

It maps out the blast radius of operations like scaling, rolling restarts, and resource deletions before you apply them to the cluster.

---

## Key Features

- **Confidence-based Decision Support:** Calculates confirmed dependents (using API references, label selectors, and Ingress routings) and computes an uncertainty index so you know exactly what is unknown.
- **Outage Prevention:** Pre-flight checks specifically tailored for high-risk commands: `scale`, `restart`, `delete`.
- **Auditable Scopes:** Inspect the deterministic mathematical scoring breakdown using `kubectl plan why`.
- **Analysis Readiness Checks:** Diagnose exactly how ready your environment is to provide high-confidence checks with `kubectl plan doctor`.
- **Infrastructure Cleanup:** Detect orphaned ConfigMaps, Secrets, or Services routing to dead workloads with `kubectl plan garbage-collect` (Phase 4).

---

## Premium Terminal Interface

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
```

---

## Installation

### Via Krew (Recommended)
```bash
kubectl krew install plan
```

### Manual Compilation
Ensure you have Go installed (`>= 1.22`), clone this repository and run:
```bash
go build -o kubectl-plan ./cmd/kubectl-plan
mv kubectl-plan /usr/local/bin/
```

---

## Quickstart

```bash
# Analyze risk before scaling a critical deployment
kubectl plan scale deployment/payment-api --replicas=0

# Investigate a rolling restart risk score
kubectl plan restart deployment/payment-api

# Get audit metrics on risk score calculations
kubectl plan why deployment/payment-api

# Perform an environment telemetry diagnostic
kubectl plan doctor
```

---

## Architecture Overview

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

---

## Contributing

We love contributions! Please read our [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to set up local Kind clusters, run unit/integration test suites, and follow our PR processes.

---

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
