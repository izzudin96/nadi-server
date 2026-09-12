-- 0002_metrics_ts_index.sql — support retention purges that delete by timestamp.
CREATE INDEX IF NOT EXISTS idx_metrics_ts ON metrics (ts);