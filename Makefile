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

.PHONY: all build clean test coverage lint fmt vet deps help install uninstall install-global uninstall-global install-config release release-dry \
	test-all test-app test-cache test-config test-converter test-domain test-fetcher test-git test-llm test-output test-renderer test-strategies test-cmd \
	coverage-all coverage-app coverage-cache coverage-config coverage-converter coverage-domain coverage-fetcher coverage-git coverage-llm coverage-output coverage-renderer coverage-strategies coverage-cmd coverage-view coverage-summary

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

## Package-specific tests

test-app: ## Run app package tests
	@echo "Running app package tests..."
	$(GOTEST) -v -race ./tests/unit/app/...

test-cache: ## Run cache package tests
	@echo "Running cache package tests..."
	$(GOTEST) -v -race ./tests/unit/cache/...
	$(GOTEST) -v ./tests/integration/cache/...

test-config: ## Run config package tests
	@echo "Running config package tests..."
	$(GOTEST) -v -race ./tests/unit/config/...

test-converter: ## Run converter package tests
	@echo "Running converter package tests..."
	$(GOTEST) -v -race ./tests/unit/converter/...

test-domain: ## Run domain package tests
	@echo "Running domain package tests..."
	$(GOTEST) -v -race ./tests/unit/domain/...

test-fetcher: ## Run fetcher package tests
	@echo "Running fetcher package tests..."
	$(GOTEST) -v -race ./tests/unit/fetcher/...
	$(GOTEST) -v ./tests/integration/fetcher/...

test-git: ## Run git package tests
	@echo "Running git package tests..."
	$(GOTEST) -v -race ./tests/unit/git/...

test-llm: ## Run llm package tests
	@echo "Running llm package tests..."
	$(GOTEST) -v -race ./tests/unit/llm/...
	$(GOTEST) -v ./tests/integration/llm/...

test-output: ## Run output package tests
	@echo "Running output package tests..."
	$(GOTEST) -v -race ./tests/unit/output/...

test-renderer: ## Run renderer package tests
	@echo "Running renderer package tests..."
	$(GOTEST) -v -race ./tests/unit/renderer/...
	$(GOTEST) -v ./tests/integration/renderer/...

test-strategies: ## Run strategies package tests
	@echo "Running strategies package tests..."
	$(GOTEST) -v -race ./tests/unit/strategies/...

test-cmd: ## Run cmd/repodocs tests
	@echo "Running cmd/repodocs tests..."
	$(GOTEST) -v -race ./cmd/repodocs/...

## Coverage reports

coverage: ## Generate overall coverage report
	@echo "Generating overall coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Overall coverage report: $(COVERAGE_DIR)/coverage.html"

coverage-all: ## Generate coverage reports for all packages
	@echo "Generating coverage reports for all packages..."
	@mkdir -p $(COVERAGE_DIR)
	@echo "# Package Coverage Report" > $(COVERAGE_DIR)/coverage-summary.md
	@echo "" >> $(COVERAGE_DIR)/coverage-summary.md
	@echo "| Package | Coverage | Target | Status |" >> $(COVERAGE_DIR)/coverage-summary.md
	@echo "|---------|----------|--------|--------|" >> $(COVERAGE_DIR)/coverage-summary.md
	@$(MAKE) coverage-app coverage-cache coverage-config coverage-converter coverage-domain coverage-fetcher coverage-git coverage-llm coverage-output coverage-renderer coverage-strategies coverage-cmd
	@echo "Coverage summary: $(COVERAGE_DIR)/coverage-summary.md"

coverage-app: ## Generate coverage for app package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-app.out -covermode=atomic ./internal/app/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-app.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/app | %s%% | 85%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=85) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-cache: ## Generate coverage for cache package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-cache.out -covermode=atomic ./internal/cache/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-cache.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/cache | %s%% | 75%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=75) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-config: ## Generate coverage for config package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-config.out -covermode=atomic ./internal/config/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-config.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/config | %s%% | 85%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=85) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-converter: ## Generate coverage for converter package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-converter.out -covermode=atomic ./internal/converter/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-converter.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/converter | %s%% | 85%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=85) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-domain: ## Generate coverage for domain package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-domain.out -covermode=atomic ./internal/domain/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-domain.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/domain | %s%% | 85%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=85) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-fetcher: ## Generate coverage for fetcher package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-fetcher.out -covermode=atomic ./internal/fetcher/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-fetcher.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/fetcher | %s%% | 70%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=70) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-git: ## Generate coverage for git package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-git.out -covermode=atomic ./internal/git/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-git.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/git | %s%% | 80%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=80) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-llm: ## Generate coverage for llm package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-llm.out -covermode=atomic ./internal/llm/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-llm.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/llm | %s%% | 80%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=80) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-output: ## Generate coverage for output package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-output.out -covermode=atomic ./internal/output/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-output.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/output | %s%% | 80%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=80) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-renderer: ## Generate coverage for renderer package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-renderer.out -covermode=atomic ./internal/renderer/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-renderer.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/renderer | %s%% | 40%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=40) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-strategies: ## Generate coverage for strategies package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-strategies.out -covermode=atomic ./internal/strategies/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-strategies.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| internal/strategies | %s%% | 85%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=85) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-cmd: ## Generate coverage for cmd/repodocs package
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage-cmd.out -covermode=atomic ./cmd/repodocs/...
	@COV=$$($(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage-cmd.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	printf "| cmd/repodocs | %s%% | 80%% | %s |\n" "$$COV" "$$(echo "$$COV" | awk '{if ($$1>=80) print "✅ PASS"; else print "❌ FAIL"}')" >> $(COVERAGE_DIR)/coverage-summary.md

coverage-view: ## View overall coverage HTML report
	@command -v xdg-open >/dev/null 2>&1 && xdg-open $(COVERAGE_DIR)/coverage.html || \
	command -v open >/dev/null 2>&1 && open $(COVERAGE_DIR)/coverage.html || \
	echo "Open $(COVERAGE_DIR)/coverage.html in your browser"

coverage-summary: ## Show coverage summary
	@if [ -f $(COVERAGE_DIR)/coverage-summary.md ]; then \
		cat $(COVERAGE_DIR)/coverage-summary.md; \
	else \
		echo "Coverage summary not found. Run 'make coverage-all' first."; \
	fi

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

## Release

release: ## Create and push a new release tag (usage: make release VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if ! echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "Error: VERSION must match semver format (e.g., v1.0.0)"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Working directory is not clean. Commit or stash changes first."; \
		exit 1; \
	fi
	@CURRENT_BRANCH=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$CURRENT_BRANCH" != "main" ] && [ "$$CURRENT_BRANCH" != "master" ]; then \
		echo "Warning: Not on main/master branch (current: $$CURRENT_BRANCH)"; \
		read -p "Continue anyway? [y/N] " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			exit 1; \
		fi; \
	fi
	@if git rev-parse "$(VERSION)" >/dev/null 2>&1; then \
		echo "Error: Tag $(VERSION) already exists"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Pushing tag $(VERSION) to origin..."
	@git push origin $(VERSION)
	@echo ""
	@echo "✓ Release $(VERSION) created and pushed!"
	@echo "  GitHub Actions will now build and publish the release."
	@echo "  Monitor: https://github.com/quantmind-br/repodocs-go/actions"

release-dry: ## Test release build locally without publishing
	@echo "Running dry-run release..."
	@which goreleaser > /dev/null || (echo "Installing goreleaser..." && go install github.com/goreleaser/goreleaser/v2@latest)
	goreleaser release --snapshot --clean
	@echo ""
	@echo "✓ Dry-run complete. Artifacts in ./dist/"

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
