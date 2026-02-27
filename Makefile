# ── Ziga-Kit API Makefile ─────────────────────────────────────────────────────
# Usage: make <target>

.PHONY: help dev build migrate migrate-down db-up db-down db-reset lint tidy

# Default target
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Local Dev ─────────────────────────────────────────────────────────────────

dev: ## Run the API with hot-reload (requires: go install github.com/air-verse/air@latest)
	air -c .air.toml

build: ## Compile the API binary to ./bin/api
	@mkdir -p bin
	go build -o bin/api ./cmd/api

run: build ## Build and run the binary directly
	./bin/api

# ── Database ──────────────────────────────────────────────────────────────────

db-up: ## Start local Postgres via Docker Compose
	docker compose up -d postgres
	@echo "⏳ Waiting for Postgres to be ready..."
	@until docker compose exec postgres pg_isready -U zigakit -d zigakit > /dev/null 2>&1; do sleep 1; done
	@echo "✅ Postgres is ready on localhost:5432"

db-down: ## Stop local Postgres
	docker compose down

db-reset: db-down ## Wipe volume and restart Postgres (destructive!)
	docker compose down -v
	$(MAKE) db-up

migrate: ## Apply all SQL migrations in order
	@echo "🔄 Running migrations..."
	@set -a && . ./.env && set +a && \
	for f in migrations/*.sql; do \
		echo "  → $$f"; \
		psql "$$DATABASE_URL" -f "$$f" --single-transaction; \
	done
	@echo "✅ Migrations complete"

# Quick shortcut: spin up DB then migrate
setup: db-up migrate ## Start DB and run migrations (first-time setup)

# ── Code Quality ──────────────────────────────────────────────────────────────

tidy: ## Tidy and verify go modules
	go mod tidy
	go mod verify

lint: ## Run golangci-lint (requires: brew install golangci-lint)
	golangci-lint run ./...

test: ## Run all tests
	go test -race -count=1 ./...