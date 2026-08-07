.DEFAULT_GOAL := help

.PHONY: help generate fmt test test-go test-web test-integration typecheck build coverage coverage-go coverage-web check start up bootstrap smoke smoke-journey down

help: ## List canonical commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "%-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

generate: ## Regenerate typed database queries
	@go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate

fmt: ## Format all source
	@if test -d cmd || test -d internal; then gofmt -w $$(find cmd internal -name '*.go' 2>/dev/null); fi
	@if test -f web/package.json; then cd web && bun run format; fi

test-go: ## Run Go tests
	@go test -race ./...

test-web: ## Run frontend tests
	@cd web && bun run test

test-integration: ## Run integration suites (requires Docker and Typst)
	@go test -race -tags=integration ./...

test: test-go test-web ## Run unit tests

typecheck: ## Run static type checks
	@go vet ./...
	@cd web && bun run typecheck

build: ## Build all Go commands and the web application
	@go build ./cmd/...
	@cd web && bun run build

coverage-go: ## Enforce the Go coverage floor
	@mkdir -p coverage
	@go test -covermode=atomic -coverprofile=coverage/go.out ./...
	@sh scripts/check-go-coverage.sh coverage/go.out

coverage-web: ## Enforce the frontend coverage floor
	@cd web && bun run test:coverage

coverage: coverage-go coverage-web ## Run coverage gates

check: ## Run repository, format, test, type, and build gates
	@sh scripts/check-repository.sh
	@$(MAKE) fmt
	@$(MAKE) test
	@$(MAKE) typecheck
	@$(MAKE) build

start: ## Bring the Etpeto up from nothing, including the first administrator
	@sh scripts/start.sh

up: ## Start the reference stack without provisioning
	@docker compose up --build -d

bootstrap: ## Provision the first administrator (password read from stdin)
	@docker compose run --rm --build --no-TTY bootstrap

smoke: ## Verify the running reference stack
	@sh scripts/smoke-compose.sh

smoke-journey: ## Exercise the main journey against the running stack
	@sh scripts/smoke-journey.sh

down: ## Stop the reference stack
	@docker compose down
