package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/izzudin96/nadi-server/internal/api"
	"github.com/izzudin96/nadi-server/internal/auth"
	"github.com/izzudin96/nadi-server/internal/config"
	"github.com/izzudin96/nadi-server/internal/db"
	"github.com/izzudin96/nadi-server/internal/store"
)

var version = "dev"

func main() {
	createDevice := flag.String("create-device", "", "register a device with this device_id, print its api key, then exit")
	createUser := flag.String("create-user", "", "create a dashboard user with this email, print its password, then exit")
	apiKey := flag.String("api-key", "", "api key to use with -create-device (a random one is generated if empty)")
	password := flag.String("password", "", "password to use with -create-user (a random one is generated if empty)")
	flag.Parse()

	if *createDevice != "" {
		if err := registerDevice(*createDevice, *apiKey); err != nil {
			slog.Error("registering device", "err", err)
			os.Exit(1)
		}
		return
	}
	if *createUser != "" {
		if err := createDashboardUser(*createUser, *password); err != nil {
			slog.Error("creating user", "err", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("server exiting", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(context.Background(), pool); err != nil {
		return err
	}

	st := store.New(pool)
	mux := api.NewRouter(st, logger, cfg)

	go runRetention(context.Background(), st, cfg.RetentionDays, logger)

	logger.Info("server starting", "addr", cfg.Addr, "version", version)
	return http.ListenAndServe(cfg.Addr, mux)
}

// retentionInterval is how often the retention job scans for expired samples.
const retentionInterval = time.Hour

// runRetention periodically deletes metric samples older than the configured
// retention window so the metrics table cannot grow without bound. It purges
// once at startup so a freshly configured window takes effect immediately.
func runRetention(ctx context.Context, st *store.Store, days int, logger *slog.Logger) {
	if days <= 0 {
		logger.Info("retention disabled", "retention_days", days)
		return
	}

	purge := func() {
		cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		deleted, err := st.PurgeMetrics(ctx, cutoff)
		if err != nil {
			logger.Error("retention purge failed", "err", err)
			return
		}
		if deleted > 0 {
			logger.Info("retention purge complete", "deleted", deleted, "older_than", cutoff)
		}
	}

	purge()
	ticker := time.NewTicker(retentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purge()
		}
	}
}

// registerDevice creates (or re-keys) a device and prints its API key. Used to
// bootstrap devices before the dashboard's device management exists.
func registerDevice(deviceID, apiKey string) error {
	pool, err := connectAndMigrate()
	if err != nil {
		return err
	}
	defer pool.Close()

	if apiKey == "" {
		if apiKey, err = auth.GenerateAPIKey(); err != nil {
			return err
		}
	}

	hash, err := auth.HashSecret(apiKey)
	if err != nil {
		return err
	}
	if err := store.New(pool).CreateDevice(context.Background(), deviceID, hash); err != nil {
		return err
	}

	fmt.Printf("device_id: %s\napi_key:   %s\n", deviceID, apiKey)
	return nil
}

// createDashboardUser creates a dashboard user, bypassing the registration
// policy. Use it to bootstrap the admin account or to add users after
// self-service registration has closed.
func createDashboardUser(email, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return fmt.Errorf("email is required")
	}

	pool, err := connectAndMigrate()
	if err != nil {
		return err
	}
	defer pool.Close()

	generated := password == ""
	if generated {
		buf := make([]byte, 12)
		if _, err := rand.Read(buf); err != nil {
			return err
		}
		password = hex.EncodeToString(buf)
	}
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := auth.HashSecret(password)
	if err != nil {
		return err
	}
	if _, err := store.New(pool).CreateUser(context.Background(), email, hash); err != nil {
		return err
	}

	fmt.Printf("email:    %s\npassword: %s\n", email, password)
	if generated {
		fmt.Println("(generated password — save it now; it will not be shown again)")
	}
	return nil
}

func connectAndMigrate() (*pgxpool.Pool, error) {
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(context.Background(), pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
