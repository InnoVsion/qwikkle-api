APP_NAME := qwikkle-api

.PHONY: run dev build test

run:
	go run ./cmd/api

dev:
	air

build:
	go build -o $(APP_NAME) ./cmd/api

test:
	go test ./...

