package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/izzudin96/nadi-server/internal/config"
	"github.com/izzudin96/nadi-server/internal/db"
)

var version = "dev"

func main() {
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

	mux := chi.NewRouter()
	mux.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			logger.Error("health check failed", "err", err)
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	logger.Info("server starting", "addr", cfg.Addr, "version", version)
	return http.ListenAndServe(cfg.Addr, mux)
}
