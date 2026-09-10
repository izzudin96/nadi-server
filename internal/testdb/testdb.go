// Package testdb provides a throwaway Postgres database for integration tests.
// It creates a fresh nadi_test database, runs migrations, and returns a pool.
// Tests skip if no Postgres is reachable (so plain CI without a DB still passes).
package testdb

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/izzudin96/nadi-server/internal/db"
)

// New returns a pool connected to a freshly migrated nadi_test database.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		adminURL = "postgres://nadi:nadi@localhost:5432/nadi?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Skipf("no Postgres available: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Skipf("no Postgres available: %v", err)
	}

	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS nadi_test WITH (FORCE)`); err != nil {
		t.Fatalf("dropping nadi_test: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE nadi_test`); err != nil {
		t.Fatalf("creating nadi_test: %v", err)
	}
	admin.Close()

	testURL := strings.Replace(adminURL, "/nadi?", "/nadi_test?", 1)
	pool, err := db.Connect(context.Background(), testURL)
	if err != nil {
		t.Fatalf("connecting to nadi_test: %v", err)
	}
	if err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrating nadi_test: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
