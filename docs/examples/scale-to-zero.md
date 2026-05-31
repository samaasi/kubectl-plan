# Example: kubectl plan scale

## Scenario

You are about to scale `payment-api` to zero replicas during an off-hours maintenance window. You want to know the blast radius before issuing the scale command.

## Command

```bash
kubectl plan scale deployment/payment-api --replicas=0 -n production
```

## Output

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
  ⚠ Score 8.7 — do not proceed without review.
  → kubectl plan why deployment payment-api   for full scoring breakdown.
```

## What to do with this information

- **checkout-service** and **billing-service** are confirmed directly affected (99% confidence each, Prometheus traffic evidence).
- **invoice-worker** is likely affected but not confirmed — investigate `env.PAYMENT_URL` before proceeding.
- **Kafka consumers and external clients** are invisible to topology analysis. If you have consumers subscribing to events produced by payment-api, they are in the unknown blast radius.

## Safe scaling approach

1. Verify `invoice-worker`'s `PAYMENT_URL` environment variable target
2. Coordinate with the checkout and billing teams
3. Run outside peak traffic hours
4. Scale to 1 first (`--replicas=1`), not directly to 0
