# Risk Scoring Model

`kubectl-plan` uses a fully deterministic, weighted rule engine. No ML. No LLMs. Given the same cluster state, the score is always the same.

## Formula

```
risk_score = Σ (rule_weight × rule_value) / Σ active_rule_weights × 10
```

The result is normalized to a `0.0–10.0` scale.

## Phase 1 Rule Registry

| Rule ID | Weight | Value Logic |
|---|---|---|
| `namespace_criticality` | 20 | Based on criticality profile; default `1.0` if namespace name contains `prod` |
| `direct_dependents` | 30 | `min(confirmed_direct_count / 5, 1.0)` |
| `indirect_dependents` | 15 | `min(confirmed_indirect_count / 10, 1.0)` |
| `ingress_exposed` | 10 | `1.0` if any Ingress routes to this resource |
| `cross_namespace_impact` | 10 | `1.0` if dependents span > 1 namespace |
| `has_pdb` | 10 | `1.0` if PDB with `minAvailable > 0` |
| `has_hpa` | 5 | `0.5` — HPA may auto-recover; partial risk only |

## Risk Levels

| Score | Level | Terminal color |
|---|---|---|
| 0.0–3.0 | LOW | Green |
| 3.1–6.0 | MEDIUM | Yellow |
| 6.1–8.5 | HIGH | Red |
| 8.6–10.0 | CRITICAL | Bold red |

## Phase 2 Additions (Prometheus)

When Prometheus is available, three additional rules activate:

| Rule | Weight | Value logic |
|---|---|---|
| `live_request_rate` | 25 | `min(rps / 1000, 1.0)` |
| `error_rate_elevated` | 15 | `1.0` if current error rate > 1% |
| `p99_latency_high` | 10 | `1.0` if P99 > 500ms |

## Worked Example

```
deployment/payment-api  namespace: production-payments (CRITICAL profile)

Rule                              Weight   Value   Contribution
───────────────────────────────── ──────   ─────   ────────────
namespace_criticality (CRITICAL)    20     1.00      +3.0
ingress_exposed                     10     1.00      +2.4
direct_dependents (3)               30     0.60      +1.8
cross_namespace_impact              10     1.00      +1.5
has_pdb                             10     1.00      +1.4  ← constrained
has_hpa                              5     0.50      +0.3
indirect_dependents (1)             15     0.10      +0.3
───────────────────────────────── ──────   ─────   ────────────
Total                                               8.7 / 10
```

Inspect any workload's breakdown with:

```bash
kubectl plan why deployment/payment-api
```
