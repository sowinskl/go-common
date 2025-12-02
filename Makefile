.PHONY: help test test-race test-coverage lint fmt vet clean build deps tidy

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run tests
	go test -v ./...

test-race: ## Run tests with race detector
	go test -race -v ./...

test-coverage: ## Run tests with coverage
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

clean: ## Clean build artifacts
	go clean
	rm -f coverage.out coverage.html

build: ## Build the package
	go build ./...

deps: ## Download dependencies
	go mod download

tidy: ## Tidy dependencies
	go mod tidy

check: fmt vet lint test ## Run all checks

ci: deps tidy check test-coverage ## Run CI pipeline locally 
