package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// rateLimiter is a small fixed-window counter keyed by client IP. It guards the
// unauthenticated auth endpoints against brute force. In-memory is fine for a
// single server instance; a shared store would be needed to scale out.
type rateLimiter struct {
	mu      sync.Mutex
	hits    map[string]*window
	limit   int
	period  time.Duration
	now     func() time.Time
	sweepAt int
}

type window struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, period time.Duration) *rateLimiter {
	return &rateLimiter{
		hits:    make(map[string]*window),
		limit:   limit,
		period:  period,
		now:     time.Now,
		sweepAt: 256,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	if len(rl.hits) >= rl.sweepAt {
		rl.sweep(now)
	}

	w, ok := rl.hits[key]
	if !ok || now.After(w.reset) {
		rl.hits[key] = &window{count: 1, reset: now.Add(rl.period)}
		return true
	}
	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}

// sweep drops expired windows. Called under the lock when the map grows.
func (rl *rateLimiter) sweep(now time.Time) {
	for k, w := range rl.hits {
		if now.After(w.reset) {
			delete(rl.hits, k)
		}
	}
}

// rateLimit rejects requests once a client IP exceeds the limit within the
// window. The window is generous enough that a human never notices.
func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authLimiter.allow(clientIP(r)) {
			http.Error(w, "too many requests, try again later", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP returns the peer address. It deliberately ignores X-Forwarded-For:
// that header is client-controlled unless a trusted proxy is rewriting it.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
