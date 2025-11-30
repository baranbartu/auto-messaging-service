# Auto Messaging Service

> See [`ASSUMPTIONS.md`](ASSUMPTIONS.md) for implementation tradeoffs and rationale.

A Go 1.25 backend that automatically sends messages by polling PostgreSQL and dispatching payloads to a configurable webhook. The app exposes endpoints to control the message loop and to list sent messages. Everything runs with Docker Compose, Redis caches accepted webhook ids, and PostgreSQL migrations are executed automatically on startup.

## Features
- Layered architecture (`cmd`, `internal/config|db|repository|service|scheduler|http`).
- Automatic 2-minute ticker that sends at most two new messages per iteration, honoring manual start/stop controls.
- PostgreSQL persistence with custom SQL migration runner executed at boot.
- Redis cache storing webhook `messageId` + sent timestamp metadata.
- REST API built with Chi, documented via OpenAPI (`api/swagger.yaml`).
- Graceful shutdown, structured configuration via environment variables, Docker Compose stack (app + Postgres + Redis).

## Getting Started
1. Copy the sample environment file and set the missing values (especially `WEBHOOK_URL`).
   ```bash
   cp .env.example .env
   # edit .env
   ```
2. Build and start the stack.
   ```bash
   docker compose up --build
   ```
   The API listens on `http://localhost:8083` by default, namespacing endpoints under `/api/v1`. Migrations run automatically before the server comes up.

### Local Development
- Run the server directly: `go run ./cmd/api` (ensure Postgres + Redis are available and `.env` exported).
- Execute tests / formatting: `go test ./...`, `gofmt -w $(find . -name '*.go' -not -path './vendor/*')`.

## Environment Variables
All parameters are .env configurable (see `.env.example`). Key values:
- `WEBHOOK_URL`: **required** URL of your webhook.site endpoint.
- `WEBHOOK_AUTH_KEY`: forwarded as `x-ins-auth-key` header.
- `HTTP_PORT`: API port (default 8083).
- `POSTGRES_*`: host/user/password/db/sslmode for DB access (defaults target docker compose PostgreSQL).
- `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`: Redis connection settings (compose redis requires `REDIS_PASSWORD`, default `automessagingredis`).
- `SCHEDULER_INTERVAL`: defaults to `2m` (ISO duration string), must stay at or above 2 minutes per requirements.
- `SCHEDULER_FETCH_LIMIT`: defaults to `2` messages per pass.
- `SERVER_SHUTDOWN_TIMEOUT`: graceful shutdown timeout (default `10s`).

## API
Swagger definition lives at `api/swagger.yaml` and is served by the app at `GET /api/v1/docs/swagger.yaml`.

### Endpoints
| Method | Path | Description |
| ------ | ---- | ----------- |
| `POST` | `/api/v1/control/start` | Start the scheduler loop. Already running -> HTTP 400. |
| `POST` | `/api/v1/control/stop` | Stop the scheduler. If already stopped -> HTTP 400. |
| `GET`  | `/api/v1/messages/sent?page=1&limit=20` | Paginated list of sent messages. |

### Example cURL
```bash
# Start automatic sending
curl -X POST http://localhost:8083/api/v1/control/start

# Stop automatic sending
curl -X POST http://localhost:8083/api/v1/control/stop

# List sent messages
curl "http://localhost:8083/api/v1/messages/sent?page=1&limit=10"
```

## Data Model
| Column | Type | Notes |
| ------ | ---- | ----- |
| `id` | UUID | Primary key, defaults to generated UUID. |
| `to` | VARCHAR(32) | Phone number / destination. |
| `content` | VARCHAR(160) | Message body, max 160 characters. |
| `sent` | BOOLEAN | Flag toggled after webhook acceptance. |
| `sent_at` | TIMESTAMPTZ | Timestamp stored when webhook indicates success. |
| `created_at` | TIMESTAMPTZ | Automatically set on insert.

## Scheduler Behavior
- Starts automatically during application boot.
- Every `SCHEDULER_INTERVAL`, fetches up to `SCHEDULER_FETCH_LIMIT` rows ordered by `created_at` where `sent=false`.
- Sends JSON payload `{ "to": "<phone>", "content": "<message>" }` to `WEBHOOK_URL` with `Content-Type: application/json` and `x-ins-auth-key` header when provided.
- Marks message as sent and records Redis metadata when webhook returns `{ "message": "Accepted", "messageId": "..." }`.
- Loop can be paused/resumed via control endpoints; scheduler is thread-safe and supports graceful shutdown.

## Project Structure
```
cmd/api           # main entrypoint
internal/config   # env loading
internal/db       # DB connection + migrations
internal/repository/postgres # SQL repositories
internal/service  # business logic + webhook/redis integration
internal/scheduler # custom ticker loop
internal/http     # router setup
internal/http/handler # REST handlers
api/swagger.yaml  # OpenAPI docs
migrations/       # SQL migrations (auto-run)
```
