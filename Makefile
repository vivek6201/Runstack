SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE_FULL   := infra/docker-compose.full.yml
COMPOSE_FC     := infra/docker-compose.firecracker.yml
DOCKER_COMPOSE ?= docker compose

.PHONY: help infra-up infra-down dev-dashboard dev-engine dev-web docker-up docker-down firecracker-setup firecracker-up firecracker-down build test clean migration-gen migration-apply migration-status

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development Targets:"
	@echo "  infra-up           Start PostgreSQL & Redis containers"
	@echo "  infra-down         Stop PostgreSQL & Redis containers"
	@echo "  dev-dashboard      Run Dashboard API locally (:8080)"
	@echo "  dev-engine         Run Execution Engine locally (:8081)"
	@echo "  dev-web            Run Web UI locally (:5173)"
	@echo "  build              Build all Go binaries to bin/"
	@echo "  test               Run all Go test suites with -race"
	@echo "  clean              Remove compiled binaries and Firecracker sockets"
	@echo ""
	@echo "Docker & Firecracker Targets:"
	@echo "  docker-up          Start full stack (local execution driver)"
	@echo "  docker-down        Stop full stack"
	@echo "  firecracker-setup  Download Firecracker binary, kernel, and build rootfs"
	@echo "  firecracker-up     Start Firecracker microVM stack (/dev/kvm)"
	@echo "  firecracker-down   Stop Firecracker stack"
	@echo ""
	@echo "Migration Targets (Atlas CLI):"
	@echo "  migration-gen      Generate migration (usage: make migration-gen name=migration_name)"
	@echo "  migration-apply    Apply pending migrations"
	@echo "  migration-status   Check migration status"

infra-up:
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) up -d postgres redis

infra-down:
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) stop postgres redis

dev-dashboard:
	cd services/dashboard && go run ./cmd/server

dev-engine:
	cd services/execution-engine && go run ./cmd/server

dev-web:
	cd apps/web && bun dev

docker-up:
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) up --build -d

docker-down:
	$(DOCKER_COMPOSE) -f $(COMPOSE_FULL) down

firecracker-setup:
	./scripts/firecracker/download-firecracker.sh
	./scripts/firecracker/download-kernel.sh
	./scripts/firecracker/build-rootfs.sh

firecracker-up:
	$(DOCKER_COMPOSE) -f $(COMPOSE_FC) up --build -d

firecracker-down:
	$(DOCKER_COMPOSE) -f $(COMPOSE_FC) down

build:
	mkdir -p bin
	cd services/dashboard && go build -o ../../bin/dashboard ./cmd/server
	cd services/execution-engine && go build -o ../../bin/execution-engine ./cmd/server
	cd services/execution-engine && GOOS=linux go build -o ../../bin/guest-agent-linux ./cmd/guest-agent

test:
	cd services/shared && go test -v -race ./...
	cd services/dashboard && go test -v -race ./...
	cd services/execution-engine && go test -v -race ./...

clean:
	rm -rf bin/
	rm -f /tmp/fc-*.sock /tmp/vsock-*.sock /tmp/fc-*-console.log

migration-gen:
	@if [ -z "$(name)" ]; then \
		echo "Error: 'name' variable is required. Example: make migration-gen name=add_feature"; \
		exit 1; \
	fi
	atlas migrate diff $(name) --env local

migration-apply:
	atlas migrate apply --env local

migration-status:
	atlas migrate status --env local
