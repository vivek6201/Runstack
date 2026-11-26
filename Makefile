# ==============================================================================
# Code-Runner Monorepo Makefile
# Multi-tenant, microVM-based code execution engine and SaaS dashboard
# ==============================================================================

SHELL := /bin/bash
.DEFAULT_GOAL := help

# Compose files and command
COMPOSE_FULL   := infra/docker-compose.full.yml
COMPOSE_FC     := infra/docker-compose.firecracker.yml
DOCKER_COMPOSE ?= $(shell which docker-compose 2>/dev/null || echo "docker compose")

# Colors for terminal output
CYAN   := \033[36m
GREEN  := \033[32m
YELLOW := \033[33m
RED    := \033[31m
RESET  := \033[0m

##@ 🚀 Getting Started

.PHONY: help
help: ## Display this interactive help menu
	@echo -e "$(CYAN)Code-Runner Makefile$(RESET)"
	@echo -e "Usage: $(GREEN)make [target]$(RESET)\n"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  $(CYAN)%-22s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ 🔥 Firecracker MicroVM (Production / KVM)

.PHONY: firecracker-setup
firecracker-setup: ## Download Firecracker binary, kernel, and build rootfs.ext4
	@echo -e "$(GREEN)==> Running Firecracker setup (binary, kernel, rootfs)...$(RESET)"
	./scripts/firecracker/download-firecracker.sh
	./scripts/firecracker/download-kernel.sh
	./scripts/firecracker/build-rootfs.sh

.PHONY: firecracker-up
firecracker-up: ## Start full production stack with Firecracker microVMs (/dev/kvm)
	@echo -e "$(GREEN)==> Starting Firecracker production stack...$(RESET)"
	$(DOCKER_COMPOSE) -f $(COMPOSE_FC) up --build -d

.PHONY: firecracker-down
firecracker-down: ## Stop Firecracker production stack
	@echo -e "$(YELLOW)==> Stopping Firecracker production stack...$(RESET)"
	$(DOCKER_COMPOSE) -f $(COMPOSE_FC) down

.PHONY: firecracker-logs
firecracker-logs: ## Follow logs for the Firecracker production stack
	$(DOCKER_COMPOSE) -f $(COMPOSE_FC) logs -f

.PHONY: firecracker-restart
firecracker-restart: firecracker-down firecracker-up ## Restart Firecracker production stack

##@ 🐳 Docker Full Stack (Local Execution Driver)

.PHONY: docker-up
docker-up: ## Start full stack using standard local subprocess driver (no KVM needed)
	@echo -e "$(GREEN)==> Starting full stack (local driver)...$(RESET)"
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) up --build -d

.PHONY: docker-down
docker-down: ## Stop full stack
	@echo -e "$(YELLOW)==> Stopping full stack...$(RESET)"
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) down

.PHONY: docker-logs
docker-logs: ## Follow logs for the full stack
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) logs -f

.PHONY: docker-restart
docker-restart: docker-down docker-up ## Restart full stack

##@ 📦 Databases & Infrastructure Only (for Native Local Dev)

.PHONY: infra-up
infra-up: ## Start only PostgreSQL and Redis containers for local development
	@echo -e "$(GREEN)==> Starting PostgreSQL & Redis...$(RESET)"
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) up -d postgres redis

.PHONY: infra-down
infra-down: ## Stop PostgreSQL and Redis containers
	@echo -e "$(YELLOW)==> Stopping PostgreSQL & Redis...$(RESET)"
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) stop postgres redis

.PHONY: infra-logs
infra-logs: ## Follow logs for PostgreSQL and Redis
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) logs -f postgres redis

##@ 💻 Local Development (Native Binaries)

.PHONY: dev-dashboard
dev-dashboard: ## Run Dashboard API service locally (port 8080)
	cd services/dashboard && go run ./cmd/server

.PHONY: dev-engine
dev-engine: ## Run Execution Engine service locally (port 8081)
	cd services/execution-engine && go run ./cmd/server

.PHONY: dev-web
dev-web: ## Run React frontend development server (port 5173)
	cd apps/web && bun dev

.PHONY: dev
dev: ## Start infra and provide native dev instructions
	@$(MAKE) infra-up
	@echo -e "\n$(GREEN)==> Infrastructure ready! Open separate terminals to run:$(RESET)"
	@echo -e "  $(CYAN)make dev-dashboard$(RESET)  (Dashboard API on :8080)"
	@echo -e "  $(CYAN)make dev-engine$(RESET)     (Execution Engine on :8081)"
	@echo -e "  $(CYAN)make dev-web$(RESET)        (Web UI on :5173)"

##@ 🧪 Testing & Verification

.PHONY: test
test: test-shared test-dashboard test-engine ## Run all Go test suites across all modules
	@echo -e "$(GREEN)==> All Go tests passed!$(RESET)"

.PHONY: test-shared
test-shared: ## Run tests in services/shared
	@echo -e "$(GREEN)==> Testing services/shared...$(RESET)"
	cd services/shared && go test -v -race ./...

.PHONY: test-dashboard
test-dashboard: ## Run tests in services/dashboard
	@echo -e "$(GREEN)==> Testing services/dashboard...$(RESET)"
	cd services/dashboard && go test -v -race ./...

.PHONY: test-engine
test-engine: ## Run tests in services/execution-engine
	@echo -e "$(GREEN)==> Testing services/execution-engine...$(RESET)"
	cd services/execution-engine && go test -v -race ./...

.PHONY: build
build: ## Build all Go binaries locally
	@echo -e "$(GREEN)==> Building all Go binaries...$(RESET)"
	mkdir -p bin
	cd services/dashboard && go build -o ../../bin/dashboard ./cmd/server
	cd services/execution-engine && go build -o ../../bin/execution-engine ./cmd/server
	cd services/execution-engine && GOOS=linux go build -o ../../bin/guest-agent-linux ./cmd/guest-agent
	@echo -e "$(GREEN)==> Binaries compiled to bin/$(RESET)"

##@ 🛠️ Database & Utilities

.PHONY: db-shell
db-shell: ## Open interactive PostgreSQL shell inside container
	docker exec -it coderunner-postgres psql -U postgres -d coderunner

.PHONY: redis-shell
redis-shell: ## Open interactive Redis CLI inside container
	docker exec -it coderunner-redis redis-cli

.PHONY: db-migrate migration-gen migration-apply migration-status migration-hash
db-migrate: migration-apply ## Alias for migration-apply

migration-gen: ## Generate a new Atlas SQL migration (usage: make migration-gen name=add_feature)
	@if [ -z "$(name)" ]; then \
		echo "Error: 'name' variable is required. Example: make migration-gen name=init"; \
		exit 1; \
	fi
	atlas migrate diff $(name) --env local

migration-apply: ## Apply pending Atlas migrations to database
	atlas migrate apply --env local

migration-status: ## Check Atlas migration status
	atlas migrate status --env local

migration-hash: ## Recompute migrations/atlas.sum integrity hash
	atlas migrate hash --dir file://migrations

.PHONY: clean
clean: ## Remove temporary build binaries and socket artifacts
	rm -rf bin/
	rm -f /tmp/fc-*.sock /tmp/vsock-*.sock /tmp/fc-*-console.log
	@echo -e "$(GREEN)==> Cleanup complete.$(RESET)"
