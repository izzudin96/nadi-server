-- 0001_init.sql — initial schema: users, devices, metrics.

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS devices (
    id            BIGSERIAL PRIMARY KEY,
    device_id     TEXT NOT NULL UNIQUE,
    api_key_hash  TEXT NOT NULL,
    hostname      TEXT,
    os            TEXT,
    arch          TEXT,
    agent_version TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at  TIMESTAMPTZ
);

-- Time-series: one row per metric per device per heartbeat. The composite index
-- supports the dashboard's per-metric range queries. TimescaleDB can replace
-- this later if volume demands (PLAN §2.1).
CREATE TABLE IF NOT EXISTS metrics (
    id          BIGSERIAL PRIMARY KEY,
    device_id   TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    unit        TEXT,
    ts          TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_metrics_device_name_ts
    ON metrics (device_id, metric_name, ts DESC);
