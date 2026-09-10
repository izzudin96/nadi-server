package store

import (
	"context"
	"time"
)

// Metric is a single time-series value.
type Metric struct {
	Name  string
	Value float64
	Unit  string
}

// InsertMetrics writes a heartbeat's worth of metrics, all sharing the same
// timestamp. Batched in one transaction; pgx CopyFrom would be faster at high
// volume but this is clearer and fine for v1.
func (s *Store) InsertMetrics(ctx context.Context, deviceID string, metrics []Metric, ts time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, m := range metrics {
		if _, err := tx.Exec(ctx,
			`INSERT INTO metrics (device_id, metric_name, value, unit, ts) VALUES ($1, $2, $3, $4, $5)`,
			deviceID, m.Name, m.Value, m.Unit, ts); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
