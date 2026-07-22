include .env.local
export

API_DIR := apps/api

.PHONY: dev run-api run-worker build test lint vet \
	migrate-up migrate-down migrate-status migrate-create \
	devbox-up devbox-patch-migrate devbox-forward-fe devbox-forward-fe-stop \
	seed-dev-catalog

dev:
	docker compose up -d
	$(MAKE) migrate-up

run-api:
	cd $(API_DIR) && go run ./cmd/api

run-worker:
	cd $(API_DIR) && go run ./cmd/worker

build:
	cd $(API_DIR) && go build ./...

test:
	cd $(API_DIR) && go test ./...

vet:
	cd $(API_DIR) && go vet ./...

lint: vet
	cd $(API_DIR) && gofmt -l .

migrate-up:
	cd $(API_DIR) && goose -dir db/migrations postgres "$$DATABASE_URL" up

migrate-down:
	cd $(API_DIR) && goose -dir db/migrations postgres "$$DATABASE_URL" down

migrate-status:
	cd $(API_DIR) && goose -dir db/migrations postgres "$$DATABASE_URL" status

migrate-create:
	cd $(API_DIR) && goose -dir db/migrations create $(name) sql

devbox-up:
	cd $(API_DIR) && devbox project up
	$(MAKE) devbox-patch-migrate

devbox-patch-migrate:
	./scripts/devbox-patch-migrate.sh

devbox-forward-fe:
	./scripts/devbox-port-forward-webstore-fe.sh

devbox-forward-fe-stop:
	./scripts/devbox-port-forward-webstore-fe-stop.sh

# Demo catalog data for devbox only — never run against production.
seed-dev-catalog:
	./scripts/devbox-seed-catalog.sh
