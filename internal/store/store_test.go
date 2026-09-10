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

func ExampleNew() {
	fmt.Println("store.New wraps a *pgxpool.Pool")
	// Output: store.New wraps a *pgxpool.Pool
}
