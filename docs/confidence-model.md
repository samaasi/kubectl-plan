# Confidence Model

Confidence measures how much of the blast radius `kubectl-plan` can actually observe. It is a **separate axis from risk score**.

## Why confidence matters

```
Risk HIGH + Confidence HIGH → take action, the blast radius is known
Risk HIGH + Confidence LOW  → take more caution, there may be hidden dependents
Risk LOW  + Confidence LOW  → do not assume safety — confidence is not evidence of safety
```

A low confidence score means the tool does not have enough data sources to be certain it has found all dependents. It does **not** mean the operation is safe.

## Evidence confidence values

Every relationship in the dependency graph is tagged with the evidence that produced it:

| Evidence Type | Confidence | Source |
|---|---|---|
| `owner_reference` | 1.00 | Kubernetes API — authoritative |
| `label_selector` | 0.95 | Direct API match |
| `ingress_backend` | 0.95 | Explicit spec match |
| `network_policy` | 0.80 | Selector analysis |
| `env_var` | 0.70 | String match — may be coincidental |
| `dns_pattern` | 0.65 | Pattern match in string values |
| `volume_mount` | 0.60 | Indirect dependency |
| `cron_url_match` | 0.50 | Best-effort regex |
| `prometheus_traffic` | 0.99 | Observed real traffic (Phase 2) |

Edge confidence = max of all evidence confidences on that edge. One strong piece of evidence makes an edge high-confidence.

## Low-confidence indicators in output

Edges with confidence `< 0.65` are prefixed `~` in terminal output:

```
~└─ invoice-worker     INDIRECT  [50%]
      Evidence: env.BILLING_URL regex match in cronjob spec
     ~Uncertain: no Prometheus confirmation
```

## Unknown blast radius

Every plan output includes an explicit section naming what the tool cannot see:

```
UNKNOWN BLAST RADIUS:
  ⚠ Cannot detect: Kafka consumers, external HTTP clients, Consul-registered services
  ℹ Run `kubectl plan doctor` to see what instrumentation would increase confidence.
```

This is intentional. The tool never hides its blind spots.

## Improving confidence

Run `kubectl plan doctor` to see which data sources are missing and what confidence boost each would provide.
