# Example: kubectl plan why

## Purpose

`kubectl plan why` makes every risk score fully auditable. Use it when you want to understand exactly which factors produced a score — and how to reduce it.

## Command

```bash
kubectl plan why deployment/payment-api -n production
```

## Output

```
RISK SCORE BREAKDOWN: deployment/payment-api

Score:       8.7 / 10  ████████░░  HIGH
Confidence:  94%        █████████░  (topology + Prometheus)
Uncertainty: LOW        (well-instrumented — strong evidence across most dependents)

CONTRIBUTORS:
  Rule                              Weight   Value   Contribution
  ─────────────────────────────── ──────   ─────   ────────────
  production-payments namespace     ×20     1.00    +3.0   ← criticality: CRITICAL
  Ingress exposed (external)        ×10     1.00    +2.4
  Direct confirmed consumers (3)    ×30     0.60    +1.8
  Cross-namespace impact            ×10     1.00    +1.5
  PodDisruptionBudget present       ×10     1.00    +1.4   ← scale ops constrained
  HPA configured (may recover)      ×5      0.50    +0.3
  Indirect confirmed consumers (1)  ×15     0.10    +0.3
  ─────────────────────────────── ──────   ─────   ────────────
  Total                                             8.7 / 10

CONFIDENCE SOURCES:
  ✓ Kubernetes topology    (label selectors, ingress routing, owner references)
  ✓ Prometheus traffic     (24h request evidence for 2 of 3 direct dependents)
  ? invoice-worker         (env var match only, no Prometheus confirmation)

UNKNOWN BLAST RADIUS:
  ⚠ Kafka consumers, external HTTP clients, Consul-registered services
```

## Reading the output

- **Weight** — the rule's importance relative to other rules. Higher weight → more impact on final score.
- **Value** — a normalized `0.0–1.0` value computed from live cluster state.
- **Contribution** — `(weight × value) / total_weight × 10`, what this rule adds to the final score.
- **Confidence sources** — which data sources contributed evidence, and for which dependents.

## How to reduce a score

The output tells you directly which rules are contributing most. To reduce the score for `payment-api`:
- Moving it to a non-production namespace would drop the `+3.0` namespace_criticality contribution
- Adding Prometheus confirmation for `invoice-worker` would lower uncertainty (not the score itself)
- Removing the Ingress exposure would drop `+2.4`
