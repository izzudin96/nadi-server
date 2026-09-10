package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/izzudin96/nadi-server/internal/store"
)

// Server holds the dependencies the HTTP handlers need.
type Server struct {
	store  *store.Store
	logger *slog.Logger
}

// NewRouter wires the HTTP routes. The device-key auth lives inside the
// heartbeat handler (not middleware) because the device_id used to look up the
// key comes from the request body, not a header.
func NewRouter(st *store.Store, logger *slog.Logger) http.Handler {
	s := &Server{store: st, logger: logger}

	r := chi.NewRouter()
	r.Get("/healthz", s.handleHealthz)
	r.Post("/api/heartbeat", s.handleHeartbeat)
	return r
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		s.logger.Error("health check failed", "err", err)
		http.Error(w, "db unreachable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
