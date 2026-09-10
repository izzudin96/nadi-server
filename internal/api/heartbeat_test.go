package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/izzudin96/nadi-server/internal/auth"
	"github.com/izzudin96/nadi-server/internal/config"
	"github.com/izzudin96/nadi-server/internal/store"
	"github.com/izzudin96/nadi-server/internal/testdb"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.New(t)
	st := store.New(pool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(NewRouter(st, logger, config.Config{JWTSecret: "test-secret"}))
	t.Cleanup(srv.Close)
	return srv, st, pool
}

func registerTestDevice(t *testing.T, st *store.Store, deviceID, key string) {
	t.Helper()
	hash, err := auth.HashSecret(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDevice(context.Background(), deviceID, hash); err != nil {
		t.Fatal(err)
	}
}

func sampleHeartbeat(deviceID string) heartbeat {
	hb := heartbeat{
		DeviceID:  deviceID,
		Timestamp: "2026-01-02T03:04:05Z",
		Metrics: []metric{
			{Name: "cpu.usage_percent", Value: 42.5, Unit: "%"},
			{Name: "memory.used_bytes", Value: 1024, Unit: "bytes"},
		},
	}
	hb.Meta.Hostname = "turn-01"
	hb.Meta.OS = "linux"
	hb.Meta.Arch = "amd64"
	hb.Meta.AgentVersion = "0.1.0"
	return hb
}

func postHeartbeat(t *testing.T, url, deviceID, key string, hb heartbeat) *http.Response {
	t.Helper()
	body, err := json.Marshal(hb)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url+"/api/heartbeat", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestHeartbeatAcceptedAndStored(t *testing.T) {
	srv, st, pool := newTestServer(t)
	registerTestDevice(t, st, "dev-1", "secret-key")

	resp := postHeartbeat(t, srv.URL, "dev-1", "secret-key", sampleHeartbeat("dev-1"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM metrics`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("metric count = %d, want 2", count)
	}

	// Device metadata + last_seen should be updated.
	d, err := st.GetDevice(context.Background(), "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Hostname != "turn-01" || d.OS != "linux" || d.LastSeenAt == nil {
		t.Fatalf("device not updated: %+v", d)
	}
}

func TestHeartbeatRejectsWrongKey(t *testing.T) {
	srv, st, _ := newTestServer(t)
	registerTestDevice(t, st, "dev-1", "correct-key")

	resp := postHeartbeat(t, srv.URL, "dev-1", "wrong-key", sampleHeartbeat("dev-1"))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestHeartbeatRejectsUnknownDevice(t *testing.T) {
	srv, _, _ := newTestServer(t)

	resp := postHeartbeat(t, srv.URL, "ghost", "some-key", sampleHeartbeat("ghost"))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestHeartbeatRejectsInvalidMetricName(t *testing.T) {
	srv, st, _ := newTestServer(t)
	registerTestDevice(t, st, "dev-1", "secret")

	hb := sampleHeartbeat("dev-1")
	hb.Metrics = []metric{{Name: "BAD_NAME", Value: 1}}
	resp := postHeartbeat(t, srv.URL, "dev-1", "secret", hb)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHeartbeatRejectsMissingKey(t *testing.T) {
	srv, st, _ := newTestServer(t)
	registerTestDevice(t, st, "dev-1", "secret")

	resp := postHeartbeat(t, srv.URL, "dev-1", "", sampleHeartbeat("dev-1"))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
