APP_NAME := qwikkle-api

.PHONY: run dev build test

MIGRATIONS_DIR := internal/db/migrations

-include .env
export

run:
	go run ./cmd/api

dev:
	air

build:
	go build -o $(APP_NAME) ./cmd/api

test:
	go test ./...

migrate-up:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is not set. Export it or add it to ./ .env" && exit 1)
	go run github.com/pressly/goose/v3/cmd/goose@v3.25.0 -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" up

migrate-down:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is not set. Export it or add it to ./ .env" && exit 1)
	go run github.com/pressly/goose/v3/cmd/goose@v3.25.0 -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" down

migrate-status:
	@test -n "$(POSTGRES_DSN)" || (echo "POSTGRES_DSN is not set. Export it or add it to ./ .env" && exit 1)
	go run github.com/pressly/goose/v3/cmd/goose@v3.25.0 -dir $(MIGRATIONS_DIR) postgres "$(POSTGRES_DSN)" status

migrate-create:
	go run github.com/pressly/goose/v3/cmd/goose@v3.25.0 -dir $(MIGRATIONS_DIR) create $(name) sql
