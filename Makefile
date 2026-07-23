# Copyright 2025 Nutanix. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build pr-monitor binary
	CGO_ENABLED=0 go build -o bin/pr-monitor ./cmd/pr-monitor

.PHONY: test
test: ## Run tests
	go test -race -coverprofile=coverage.out -v ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run --fix

.PHONY: mod-tidy
mod-tidy: ## Run go mod tidy
	go mod tidy -v

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin/ coverage.out
