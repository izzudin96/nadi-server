# nadi-server

Central server for Nadi: receives agent heartbeats, stores metrics in Postgres,
and serves the Vue dashboard (embedded in the same binary).

## Stack

- **Go** + `net/http` via `chi` router
- **Postgres** (`pgx/v5`) — one database everywhere, run via docker-compose
- **Auth** — JWT in `httpOnly` cookies for the dashboard (`golang-jwt/v5`),
  bcrypt-hashed device API keys for the heartbeat API
- **Frontend** — Vue 3 + Vite + Pinia + Tailwind + ECharts, embedded at build time

## Development

Prerequisites: Go 1.22+, Node 20+, GNU make, Docker.

```sh
docker compose up -d postgres   # start Postgres
make run                        # run the API server on :8080

# in another terminal, run the dashboard with hot reload (proxies /api to :8080):
cd web && npm install && npm run dev
```

### Bootstrap a device

The agent authenticates with a per-device API key. Register a device (prints its key):

```sh
go run ./cmd/nadi-server -create-device my-device
```

Then put that `device_id` + `api_key` into the agent's `agent.yaml`.

### Register a user

Open the dashboard (http://localhost:5173 in dev) and register/login. The
dashboard is read-only and protected by the JWT cookie.

## Configuration (env vars)

| Var | Default | Purpose |
|---|---|---|
| `NADI_ADDR` | `:8080` | listen address |
| `DATABASE_URL` | `postgres://nadi:nadi@localhost:5432/nadi?sslmode=disable` | Postgres DSN |
| `JWT_SECRET` | `dev-secret-change-me` | HMAC key for dashboard JWTs (set in prod) |
| `NADI_SECURE_COOKIES` | `false` | set `true` behind HTTPS |

## Build (production)

```sh
make build-full    # npm build + embed dashboard + go build
make build-linux   # cross-compiled amd64 binary (no CGO)
```

## Test

```sh
make test   # unit + integration tests; integration tests use a throwaway
            # nadi_test database (skipped if no Postgres is reachable)
```

## Project layout

```
cmd/nadi-server/   entrypoint (server + -create-device bootstrap)
internal/api/      HTTP handlers, auth middleware, SPA serving
internal/auth/     bcrypt + JWT helpers
internal/config/   env-var config
internal/db/       pgx pool + embedded SQL migrations
internal/store/    data access (devices, metrics, users)
internal/webui/    embedded dashboard files
web/               Vue 3 dashboard source (Vite)
```

See `PLAN.md` (parent repo) for the full roadmap.
