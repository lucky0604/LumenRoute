# LumenRoute

Lightweight internal model control plane for existing vLLM / SGLang services. Provides an OpenAI-compatible unified proxy, a React admin console, route configuration, API key management, request logs, and Prometheus metrics — without taking over model deployment or GPU scheduling.

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, SQLite, WAL mode |
| Frontend | React, Vite, TypeScript, Ant Design |
| Proxy | OpenAI-compatible `/v1/models`, `/v1/chat/completions` |
| Metrics | Prometheus `/metrics` |
| Deployment | Docker Compose, single binary + systemd |

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 18+
- (Optional) Docker

### Backend

```bash
# Run all tests
go test ./...

# Build and run
go build -o lumenroute ./cmd/server
./lumenroute
```

The server starts on `:8080` by default. On first startup, an admin account is created:

- If `LUMENROUTE_ADMIN_PASSWORD` is set, that password is used.
- Otherwise, a random password is written to `data/bootstrap-admin-password`.

### Frontend

```bash
cd web
npm install
npm run dev      # dev server with HMR
npm run build    # production build
```

## Configuration

All settings via environment variables:

| Variable | Default | Description |
|---|---|---|
| `LUMENROUTE_SERVER_PORT` | `8080` | HTTP listen port |
| `LUMENROUTE_DB_DSN` | `file:data/lumenroute.db?...` | SQLite DSN |
| `LUMENROUTE_ADMIN_USER` | `admin` | Admin username |
| `LUMENROUTE_ADMIN_PASSWORD` | (generated) | Admin password |
| `LUMENROUTE_PROXY_AUTH_MODE` | `required` | `required` / `optional` / `disabled` |
| `LUMENROUTE_SESSION_SECRET` | (generated) | Session signing key |
| `LUMENROUTE_API_KEY_PREFIX` | `llmcp_` | Proxy API key prefix |
| `LUMENROUTE_METRICS_PATH` | `/metrics` | Prometheus metrics endpoint |
| `LUMENROUTE_HEALTH_CHECK_INTERVAL_SECONDS` | `30` | Provider health check interval |
| `LUMENROUTE_REQUEST_LOG_RETENTION_DAYS` | `7` | Log retention period |

## Docker

```bash
docker compose -f docker/docker-compose.yml up -d
```

The service uses `network_mode: host` by default for internal network deployments.

## API Reference

### Admin API (requires session cookie)

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/auth/login` | Admin login |
| `POST` | `/api/auth/logout` | Admin logout |
| `GET` | `/api/providers` | List providers |
| `POST` | `/api/providers` | Create provider |
| `GET` | `/api/providers/:id` | Get provider |
| `PUT` | `/api/providers/:id` | Update provider |
| `DELETE` | `/api/providers/:id` | Delete provider |
| `POST` | `/api/providers/:id/check` | Health check provider |
| `GET` | `/api/routes` | List routes |
| `POST` | `/api/routes` | Create route |
| `GET` | `/api/routes/:id` | Get route |
| `PUT` | `/api/routes/:id` | Update route |
| `DELETE` | `/api/routes/:id` | Delete route |
| `GET` | `/api/routes/:id/targets` | List route targets |
| `POST` | `/api/routes/:id/targets` | Create target |
| `PUT` | `/api/route-targets/:id` | Update target |
| `DELETE` | `/api/route-targets/:id` | Delete target |
| `POST` | `/api/route-targets/:id/test` | Test target |
| `GET` | `/api/api-keys` | List API keys |
| `POST` | `/api/api-keys` | Create API key |
| `DELETE` | `/api/api-keys/:id` | Delete API key |
| `POST` | `/api/api-keys/:id/disable` | Disable API key |
| `POST` | `/api/api-keys/:id/enable` | Enable API key |
| `GET` | `/api/request-logs` | List request logs |
| `GET` | `/api/request-logs/:id` | Get request log detail |

### Proxy API (OpenAI-compatible)

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/models` | List available models |
| `POST` | `/v1/chat/completions` | Chat completions (stream & non-stream) |

Proxy auth is configured via `LUMENROUTE_PROXY_AUTH_MODE`:

- `required`: `Authorization: Bearer <api_key>` required
- `optional`: Valid key enforced if provided; allows missing key
- `disabled`: No API key validation

## Architecture

```
cmd/server/         Entrypoint
internal/
  api/              HTTP handlers
  apikey/           API key management (SHA-256 hashed, one-time reveal)
  auth/             Admin bootstrap, password hashing, session management
  config/           Environment configuration
  db/               SQLite connectivity, versioned migrations, indexes
  logs/             Request log write, query, filter
  metrics/          Prometheus metrics registry
  models/           Domain models
  provider/         Provider CRUD, health state
  proxy/            OpenAI proxy (list models, chat completions, SSE streaming)
  route/            Route/target CRUD, weighted target selection
  scheduler/        Provider health checker, log retention cleanup
web/
  src/pages/        Login, Providers, Routes, API Keys, Request Logs, Health
  src/components/   AdminLayout
docker/             Dockerfile, docker-compose.yml
tests/              Contract tests
```

## Domain Concepts

| Concept | Description |
|---|---|
| **Provider** | An existing OpenAI-compatible upstream service (vLLM, SGLang) |
| **Route** | A public model name exposed to clients |
| **Route Target** | A specific upstream model on a provider backing a route |
| **API Key** | Proxy credential for `/v1/*` endpoints (SHA-256 hashed) |
| **Request Log** | Metadata per proxy request (no prompt/response stored) |

## Development

```bash
# Backend
go test ./...              # run all tests
go build -o lumenroute ./cmd/server  # build
go vet ./...               # static analysis

# Frontend
cd web
npm run dev                # dev server (http://localhost:5173)
npm run build              # production build
npm run preview            # preview production build
```

## License

Internal use.
