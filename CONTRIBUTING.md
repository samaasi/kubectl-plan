# Contributing to `kubectl-plan`

Thank you for your interest in contributing to `kubectl-plan`!

## Code of Conduct
We expect all contributors to adhere to standard respectful communications in our issue tracker, pull requests, and discussions.

## How to Help

You can contribute in several ways:
- **Reporting Bugs:** Create an issue with clear replication instructions and error traces.
- **Requesting Features:** Pitch new features or additional dependency checks.
- **Submitting Pull Requests:** Fix bugs or add capabilities directly.

## Development Setup

### Prerequisite Libraries
- Go (`>= 1.22`)
- A local Kubernetes cluster (like [Kind](https://kind.sigs.k8s.io/) or [Minikube](https://minikube.sigs.k8s.io/))

### Cloning the Project
```bash
git clone https://github.com/samaasi/kubectl-plan.git
cd kubectl-plan
go mod download
```

### Running Unit Tests
We require all contributions to have passing unit tests. Run:
```bash
go test ./... -v
```

### Locally Installing the Binary
```bash
go build -o kubectl-plan ./cmd/kubectl-plan
# Add to your PATH to use as a kubectl plugin
export PATH=$PATH:$(pwd)
```

## Pull Request Guidelines

1. **Keep it Focused:** Keep PRs dedicated to a single feature or bug fix.
2. **Write Unit Tests:** Ensure new topological checks or scorer modifications are tested.
3. **Format Code:** Run `go fmt ./...` before committing.
4. **Follow Branch Naming:**
   - `feat/some-capability`
   - `fix/some-bug`
   - `docs/some-documentation`
