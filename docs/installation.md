# Installation

## Prerequisites

- Go `>= 1.22` (build from source only)
- `kubectl` with a valid kubeconfig
- Kubernetes cluster with read access

## Via Krew

```bash
kubectl krew install plan
```

## Pre-built Binary

Download from the [Releases](https://github.com/samaasi/kubectl-plan/releases/latest) page.

**Linux / macOS:**
```bash
curl -Lo kubectl-plan \
  https://github.com/samaasi/kubectl-plan/releases/latest/download/kubectl-plan_linux_amd64
chmod +x kubectl-plan
sudo mv kubectl-plan /usr/local/bin/
```

**Windows:**
Download `kubectl-plan_windows_amd64.exe`, rename to `kubectl-plan.exe`, and place it in a directory on `%PATH%`.

**Verify:**
```bash
kubectl plan --help
```

## Build from Source

```bash
git clone https://github.com/samaasi/kubectl-plan.git
cd kubectl-plan
go build -o kubectl-plan ./cmd/kubectl-plan
sudo mv kubectl-plan /usr/local/bin/
```

## RBAC

`kubectl-plan` needs read-only access to cluster resources. Apply the bundled manifests:

```bash
kubectl apply -f deploy/rbac/clusterrole.yaml
kubectl apply -f deploy/rbac/clusterrolebinding.yaml
```

### Minimum permissions required

| API Group | Resources | Verbs |
|---|---|---|
| `apps` | deployments, replicasets, statefulsets, daemonsets | get, list |
| _(core)_ | pods, services, endpoints, namespaces, configmaps, secrets, nodes | get, list |
| `networking.k8s.io` | ingresses, networkpolicies | get, list |
| `policy` | poddisruptionbudgets | get, list |
| `autoscaling` | horizontalpodautoscalers | get, list |
| `batch` | cronjobs, jobs | get, list |

`kubectl-plan` never writes to the cluster.
