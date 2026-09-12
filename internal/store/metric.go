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

// PurgeMetrics deletes metric samples older than olderThan and returns how many
// rows were removed. Backing the retention policy, the server calls this on an
// interval (see cmd/nadi-server).
func (s *Store) PurgeMetrics(ctx context.Context, olderThan time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM metrics WHERE ts < $1`, olderThan)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
