# Namespace Criticality Profiles

By default, `kubectl-plan` uses a simple heuristic: any namespace whose name contains `prod` is treated as `HIGH` criticality. This is intentionally conservative for out-of-box safety.

Real organizations need more nuance. `production-payments` is not the same risk level as `production-marketing`.

## Configuration file

Place the config at `~/.kubectl-plan/criticality.yaml`:

```yaml
profiles:
  - namespace: production-payments
    level: CRITICAL   # weight multiplier: 1.00 (full weight_20)
  - namespace: production-checkout
    level: HIGH       # weight multiplier: 0.80
  - namespace: production-marketing
    level: MEDIUM     # weight multiplier: 0.50
  - namespace: staging
    level: LOW        # weight multiplier: 0.10
  - namespace: development
    level: LOW
```

Copy the example from the repo:

```bash
cp config/criticality.example.yaml ~/.kubectl-plan/criticality.yaml
```

## Criticality levels

| Level | Risk multiplier | When to use |
|---|---|---|
| `CRITICAL` | 1.00 | Revenue-critical, customer-facing, regulated |
| `HIGH` | 0.80 | Important internal services, SLO-governed |
| `MEDIUM` | 0.50 | Non-critical workloads in production |
| `LOW` | 0.10 | Staging, dev, test namespaces |

## Matching rules

- Exact match on namespace name
- If no profile matches, falls back to the heuristic (`prod` substring → `HIGH`, else `LOW`)
- The first matching profile wins (order matters)

## Verifying your profile loaded

```bash
kubectl plan doctor
```

Output includes:

```
NAMESPACE CRITICALITY PROFILE:
  ✓ Config loaded: ~/.kubectl-plan/criticality.yaml
  production-payments    → CRITICAL
  production-checkout    → HIGH
  production-marketing   → MEDIUM
```
