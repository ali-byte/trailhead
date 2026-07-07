.PHONY: lint test test-int test-all build docker clean migrate seed

# Binary name and module path — Trailhead-specific (Phase B gate round 4 fix,
# 2026-07-05: this file was still carrying unfilled generic template values
# — BINARY_NAME=app and a github.com/org/project placeholder MODULE — that
# never matched this project's actual go.mod (module trailhead) or its
# single binary at cmd/trailhead; Codex round-4 finding N1).
BINARY_NAME ?= trailhead
MODULE      ?= trailhead

lint:
	golangci-lint run ./...

test:
	go test ./... -count=1 -race

test-int:
	go test -tags integration ./tests/integration/... -count=1 -v

test-all: test test-int

build:
	go build -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)/

build-all:
	go build ./cmd/...

docker:
	docker build -t $(BINARY_NAME):dev .

clean:
	rm -rf bin/
	go clean -testcache

migrate:
	go run ./scripts/migrate/main.go up

seed:
	go run ./scripts/seed/main.go

# Check all — run before opening a PR
check: lint test build
	@echo "All checks passed"

# Install dev tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
