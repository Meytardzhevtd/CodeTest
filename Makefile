.PHONY: up down server fmt test

UP ?= docker compose up -d
DOWN ?= docker compose down
SERVER_CMD ?= go run ./server/cmd

up:
	$(UP)

down:
	$(DOWN)

server:
	@echo "Starting backend server..."
	@set -a && . ./.env && set +a && $(SERVER_CMD)

fmt:
	go fmt ./...

test:
	go test ./...
