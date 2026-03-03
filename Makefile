APP_NAME := qwikkle-api

.PHONY: run build test

run:
	go run ./cmd/api

build:
	go build -o $(APP_NAME) ./cmd/api

test:
	go test ./...

