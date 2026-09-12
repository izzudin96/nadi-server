# Deploying nadi-server

The server is a single Docker image (Go binary with the dashboard embedded) plus
Postgres. It must sit behind a TLS-terminating reverse proxy because dashboard
auth uses `Secure` cookies.

## Prerequisites

- A Linux host with Docker + Docker Compose v2
- A domain pointing at the host (e.g. `monitor.example.com`)
- A TLS terminator (Caddy is the least effort; nginx/Traefik work too)

## 1. Configure secrets

```sh
cp deploy/.env.example deploy/.env
# Generate strong values and paste them in:
openssl rand -hex 32   # POSTGRES_PASSWORD
openssl rand -hex 32   # JWT_SECRET
```

`deploy/.env` is gitignored. `JWT_SECRET` must not be the dev default — the
server refuses to start with `NADI_SECURE_COOKIES=true` and the default secret.

## 2. Start the stack

```sh
docker compose -f deploy/docker-compose.yml up -d
docker compose -f deploy/docker-compose.yml ps      # server should become healthy
```

The server listens on `127.0.0.1:8080` (published on `0.0.0.0:8080` by default;
bind it to loopback in `docker-compose.yml` if the proxy runs on the same host).

## 3. Put TLS in front

Example Caddyfile:

```
monitor.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

Then `caddy reload`. Verify `https://monitor.example.com/healthz` returns `ok`.
Because `NADI_SECURE_COOKIES` defaults to `true`, the login cookie is only sent
over HTTPS — plain-HTTP access will silently fail to log in.

## 4. Create the admin user

Self-service registration is open only until the first user exists
(`NADI_ALLOW_REGISTRATION=auto`). Either:

- open `https://monitor.example.com` and register the first account, **or**
- bootstrap from the CLI (works even when registration is closed):

```sh
docker compose -f deploy/docker-compose.yml exec server \
  nadi-server -create-user admin@example.com
```

The CLI prints a generated password (or pass `-password '...'`). Save it — it is
not shown again. To add more users later, re-run `-create-user`, or set
`NADI_ALLOW_REGISTRATION=true` temporarily.

> The first dashboard user is a normal user; there is no separate admin role in
> v1. Keep registration closed (`NADI_ALLOW_REGISTRATION=false`) in production.

## 5. Generate a device API key

Every agent needs a `device_id` + `api_key`. Create one via the dashboard
(**Add device**) or the CLI:

```sh
docker compose -f deploy/docker-compose.yml exec server \
  nadi-server -create-device turn-01
```

It prints:

```
device_id: turn-01
api_key:   <48 hex chars>
```

Only the bcrypt hash is stored. Rotate a key from the dashboard row menu
(**Rotate key**) or by re-running `-create-device` with the same id. Deleting a
device from the dashboard also deletes its stored metrics.

## 6. Point the agent at the server

On the monitored host, edit `/etc/nadi-agent/agent.yaml`:

```yaml
device_id: "turn-01"
server_url: "https://monitor.example.com/api/heartbeat"
api_key: "<the key from step 5>"
interval_seconds: 15
```

`chmod 600` the file, then install the agent (see the agent's
`deploy/README.md`).

## 7. Verify

```sh
curl -fsS https://monitor.example.com/healthz        # -> ok
docker compose -f deploy/docker-compose.yml logs -f server
```

The device should appear as **Online** in the dashboard within one interval.

## Operations

- **Backups:** the data lives in the `pgdata` volume. `pg_dump` it on a schedule.
- **Retention:** the `metrics` table grows without bound; add a periodic
  `DELETE FROM metrics WHERE ts < now() - interval '90 days'` (or TimescaleDB).
- **Upgrades:** `docker compose -f deploy/docker-compose.yml pull && ... up -d`.
  Migrations run automatically on startup.
- **Registration policy:** `NADI_ALLOW_REGISTRATION` = `auto` (default),
  `true`, or `false`.
