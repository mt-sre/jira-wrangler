SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c

export CGO_ENABLED=0

build:
	go build -o bin/jira-wrangler ./cmd/jira-wrangler
.PHONY: build

test:
	CGO_ENABLED=1 go test -v -cover -race -count=1 ./...
.PHONY: test

build-image:
	podman build -f Dockerfile .
.PHONY: build-image
