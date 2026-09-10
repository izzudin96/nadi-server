package api

import (
	"context"
	"net/http"

	"github.com/izzudin96/nadi-server/internal/auth"
)

type ctxKey int

const userIDKey ctxKey = iota

// requireUser protects dashboard routes: it verifies the JWT in the httpOnly
// cookie and stashes the user id in the request context.
func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		uid, err := auth.ParseToken(c.Value, s.jwtSecret)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, uid)))
	})
}

func userIDFrom(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(userIDKey).(int64)
	return uid, ok
}
