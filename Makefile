.DEFAULT_GOAL := help

.PHONY: help fmt test test-go test-web typecheck build check up down

help: ## List canonical commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "%-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format all source
	@if test -d cmd || test -d internal; then gofmt -w $$(find cmd internal -name '*.go' 2>/dev/null); fi
	@if test -f web/package.json; then cd web && bun run format; fi

test-go: ## Run Go tests
	@go test ./...

test-web: ## Run frontend tests
	@cd web && bun run test

test: test-go test-web ## Run unit tests

typecheck: ## Run static type checks
	@go vet ./...
	@cd web && bun run typecheck

build: ## Build API, renderer, and web
	@go build ./cmd/api ./cmd/renderer
	@cd web && bun run build

check: fmt test typecheck build ## Run required local gates

up: ## Start the reference stack
	@docker compose up --build

down: ## Stop the reference stack
	@docker compose down

