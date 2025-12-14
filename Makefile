# death-level-tracker Makefile
# Professional build and development automation

# ============================================================================
# Variables
# ============================================================================

BINARY_NAME := death-level-tracker
BUILD_DIR := bin
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOMOD := $(GO) mod

# Docker Compose
DOCKER_COMPOSE := docker-compose
DEV_SERVICE := dev
BOT_SERVICE := bot prometheus grafana
MIGRATE_SERVICE := migrate

# Build flags
LDFLAGS := -w -s
BUILD_FLAGS := CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)"

# Helper variables for string manipulation
comma := ,
space := $(subst ,, )

# ============================================================================
# PHONY Targets
# ============================================================================

.PHONY: all help
.PHONY: tidy fmt vet lint
.PHONY: test coverage coverage-html
.PHONY: build clean
.PHONY: dev-up dev-down dev-shell dev-test dev-coverage dev-sqlc
.PHONY: up down logs
.PHONY: db-reset db-hash db-new
.PHONY: sqlc

# ============================================================================
# Default Target
# ============================================================================

all: tidy fmt vet build ## Run quality checks and build

# ============================================================================
# Help
# ============================================================================

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ============================================================================
# Development
# ============================================================================

tidy: ## Run go mod tidy
	$(GOMOD) tidy

fmt: ## Format Go code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: vet ## Run linters (alias for vet)

# ============================================================================
# Testing & Coverage
# ============================================================================

# Packages to include in coverage (excludes sqlc-generated code)
COVER_PKGS := ./cmd/... ./internal/config/... ./internal/formatting/... ./internal/handlers/... ./internal/tracker/... ./internal/tibiadata/...

test: ## Run tests (usage: make test [death-level-tracker/internal/config])
	@$(if $(filter-out $@,$(MAKECMDGOALS)), \
		$(GOTEST) -v $(filter-out $@,$(MAKECMDGOALS)), \
		$(GOTEST) -v ./...)

coverage: ## Run coverage (excludes sqlc-generated files)
	@$(GOTEST) -coverprofile=$(COVERAGE_FILE) -coverpkg=$(subst $(space),$(comma),$(COVER_PKGS)) ./...
	@$(GO) tool cover -func=$(COVERAGE_FILE)
	@rm $(COVERAGE_FILE)

coverage-html: ## Generate HTML coverage report (excludes sqlc-generated files)
	@echo "Generating HTML coverage report..."
	@$(GOTEST) -coverprofile=$(COVERAGE_FILE) -coverpkg=$(subst $(space),$(comma),$(COVER_PKGS)) ./...
	@$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

# Catch-all target for positional parameters
%:
	@:

# ============================================================================
# Code Generation
# ============================================================================

sqlc: ## Generate code from SQL
	sqlc generate

# ============================================================================
# Build
# ============================================================================

build: ## Build the application binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/bot
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "Clean complete"

# ============================================================================
# Docker Development Environment
# ============================================================================

dev-up: ## Start development environment
	$(DOCKER_COMPOSE) up -d --build $(DEV_SERVICE)

dev-down: ## Stop development environment
	$(DOCKER_COMPOSE) stop $(DEV_SERVICE)

dev-shell: ## Open shell in development container
	$(DOCKER_COMPOSE) exec $(DEV_SERVICE) /bin/sh

dev-test: ## Run tests in Docker (usage: make dev-test [death-level-tracker/internal/config])
	@$(if $(filter-out $@,$(MAKECMDGOALS)), \
		$(DOCKER_COMPOSE) exec -T $(DEV_SERVICE) go test -v $(filter-out $@,$(MAKECMDGOALS)), \
		$(DOCKER_COMPOSE) exec -T $(DEV_SERVICE) go test -v ./...)

dev-coverage: ## Run coverage in Docker (usage: make dev-coverage [death-level-tracker/internal/tracker])
	@$(if $(filter-out $@,$(MAKECMDGOALS)), \
		$(DOCKER_COMPOSE) exec -T $(DEV_SERVICE) sh -c 'go test -coverprofile=coverage.out $(filter-out $@,$(MAKECMDGOALS)) && go tool cover -func=coverage.out', \
		$(DOCKER_COMPOSE) exec -T $(DEV_SERVICE) sh -c 'go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out')

dev-sqlc: ## Generate code in Docker
	$(DOCKER_COMPOSE) exec -T $(DEV_SERVICE) sqlc generate -f config/sqlc.yaml

# ============================================================================
# Docker Production
# ============================================================================

up: ## Start production services
	$(DOCKER_COMPOSE) up -d --build $(BOT_SERVICE)

down: ## Stop all services
	$(DOCKER_COMPOSE) down

logs: ## View production logs
	$(DOCKER_COMPOSE) logs -f $(BOT_SERVICE)

# ============================================================================
# Database Operations
# ============================================================================

db-reset: ## Reset database (WARNING: destroys all data)
	@echo "Resetting database..."
	@$(DOCKER_COMPOSE) down -v
	@echo "Database reset complete"

db-hash: ## Update atlas.sum hash file
	$(DOCKER_COMPOSE) run --rm $(MIGRATE_SERVICE) migrate hash --dir "file:///sql/migrations"

db-new: ## Create new migration file
	@read -p "Enter migration name: " name; \
	timestamp=$$(date +%Y%m%d%H%M%S); \
	filename="sql/migrations/$${timestamp}_$${name}.sql"; \
	touch $$filename; \
	echo "Created $$filename"
