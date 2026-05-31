# Example: kubectl plan doctor

## When to use this command

Run `kubectl plan doctor` when:
- You see a low confidence score and don't know why
- You've just set up kubectl-plan in a new cluster
- You want to understand which data sources are active

## Command

```bash
kubectl plan doctor
```

## Output — well-instrumented cluster

```
ANALYSIS READINESS  [cluster: production-eks · namespace: production]

DATA SOURCES:
  ✓ Kubernetes API          reachable · 847 resources scanned
  ✓ Prometheus              http://prometheus.monitoring:9090
                            Confidence boost: +25%
                            Coverage: 94% of services have traffic metrics
  ✗ Istio / Service Mesh    not detected
                            Missing: cross-service traffic topology
                            Confidence impact: -15% (estimated)
  ✗ OpenTelemetry           no otel-collector found
                            Missing: trace-based dependency evidence

NAMESPACE CRITICALITY PROFILE:
  ✓ Config loaded: ~/.kubectl-plan/criticality.yaml
  production-payments    → CRITICAL
  production-checkout    → HIGH
  production-marketing   → MEDIUM

ESTIMATED ANALYSIS CONFIDENCE:
  87%  ████████░░

TO IMPROVE CONFIDENCE:
  → Install Istio or Linkerd for traffic topology evidence
  → Install an OTel collector for trace-based dependency mapping
  → Run kubectl plan scale/restart/delete to build historical records

Run `kubectl plan doctor --json` for machine-readable output.
```

## Output — no Prometheus

```
ANALYSIS READINESS  [cluster: dev-cluster · namespace: default]

DATA SOURCES:
  ✓ Kubernetes API          reachable · 42 resources scanned
  ✗ Prometheus              not found
                            ⚠ Confidence reduced — topology-only scoring active
                            Run with --prometheus-url to connect manually
  ✗ Istio / Service Mesh    not detected
  ✗ OpenTelemetry           not detected

NAMESPACE CRITICALITY PROFILE:
  ✗ No config found at ~/.kubectl-plan/criticality.yaml
  Using default heuristic: namespaces containing 'prod' → HIGH

ESTIMATED ANALYSIS CONFIDENCE:
  52%  █████░░░░░

TO IMPROVE CONFIDENCE:
  → Install Prometheus: https://prometheus-community.github.io/helm-charts
  → Create a criticality profile: cp config/criticality.example.yaml ~/.kubectl-plan/criticality.yaml
```
