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

	"github.com/izzudin96/nadi-server/internal/api"
	"github.com/izzudin96/nadi-server/internal/auth"
	"github.com/izzudin96/nadi-server/internal/config"
	"github.com/izzudin96/nadi-server/internal/db"
	"github.com/izzudin96/nadi-server/internal/store"
)

var version = "dev"

func main() {
	createDevice := flag.String("create-device", "", "register a device with this device_id, print its api key, then exit")
	apiKey := flag.String("api-key", "", "api key to use with -create-device (a random one is generated if empty)")
	flag.Parse()

	if *createDevice != "" {
		if err := registerDevice(*createDevice, *apiKey); err != nil {
			slog.Error("registering device", "err", err)
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(context.Background(), pool); err != nil {
		return err
	}

	mux := api.NewRouter(store.New(pool), logger, cfg)

	logger.Info("server starting", "addr", cfg.Addr, "version", version)
	return http.ListenAndServe(cfg.Addr, mux)
}

// registerDevice creates (or re-keys) a device and prints its API key. Used to
// bootstrap devices before the dashboard's device management exists.
func registerDevice(deviceID, apiKey string) error {
	cfg := config.Load()

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(context.Background(), pool); err != nil {
		return err
	}

	if apiKey == "" {
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil {
			return err
		}
		apiKey = hex.EncodeToString(buf)
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
