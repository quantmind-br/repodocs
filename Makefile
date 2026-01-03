# Makefile

BINARY_NAME=repodocs
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/quantmind-br/repodocs-go/pkg/version.Version=$(VERSION) -X github.com/quantmind-br/repodocs-go/pkg/version.BuildTime=$(BUILD_TIME) -X github.com/quantmind-br/repodocs-go/pkg/version.Commit=$(COMMIT) -s -w"

BUILD_DIR=./build
INSTALL_DIR=$(HOME)/.local/bin
CONFIG_DIR=$(HOME)/.repodocs

.PHONY: build test test-all coverage lint deps install uninstall release release-dry clean help

build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/repodocs

test: ## Run unit tests
	go test -v -race -short ./...

test-all: ## Run all tests (unit + integration + e2e)
	go test -v -race ./...

coverage: ## Generate coverage report (HTML in ./coverage/)
	@mkdir -p ./coverage
	go test -coverprofile=./coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=./coverage/coverage.out -o ./coverage/coverage.html
	@go tool cover -func=./coverage/coverage.out | tail -1
	@echo "Report: ./coverage/coverage.html"

lint: ## Run linters and format code
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	gofmt -s -w .
	golangci-lint run ./...

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

install: build ## Build and install to ~/.local/bin
	@mkdir -p $(INSTALL_DIR) $(CONFIG_DIR)
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@test -f $(CONFIG_DIR)/config.yaml || cp ./configs/config.yaml.template $(CONFIG_DIR)/config.yaml
	@echo "✓ Installed to $(INSTALL_DIR)/$(BINARY_NAME)"

uninstall: ## Remove from ~/.local/bin
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Uninstalled"

release: ## Create GitHub release (interactive)
	@./scripts/release.sh

release-dry: ## Test release build locally (creates ./dist/)
	@which goreleaser > /dev/null || go install github.com/goreleaser/goreleaser/v2@latest
	goreleaser release --snapshot --clean

clean: ## Remove build artifacts
	@rm -rf $(BUILD_DIR) ./coverage ./dist

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
