.PHONY: lint test test-int test-all build docker clean migrate seed

# Binary names — override in project-specific Makefile
BINARY_NAME ?= app
MODULE      ?= github.com/org/project

lint:
	golangci-lint run ./...

test:
	go test ./... -count=1 -race

test-int:
	go test -tags integration ./test/integration/... -count=1 -v

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
