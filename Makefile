.PHONY: build test lint clean install

BINARY    := kubectl-plan
VERSION   ?= dev
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILT     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -X github.com/samaasi/kubectl-plan/pkg/version.Version=$(VERSION) \
             -X github.com/samaasi/kubectl-plan/pkg/version.Commit=$(COMMIT) \
             -X github.com/samaasi/kubectl-plan/pkg/version.BuildDate=$(BUILT)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/kubectl-plan

test:
	go test ./... -race -coverprofile=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out

install: build
	mv $(BINARY) /usr/local/bin/

# Runs the binary against the current kubeconfig context (dev convenience)
run:
	go run -ldflags "$(LDFLAGS)" ./cmd/kubectl-plan $(ARGS)
