.PHONY: up down server fmt test migrate-create

UP ?= docker compose up -d
DOWN ?= docker compose down
SERVER_CMD ?= go run ./server/cmd

up:
	$(UP)

down:
	$(DOWN)

force-up:
	$(DOWN) -v
	$(UP)

server:
	@echo "Starting backend server..."
	@set -a && . ./.env && set +a && $(SERVER_CMD)

fmt:
	go fmt ./...

test:
	go test ./...

migrate-create:
	@read -p "Migration name: " name; \
	if [ -z "$$name" ]; then echo "Migration name is required"; exit 1; fi; \
	timestamp=$$(date +%Y%m%d%H%M%S); \
	safe_name=$$(echo "$$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_]/_/g' | sed 's/_\+/_/g' | sed 's/^_//; s/_$$//'); \
	mkdir -p server/migrations; \
	up_file="server/migrations/$$timestamp_$$safe_name.up.sql"; \
	down_file="server/migrations/$$timestamp_$$safe_name.down.sql"; \
	printf -- '-- Write your UP migration here\n' > "$$up_file"; \
	printf -- '-- Write your DOWN migration here\n' > "$$down_file"; \
	echo "Created $$up_file and $$down_file"
