package store

import (
	"context"
	"time"
)

// DeviceSummary is a device row for the dashboard list view.
type DeviceSummary struct {
	DeviceID     string     `json:"device_id"`
	Hostname     string     `json:"hostname"`
	OS           string     `json:"os"`
	Arch         string     `json:"arch"`
	AgentVersion string     `json:"agent_version"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
}

// ListDevices returns all devices ordered by id.
func (s *Store) ListDevices(ctx context.Context) ([]DeviceSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT device_id, COALESCE(hostname, ''), COALESCE(os, ''), COALESCE(arch, ''),
		       COALESCE(agent_version, ''), last_seen_at
		FROM devices ORDER BY device_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DeviceSummary
	for rows.Next() {
		var d DeviceSummary
		if err := rows.Scan(&d.DeviceID, &d.Hostname, &d.OS, &d.Arch, &d.AgentVersion, &d.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// LatestMetric is the most recent value of a metric for a device.
type LatestMetric struct {
	Name  string    `json:"name"`
	Value float64   `json:"value"`
	Unit  string    `json:"unit"`
	Ts    time.Time `json:"ts"`
}

// LatestMetrics returns the newest value of every metric seen for a device.
func (s *Store) LatestMetrics(ctx context.Context, deviceID string) ([]LatestMetric, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (metric_name) metric_name, value, COALESCE(unit, ''), ts
		FROM metrics
		WHERE device_id = $1
		ORDER BY metric_name, ts DESC`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LatestMetric
	for rows.Next() {
		var m LatestMetric
		if err := rows.Scan(&m.Name, &m.Value, &m.Unit, &m.Ts); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Point is a single (timestamp, value) sample in a metric series.
type Point struct {
	Ts    time.Time `json:"ts"`
	Value float64   `json:"value"`
}

// MetricSeries returns a metric's samples in [from, to], oldest first, capped
// at limit rows.
func (s *Store) MetricSeries(ctx context.Context, deviceID, metricName string, from, to time.Time, limit int) ([]Point, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ts, value
		FROM metrics
		WHERE device_id = $1 AND metric_name = $2 AND ts >= $3 AND ts <= $4
		ORDER BY ts ASC
		LIMIT $5`, deviceID, metricName, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Ts, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MetricSeriesBucketed returns a metric's samples in [from, to] averaged into
// stride-sized buckets, oldest first. It is used for long time windows so a
// chart gets a bounded number of points regardless of range; buckets are
// aligned to a fixed epoch so boundaries are stable across requests. Empty
// buckets produce no rows, so when stride is smaller than the heartbeat
// interval the result matches the raw series.
func (s *Store) MetricSeriesBucketed(ctx context.Context, deviceID, metricName string, from, to time.Time, stride time.Duration) ([]Point, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT date_bin($1, ts, TIMESTAMPTZ '2000-01-01') AS bucket, avg(value) AS value
		FROM metrics
		WHERE device_id = $2 AND metric_name = $3 AND ts >= $4 AND ts <= $5
		GROUP BY bucket
		ORDER BY bucket ASC`,
		stride, deviceID, metricName, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Ts, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
