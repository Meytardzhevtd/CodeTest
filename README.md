# CodeTest

Online judge platform: submit source code against a task's test suite and get a verdict back — powered by an asynchronous, Kafka-driven checking pipeline.

[![Go](https://img.shields.io/badge/go-1.25-00ADD8?logo=go&logoColor=white)](go.mod)
[![Kafka](https://img.shields.io/badge/kafka-3.8-231F20?logo=apachekafka&logoColor=white)](docker-compose.yml)
[![PostgreSQL](https://img.shields.io/badge/postgres-16-4169E1?logo=postgresql&logoColor=white)](docker-compose.yml)
[![React](https://img.shields.io/badge/react-19-149ECA?logo=react&logoColor=white)](client/package.json)
[![License](https://img.shields.io/badge/license-MIT-lightgrey)](LICENSE)

## Architecture

```
┌──────────┐   REST + JWT    ┌───────────────┐
│  Client  │ ───────────────▶│    Server     │──── Postgres (users, tasks, submissions)
│ (React)  │◀─────────────── │  (go-chi API) │──── MinIO (test case fixtures)
└──────────┘    verdicts     └───────┬───▲───┘
                                      │   │
                          submissions │   │ results
                                      ▼   │
                              ┌──────────────────┐
                              │       Kafka       │
                              └──────────┬───▲────┘
                                         │   │
                             submissions │   │ results
                                         ▼   │
                              ┌──────────────────┐
                              │    Coordinator    │
                              │ (checker service) │
                              └──────────────────┘
```

The server never runs untrusted code itself. It accepts a submission over REST, persists it, and publishes it to the `submissions` topic. The **coordinator** consumes that topic, judges the solution, and publishes a verdict to the `submission-results` topic. The server consumes results and makes them available through the same REST API the client already polls — the two services never talk to each other directly, only through Kafka, so either can restart or scale independently without losing in-flight work.

## Services

| Service | Path | Responsibility |
|---|---|---|
| **server** | `server/` | Auth, task management, submission intake and status, REST API |
| **coordinator** | `checker/cmd/coordinator` | Consumes submissions, judges them, publishes verdicts |
| **client** | `client/` | React SPA consuming the REST API |
| **pkg/kafka** | `pkg/kafka` | Shared producer/consumer/admin wrappers and message contracts |
| **pkg/storage** | `pkg/storage` | MinIO-backed test case storage |

## Stack

- **Go** — server & coordinator, [chi](https://github.com/go-chi/chi) router, [pgx](https://github.com/jackc/pgx) driver
- **PostgreSQL** — users, tasks, submissions
- **Apache Kafka** — asynchronous submission/result pipeline
- **MinIO** — S3-compatible object storage for test cases
- **React + Vite + TypeScript** — frontend

## Getting started

**Prerequisites:** Go 1.25+, Docker, Node.js (for the client)

```bash
# infrastructure: postgres, kafka, minio
cp .env.example .env
make up

# server (runs migrations, ensures kafka topics, serves the API)
make server

# coordinator (separate process)
go run ./checker/cmd/coordinator

# client
cd client && npm install && npm run dev
```

The API listens on `:8080` by default; see `.env.example` for all configuration options.

## API overview

| Endpoint | Auth | Description |
|---|---|---|
| `POST /api/auth/register` | — | Create an account |
| `POST /api/auth/login` | — | Obtain a JWT |
| `GET/POST /api/tasks` | JWT | List / create tasks |
| `POST /api/submit` | JWT | Submit a solution for judging |
| `GET /api/submit?id=` | JWT | Poll a submission's verdict |

## Development

```bash
make test   # go test ./...
make fmt    # go fmt ./...
```

## License

MIT — see [LICENSE](LICENSE).
