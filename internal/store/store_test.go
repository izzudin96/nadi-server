package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/izzudin96/nadi-server/internal/testdb"
)

func TestCreateAndGetDevice(t *testing.T) {
	pool := testdb.New(t)
	s := New(pool)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, "dev-1", "hash1"); err != nil {
		t.Fatalf("CreateDevice() error = %v", err)
	}

	d, err := s.GetDevice(ctx, "dev-1")
	if err != nil {
		t.Fatalf("GetDevice() error = %v", err)
	}
	if d.DeviceID != "dev-1" || d.APIKeyHash != "hash1" {
		t.Fatalf("device = %+v", d)
	}
}

func TestGetDeviceNotFound(t *testing.T) {
	pool := testdb.New(t)
	s := New(pool)

	if _, err := s.GetDevice(context.Background(), "missing"); err != ErrNotFound {
		t.Fatalf("GetDevice() error = %v, want ErrNotFound", err)
	}
}

func TestUpdateDeviceSeen(t *testing.T) {
	pool := testdb.New(t)
	s := New(pool)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, "dev-1", "hash"); err != nil {
		t.Fatal(err)
	}

	ts := time.Now().UTC().Truncate(time.Microsecond)
	if err := s.UpdateDeviceSeen(ctx, "dev-1", "turn-01", "linux", "amd64", "0.1.0", ts); err != nil {
		t.Fatalf("UpdateDeviceSeen() error = %v", err)
	}

	d, err := s.GetDevice(ctx, "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Hostname != "turn-01" || d.OS != "linux" || d.Arch != "amd64" || d.AgentVersion != "0.1.0" {
		t.Fatalf("device metadata = %+v", d)
	}
	if d.LastSeenAt == nil || !d.LastSeenAt.Equal(ts) {
		t.Fatalf("last_seen_at = %v, want %v", d.LastSeenAt, ts)
	}
}

func TestInsertMetrics(t *testing.T) {
	pool := testdb.New(t)
	s := New(pool)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, "dev-1", "hash"); err != nil {
		t.Fatal(err)
	}

	ts := time.Now().UTC().Truncate(time.Microsecond)
	metrics := []Metric{
		{Name: "cpu.usage_percent", Value: 42.5, Unit: "%"},
		{Name: "memory.used_bytes", Value: 1024, Unit: "bytes"},
	}
	if err := s.InsertMetrics(ctx, "dev-1", metrics, ts); err != nil {
		t.Fatalf("InsertMetrics() error = %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM metrics`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("metric count = %d, want 2", count)
	}
}

func TestInsertMetricsRejectsUnknownDevice(t *testing.T) {
	pool := testdb.New(t)
	s := New(pool)
	ctx := context.Background()

	// device_id has a FK to devices; inserting for an unknown device must fail.
	err := s.InsertMetrics(ctx, "nope", []Metric{{Name: "cpu.usage_percent", Value: 1}}, time.Now())
	if err == nil {
		t.Fatal("expected FK violation for unknown device")
	}
}

func TestPurgeMetrics(t *testing.T) {
	pool := testdb.New(t)
	s := New(pool)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, "dev-1", "hash"); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	old := []Metric{{Name: "cpu.usage_percent", Value: 1, Unit: "%"}}
	recent := []Metric{{Name: "cpu.usage_percent", Value: 2, Unit: "%"}}
	if err := s.InsertMetrics(ctx, "dev-1", old, now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertMetrics(ctx, "dev-1", recent, now); err != nil {
		t.Fatal(err)
	}

	deleted, err := s.PurgeMetrics(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("PurgeMetrics() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM metrics`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("metric count after purge = %d, want 1", count)
	}
}

func TestMetricSeriesBucketed(t *testing.T) {
	pool := testdb.New(t)
	s := New(pool)
	ctx := context.Background()

	if err := s.CreateDevice(ctx, "dev-1", "hash"); err != nil {
		t.Fatal(err)
	}

	base := time.Now().UTC().Truncate(time.Hour)
	for i, v := range []float64{1, 2, 3} {
		ts := base.Add(time.Duration(i) * 10 * time.Minute)
		if err := s.InsertMetrics(ctx, "dev-1", []Metric{{Name: "cpu.usage_percent", Value: v}}, ts); err != nil {
			t.Fatal(err)
		}
	}

	points, err := s.MetricSeriesBucketed(ctx, "dev-1", "cpu.usage_percent", base, base.Add(time.Hour), 10*time.Minute)
	if err != nil {
		t.Fatalf("MetricSeriesBucketed() error = %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("points = %d, want 3", len(points))
	}
	for i, v := range []float64{1, 2, 3} {
		if points[i].Value != v {
			t.Fatalf("points[%d].Value = %v, want %v", i, points[i].Value, v)
		}
	}

	// Two samples in the same bucket average together.
	if err := s.InsertMetrics(ctx, "dev-1", []Metric{{Name: "mem.used_bytes", Value: 100}}, base.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertMetrics(ctx, "dev-1", []Metric{{Name: "mem.used_bytes", Value: 200}}, base.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	avg, err := s.MetricSeriesBucketed(ctx, "dev-1", "mem.used_bytes", base, base.Add(time.Hour), 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(avg) != 1 || avg[0].Value != 150 {
		t.Fatalf("avg = %+v, want one point with value 150", avg)
	}
}

func ExampleNew() {
	fmt.Println("store.New wraps a *pgxpool.Pool")
	// Output: store.New wraps a *pgxpool.Pool
}
