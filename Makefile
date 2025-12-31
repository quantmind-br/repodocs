# Makefile

# Variables
BINARY_NAME=repodocs
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/quantmind-br/repodocs-go/pkg/version.Version=$(VERSION) -X github.com/quantmind-br/repodocs-go/pkg/version.BuildTime=$(BUILD_TIME) -X github.com/quantmind-br/repodocs-go/pkg/version.Commit=$(COMMIT) -s -w"

# Go
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=gofmt

# Directories
CMD_DIR=./cmd/repodocs
BUILD_DIR=./build
COVERAGE_DIR=./coverage
INSTALL_DIR=$(HOME)/.local/bin
INSTALL_DIR_GLOBAL=/usr/local/bin
CONFIG_DIR=$(HOME)/.repodocs
CONFIG_TEMPLATE=./configs/config.yaml.template

.PHONY: all build clean test coverage lint fmt vet deps help install uninstall install-global uninstall-global install-config

## Main commands

all: deps lint test build ## Run all steps

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)

## Tests

test: ## Run unit tests
	@echo "Running tests..."
	$(GOTEST) -v -race -short ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -v -race -run Integration ./...

test-e2e: ## Run E2E tests
	@echo "Running E2E tests..."
	$(GOTEST) -v -race -run E2E ./tests/e2e/...

test-all: ## Run all tests
	@echo "Running all tests..."
	$(GOTEST) -v -race -short ./tests/unit/...
	$(GOTEST) -v ./tests/integration/...
	$(GOTEST) -v ./tests/e2e/...

coverage: ## Generate coverage report
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## Code quality

lint: ## Run linters
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint v2..." && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest)
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

## Dependencies

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

## Development

run: ## Run in development mode
	@$(GOCMD) run $(CMD_DIR) $(ARGS)

dev: ## Watch mode (requires air)
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

## Installation

install: build install-config ## Install to ~/.local/bin (user installation)
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@echo "✓ Installed $(BINARY_NAME) to $(INSTALL_DIR)"
	@echo "  Make sure $(INSTALL_DIR) is in your PATH"
	@echo "  Run: export PATH=\"$(INSTALL_DIR):\$$PATH\" (add to ~/.bashrc or ~/.zshrc)"

install-config: ## Install config file to ~/.repodocs (only if not exists)
	@mkdir -p $(CONFIG_DIR)
	@if [ ! -f "$(CONFIG_DIR)/config.yaml" ]; then \
		echo "Creating config file at $(CONFIG_DIR)/config.yaml..."; \
		cp $(CONFIG_TEMPLATE) $(CONFIG_DIR)/config.yaml; \
		echo "✓ Created $(CONFIG_DIR)/config.yaml"; \
		echo "  Edit this file to configure LLM providers and other settings"; \
	else \
		echo "✓ Config file already exists at $(CONFIG_DIR)/config.yaml (not overwritten)"; \
	fi

uninstall: ## Remove from ~/.local/bin
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Uninstalled $(BINARY_NAME)"

install-global: build install-config ## Install to /usr/local/bin (requires sudo)
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR_GLOBAL) (requires sudo)..."
	@sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR_GLOBAL)/
	@echo "✓ Installed $(BINARY_NAME) to $(INSTALL_DIR_GLOBAL)"
	@which $(BINARY_NAME) > /dev/null && echo "  $(BINARY_NAME) is now available globally" || echo "  Warning: $(INSTALL_DIR_GLOBAL) not in PATH"

uninstall-global: ## Remove from /usr/local/bin (requires sudo)
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR_GLOBAL) (requires sudo)..."
	@sudo rm -f $(INSTALL_DIR_GLOBAL)/$(BINARY_NAME)
	@echo "✓ Uninstalled $(BINARY_NAME)"

check-install: ## Check installation status
	@echo "Checking $(BINARY_NAME) installation..."
	@if [ -f "$(INSTALL_DIR)/$(BINARY_NAME)" ]; then \
		echo "✓ User installation found: $(INSTALL_DIR)/$(BINARY_NAME)"; \
		echo "  Version: $$($(INSTALL_DIR)/$(BINARY_NAME) version 2>/dev/null || echo 'unknown')"; \
	else \
		echo "✗ Not installed in $(INSTALL_DIR)"; \
	fi
	@if [ -f "$(INSTALL_DIR_GLOBAL)/$(BINARY_NAME)" ]; then \
		echo "✓ Global installation found: $(INSTALL_DIR_GLOBAL)/$(BINARY_NAME)"; \
		echo "  Version: $$($(INSTALL_DIR_GLOBAL)/$(BINARY_NAME) version 2>/dev/null || echo 'unknown')"; \
	else \
		echo "✗ Not installed in $(INSTALL_DIR_GLOBAL)"; \
	fi
	@which $(BINARY_NAME) > /dev/null && echo "✓ $(BINARY_NAME) is in PATH: $$(which $(BINARY_NAME))" || echo "✗ $(BINARY_NAME) not found in PATH"
	@if [ -f "$(CONFIG_DIR)/config.yaml" ]; then \
		echo "✓ Config file found: $(CONFIG_DIR)/config.yaml"; \
	else \
		echo "✗ Config file not found at $(CONFIG_DIR)/config.yaml"; \
	fi

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
