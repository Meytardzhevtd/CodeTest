# CodeTest

**Онлайн платформа длья проверки решений олимпиадных задач.**

[![Go](https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white)](go.mod)
[![gRPC](https://img.shields.io/badge/gRPC-protobuf-244c5a?logo=google&logoColor=white)](checker/proto/judge.proto)
[![Kafka](https://img.shields.io/badge/kafka-3.8-231F20?logo=apachekafka&logoColor=white)](docker-compose.yml)
[![PostgreSQL](https://img.shields.io/badge/postgres-16-4169E1?logo=postgresql&logoColor=white)](docker-compose.yml)
[![Docker](https://img.shields.io/badge/sandbox-docker--in--docker-2496ED?logo=docker&logoColor=white)](checker/internal/worker/docker.go)
[![React](https://img.shields.io/badge/react-19-149ECA?logo=react&logoColor=white)](client/package.json)
[![License](https://img.shields.io/badge/license-MIT-lightgrey)](LICENSE)

<img src="./demo.gif" width="700" />

## Архитектура

```
 ┌──────────┐  REST + JWT  ┌──────────┐  submissions   ┌─────────────┐
 │  Клиент  │─────────────▶│  Сервер  │───────────────▶│ Координатор │
 │  React   │◀─────────────│  go-chi  │◀───────────────│             │
 └──────────┘   вердикт    └────┬─────┘    results     └──────┬──────┘
                                │           (Kafka)           │
                        ┌───────┴───────┐                     │ gRPC, pull
                        │   Postgres    │                     ▼
                        │     MinIO     │              ┌─────────────┐
                        └───────────────┘              │  Воркеры ×3 │
                                                       └──────┬──────┘
                                                              │ Docker API
                                                              ▼
                                                  ┌───────────────────────┐
                                                  │ контейнер на сабмишн  │
                                                  └───────────────────────┘
```

Сервер принимает решение по REST, сохраняет его и кладёт в Kafka — на этом его
работа заканчивается. Дальше координатор раздаёт задачи пулу воркеров по gRPC, а
воркеры исполняют код в контейнерах на отдельном Docker-демоне. **Процесс,
который принимает HTTP-трафик, никогда не запускает чужой код**, а процесс,
который его запускает, не имеет доступа ни к базе, ни к хранилищу, ни к сети.

Воркеры забирают задачи сами (pull), поэтому пул масштабируется просто числом
реплик — координатору о них ничего знать не нужно.

> 📐 Устройство системы целиком — границы сервисов, контракты, модель данных,
> гарантии доставки и известные ограничения — в
> **[doc/ARCHITECTURE.md](doc/ARCHITECTURE.md)**.

---

## Submit service


| Ограничение | Значение |
|---|---|
| Сеть | отключена полностью (`network=none`) |
| Память | 256 MiB, своп не добавляется поверх лимита |
| CPU | 1 vCPU |
| Процессы | 64 — fork-бомба упирается в потолок |
| Время | 5 с на тест, 40 с на компиляцию |
| Демон | отдельный Docker-in-Docker, без доступа к демону хоста |


| Язык | Образ | Особенности |
|---|---|---|
| Python | `python:3.12-slim` | без компиляции |
| C++ | `gcc:13` | `g++ -O2` |
| Go | `golang:1.23` | общий том с кешем сборки — иначе каждый сабмишн пересобирает stdlib |


---

## Стек

| Слой | Технологии |
|---|---|
| Бэкенд | Go 1.25, [chi](https://github.com/go-chi/chi), [pgx](https://github.com/jackc/pgx), [golang-migrate](https://github.com/golang-migrate/migrate) |
| Асинхронность | Apache Kafka (KRaft), gRPC + Protocol Buffers |
| Базы данных | PostgreSQL 16, MinIO (S3-совместимое), Redis (LRU-кеш тестов) |
| Фронтенд | React 19, Vite, TypeScript, CodeMirror |

---

## Запуск

```bash
cp .env.example .env
make up          # поднимает всё: инфраструктуру, сервер, координатор и 3 воркера

cd client && npm install && npm run dev
```

Клиент — на `http://localhost:5173`, API — на `:8080`.

> При первом запуске воркеры тянут образы языков (`gcc:13` и `golang:1.23` — это
> больше двух гигабайт) в свой изолированный демон. Загрузка идёт в фоне и не
> блокирует старт, но первые сабмишны на C++ и Go подождут.

Все переменные окружения — в [`.env.example`](.env.example).

---

## Разработка

```bash
make test          # go test ./...
make fmt           # go fmt ./...
golangci-lint run  # конфигурация в .golangci.yml

make migrate-create   # заготовка новой миграции
make force-up         # пересоздать стек, удалив тома
```

---

## Лицензия

MIT — см. [LICENSE](LICENSE).
