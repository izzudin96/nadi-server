// Package testdb provides a throwaway Postgres database for integration tests.
// It creates a fresh, uniquely named test database, runs migrations, and returns
// a pool. Tests skip if no Postgres is reachable (so plain CI without a DB still
// passes).
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/izzudin96/nadi-server/internal/db"
)

// New returns a pool connected to a freshly migrated test database. Each call
// uses a unique database name so parallel test packages cannot drop or recreate
// each other's database.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		adminURL = "postgres://nadi:nadi@localhost:5432/nadi?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Skipf("no Postgres available: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Skipf("no Postgres available: %v", err)
	}

	dbName := "nadi_test_" + randomSuffix()
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{dbName}.Sanitize())); err != nil {
		admin.Close()
		t.Fatalf("creating %s: %v", dbName, err)
	}
	admin.Close()

	testURL, err := databaseURL(adminURL, dbName)
	if err != nil {
		t.Fatalf("building test database URL: %v", err)
	}

	pool, err := db.Connect(context.Background(), testURL)
	if err != nil {
		t.Fatalf("connecting to %s: %v", dbName, err)
	}
	if err := db.Migrate(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("migrating %s: %v", dbName, err)
	}

	t.Cleanup(func() {
		pool.Close()

		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()

		admin, err := pgxpool.New(dropCtx, adminURL)
		if err != nil {
			return
		}
		defer admin.Close()
		_, _ = admin.Exec(dropCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", pgx.Identifier{dbName}.Sanitize()))
	})

	return pool
}

func randomSuffix() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func databaseURL(rawURL, dbName string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.Path = "/" + dbName
	return u.String(), nil
}
