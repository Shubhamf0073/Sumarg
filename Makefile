SHELL := /bin/bash
.DEFAULT_GOAL := help

DEV_COMPOSE := docker compose -f infra/docker-compose.dev.yml
BIN := $(CURDIR)/.bin
export PATH := $(BIN):$(PATH)

# Local development settings (production values live only in /srv/schoolfleet/compose/.env and db.env)
DB_PORT ?= 5434
DEV_PW ?= dev
db_url = postgres://$(1):$(DEV_PW)@localhost:$(DB_PORT)/$(2)?sslmode=disable
export APP_ENV ?= local
# APP_DOMAIN unset locally: the portal origin is the Vite dev server (config.PortalOrigins)
export APP_DOMAIN ?=
export LOG_FORMAT ?= text
export SECRETS_KEY ?= 0000000000000000000000000000000000000000000000000000000000000000
export FILES_DIR ?= $(CURDIR)/.data/files
# Comments go on their own lines: make keeps the spaces before a trailing comment as part of the value.
# MQTT_PLAIN_PORT: dev only, plain MQTT for the simulator on localhost
export MQTT_PLAIN_PORT ?= 1883
export MQTT_PORT ?= 8883
# SMS_PROVIDER=stub: OTP codes are logged locally, never sent
export SMS_PROVIDER ?= stub
# FCM_PROJECT_ID empty: pushes are logged, not sent
export FCM_PROJECT_ID ?=
export DB_OWNER_PASSWORD ?= $(DEV_PW)

SQLC_VERSION := v1.31.1
GOLANGCI_VERSION := v2.15.0

.PHONY: help tools dev-db dev-db-down migrate seed run web app sim test test-integration fw-test fw-build lint fmt sqlc gen-api schema-diff e2e

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-17s %s\n", $$1, $$2}'

tools: ## Install Go dev tools into ./.bin (sqlc, golangci-lint, govulncheck, goimports)
	mkdir -p $(BIN)
	GOBIN=$(BIN) go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	GOBIN=$(BIN) go install golang.org/x/vuln/cmd/govulncheck@latest
	GOBIN=$(BIN) go install golang.org/x/tools/cmd/goimports@latest

dev-db: ## Start local Postgres 17 on 127.0.0.1:5434 (databases schoolfleet + sf_test, schoolfleet roles)
	$(DEV_COMPOSE) up -d --wait

dev-db-down: ## Stop local Postgres (data kept in a docker volume)
	$(DEV_COMPOSE) down

migrate: ## goose up as sf_owner on the dev and test databases
	cd backend && SVC_NAME=migrate DATABASE_URL='$(call db_url,sf_owner,schoolfleet)' go run ./cmd/migrate up
	cd backend && SVC_NAME=migrate DATABASE_URL='$(call db_url,sf_owner,sf_test)' go run ./cmd/migrate up

seed: ## Dev sample data: Previa admin (prints an enrolment link) + 2 sample tenants
	cd backend && SVC_NAME=migrate DATABASE_URL='$(call db_url,sf_owner,schoolfleet)' go run ./cmd/migrate seed --sample

run: ## API + worker + ingest in one process (:8000 HTTP, :1883/:8883 MQTT)
	mkdir -p $(FILES_DIR)
	cd backend && SVC_NAME=schoolfleet DB_MAX_CONNS=4 \
	  DATABASE_URL='$(call db_url,sf_app,schoolfleet)' PLATFORM_DATABASE_URL='$(call db_url,sf_platform,schoolfleet)' \
	  go run ./cmd/schoolfleet serve --mode=all

web: ## Vite dev server on :5173 (proxies /api and /b to :8000)
	cd frontend && npm run dev

test: ## Unit tests (no database) + portal/app typecheck + firmware logic tests
	cd backend && go test ./...
	cd frontend && npm run typecheck
	cd mobile && npm run typecheck
	$(MAKE) fw-test

test-integration: ## Isolation, numbering, permissions on sf_test with two tenants (needs make dev-db migrate)
	cd backend && TEST_APP_URL='$(call db_url,sf_app,sf_test)' \
	  TEST_PLATFORM_URL='$(call db_url,sf_platform,sf_test)' \
	  go test -tags integration -count=1 ./...

lint: ## golangci-lint (incl. architecture rules) + Biome (portal, app)
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint
	cd mobile && npm run lint

fmt: ## Format both sides
	cd backend && golangci-lint fmt ./...
	cd frontend && npm run fmt

sqlc: ## Regenerate typed Go from backend/queries/**/*.sql
	cd backend && sqlc generate

gen-api: ## OpenAPI from the Go handlers → portal and app types
	mkdir -p frontend/src/api mobile/src/api
	cd backend && SVC_NAME=schoolfleet go run ./cmd/schoolfleet openapi > ../frontend/src/api/openapi.json
	cd frontend && npm run gen:api
	cd mobile && npm run gen:api

schema-diff: ## Prove goose migrations == docs/schema.sql (needs make dev-db)
	bash infra/dev/schema-diff.sh

e2e: ## Playwright UAT smoke against the local stack (run + web must be running)
	cd frontend && npx playwright test

app: ## Expo dev server for the parent/driver app
	cd mobile && npx expo start

sim: ## Device simulator: virtual buses over MQTT (SCENARIO=morning-route by default; M3.6)
	cd tools/simulator && go run ./cmd/sim --scenario $${SCENARIO:-morning-route}

fw-test: ## Firmware logic tests on the host (PlatformIO native env; falls back to g++ if pio is missing)
	@if command -v pio >/dev/null 2>&1; then cd firmware && pio test -e native; \
	else echo "pio not found: compile-checking firmware/src/logic with g++"; \
	  g++ -std=c++17 -Wall -Wextra -Werror -DUNIT_TEST -fsyntax-only firmware/src/logic/*.cpp; fi

fw-build: ## Build both firmware targets (needs board_pins.h filled for that HW_REV)
	cd firmware && pio run -e sb_proto -e sb_r1
