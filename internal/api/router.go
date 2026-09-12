package api

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/izzudin96/nadi-server/internal/config"
	"github.com/izzudin96/nadi-server/internal/store"
	"github.com/izzudin96/nadi-server/internal/webui"
)

// Server holds the dependencies the HTTP handlers need.
type Server struct {
	store             *store.Store
	logger            *slog.Logger
	jwtSecret         string
	tokenTTL          time.Duration
	cookieSecure      bool
	allowRegistration string
	authLimiter       *rateLimiter
}

// NewRouter wires the HTTP routes. The device-key auth lives inside the
// heartbeat handler (not middleware) because the device_id used to look up the
// key comes from the request body, not a header.
func NewRouter(st *store.Store, logger *slog.Logger, cfg config.Config) http.Handler {
	s := &Server{
		store:             st,
		logger:            logger,
		jwtSecret:         cfg.JWTSecret,
		tokenTTL:          24 * time.Hour,
		cookieSecure:      cfg.CookieSecure,
		allowRegistration: cfg.AllowRegistration,
		authLimiter:       newRateLimiter(10, time.Minute),
	}

	r := chi.NewRouter()
	r.Use(s.securityHeaders)
	r.Get("/healthz", s.handleHealthz)
	r.Post("/api/heartbeat", s.handleHeartbeat)

	r.Get("/api/auth/registration", s.handleRegistrationStatus)
	r.Group(func(r chi.Router) {
		r.Use(s.rateLimit)
		r.Post("/api/auth/register", s.handleRegister)
		r.Post("/api/auth/login", s.handleLogin)
	})
	r.Post("/api/auth/logout", s.handleLogout)

	r.Group(func(r chi.Router) {
		r.Use(s.requireUser)
		r.Get("/api/auth/me", s.handleMe)
		r.Get("/api/devices", s.handleListDevices)
		r.Post("/api/devices", s.handleCreateDevice)
		r.Post("/api/devices/{deviceID}/rotate", s.handleRotateDevice)
		r.Delete("/api/devices/{deviceID}", s.handleDeleteDevice)
		r.Get("/api/devices/{deviceID}/latest", s.handleLatest)
		r.Get("/api/devices/{deviceID}/metrics/{metricName}", s.handleMetricSeries)
	})

	// Serve the embedded dashboard if it has been built (make web). In dev the
	// Vite server proxies /api here, so the SPA is only served from this binary
	// in production.
	if fsys, err := webui.FS(); err == nil {
		if _, err := fs.Stat(fsys, "index.html"); err == nil {
			r.NotFound(spaHandler(fsys))
		}
	}

	return r
}

// securityHeaders sets conservative defaults for a same-origin SPA. Inline
// styles are allowed because the UI library sets style attributes at runtime.
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; font-src 'self'; connect-src 'self'; object-src 'none'; "+
				"frame-ancestors 'none'; base-uri 'self'")
		if s.cookieSecure {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
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
