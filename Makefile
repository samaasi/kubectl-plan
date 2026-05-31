.PHONY: build test clean run

build:
	go build -o kubectl-plan ./cmd/kubectl-plan

test:
	go test ./... -v

run:
	go run ./cmd/kubectl-plan

clean:
	rm -f kubectl-plan
