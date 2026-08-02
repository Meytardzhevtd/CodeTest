# read -p и арифметика 10#NNN — bash, а не POSIX sh.
SHELL := /bin/bash

# Пакеты Go, принадлежащие проекту. Не ./...: внутри client/node_modules лежит
# чужой Go-код (flatted), который иначе попадает в сборку, тесты и линтер.
# На CI его нет (node_modules в .gitignore), поэтому там ./... эквивалентен.
GO_PKGS        ?= ./checker/... ./pkg/... ./server/...
CLIENT_DIR     ?= client
MIGRATIONS_DIR ?= server/migrations
COMPOSE        ?= docker compose
MODULE         ?= github.com/meytardzhevtd/CodeTest

.DEFAULT_GOAL := help

.PHONY: help up down force-up logs build-images \
        server coordinator worker \
        build vet test test-race lint fmt tidy tidy-check \
        client-install client-dev client-build client-lint \
        proto migrate-create ci

help: ## Показать список команд
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ─── Инфраструктура ──────────────────────────────────────────────────────────

up: ## Поднять весь стек (инфраструктура, сервер, координатор, воркеры)
	$(COMPOSE) up -d

down: ## Остановить стек, сохранив тома
	$(COMPOSE) down

force-up: ## Пересоздать стек с нуля — удаляет тома вместе с данными
	$(COMPOSE) down -v
	$(COMPOSE) up -d

logs: ## Смотреть логи всех сервисов
	$(COMPOSE) logs -f

build-images: ## Собрать docker-образы, не поднимая стек
	$(COMPOSE) build

# ─── Запуск вне Docker ───────────────────────────────────────────────────────
# Полезно, когда через `make up` поднята только инфраструктура, а сервисы
# гоняются локально под отладчиком.

server: ## Запустить сервер локально, с переменными из .env
	@echo "Starting backend server..."
	@set -a && . ./.env && set +a && go run ./server/cmd

coordinator: ## Запустить координатор локально, с переменными из .env
	@set -a && . ./.env && set +a && go run ./checker/cmd/coordinator

worker: ## Запустить воркер локально, с переменными из .env
	@set -a && . ./.env && set +a && go run ./checker/cmd/worker

# ─── Go ──────────────────────────────────────────────────────────────────────

build: ## Собрать все Go-пакеты
	go build $(GO_PKGS)

vet: ## go vet
	go vet $(GO_PKGS)

test: ## Юнит-тесты (интеграционные сами пропускаются без MinIO)
	go test $(GO_PKGS)

test-race: ## Тесты с детектором гонок (нужен gcc: -race требует cgo)
	CGO_ENABLED=1 go test -race $(GO_PKGS)

lint: ## golangci-lint по конфигу .golangci.yml
	golangci-lint run $(GO_PKGS)

fmt: ## Форматирование
	go fmt $(GO_PKGS)

tidy: ## Привести go.mod/go.sum в порядок
	go mod tidy

tidy-check: ## Упасть, если go.mod/go.sum разошлись с кодом (правит файлы!)
	go mod tidy
	git diff --exit-code go.mod go.sum

# ─── Клиент ──────────────────────────────────────────────────────────────────

client-install: ## Установить зависимости строго по package-lock.json
	cd $(CLIENT_DIR) && npm ci

client-dev: ## Дев-сервер Vite на http://localhost:5173
	cd $(CLIENT_DIR) && npm run dev

client-build: ## Проверка типов и продакшн-сборка (tsc -b && vite build)
	cd $(CLIENT_DIR) && npm run build

client-lint: ## ESLint
	cd $(CLIENT_DIR) && npm run lint

# ─── Кодогенерация ───────────────────────────────────────────────────────────

proto: ## Перегенерировать judgepb из checker/proto/judge.proto
	protoc \
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		checker/proto/judge.proto

# ─── Миграции ────────────────────────────────────────────────────────────────

migrate-create: ## Создать пару .up.sql/.down.sql со следующим номером
	@read -p "Migration name: " name; \
	if [ -z "$$name" ]; then echo "Migration name is required"; exit 1; fi; \
	safe_name=$$(echo "$$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_]/_/g; s/_\+/_/g; s/^_//; s/_$$//'); \
	if [ -z "$$safe_name" ]; then echo "Migration name has no usable characters"; exit 1; fi; \
	mkdir -p $(MIGRATIONS_DIR); \
	last=$$(ls $(MIGRATIONS_DIR) 2>/dev/null | sed -n 's/^\([0-9]\{3\}\)_.*/\1/p' | sort -n | tail -1); \
	next=$$(printf '%03d' $$(( 10#$${last:-0} + 1 ))); \
	up_file="$(MIGRATIONS_DIR)/$${next}_$${safe_name}.up.sql"; \
	down_file="$(MIGRATIONS_DIR)/$${next}_$${safe_name}.down.sql"; \
	printf -- '-- Write your UP migration here\n' > "$$up_file"; \
	printf -- '-- Write your DOWN migration here\n' > "$$down_file"; \
	echo "Created $$up_file and $$down_file"

# ─── Всё сразу ───────────────────────────────────────────────────────────────

ci: build vet test lint client-lint client-build ## Прогнать локально то, что гоняет CI
