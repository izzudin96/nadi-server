package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/izzudin96/nadi-server/internal/auth"
	"github.com/izzudin96/nadi-server/internal/store"
)

const cookieName = "nadi_token"

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// registrationOpen reports whether self-service signup is currently allowed.
// The "auto" policy opens registration only until the first user exists, which
// closes the door after the admin bootstraps their account.
func (s *Server) registrationOpen(ctx context.Context) (bool, error) {
	switch s.allowRegistration {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default: // "auto" and the zero value
		n, err := s.store.CountUsers(ctx)
		if err != nil {
			return false, err
		}
		return n == 0, nil
	}
}

func (s *Server) handleRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	open, err := s.registrationOpen(r.Context())
	if err != nil {
		s.logger.Error("checking registration status", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"open": open})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	open, err := s.registrationOpen(r.Context())
	if err != nil {
		s.logger.Error("checking registration status", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !open {
		http.Error(w, "registration is disabled", http.StatusForbidden)
		return
	}

	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	if c.Email == "" || len(c.Password) < 8 {
		http.Error(w, "email required and password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashSecret(c.Password)
	if err != nil {
		s.logger.Error("hashing password", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, err := s.store.CreateUser(r.Context(), c.Email, hash)
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		s.logger.Error("creating user", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.issueSession(w, id, c.Email)
	writeJSON(w, http.StatusCreated, map[string]string{"email": c.Email})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))

	u, err := s.store.GetUserByEmail(r.Context(), c.Email)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		s.logger.Error("user lookup failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !auth.CheckSecret(u.PasswordHash, c.Password) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	s.issueSession(w, u.ID, u.Email)
	writeJSON(w, http.StatusOK, map[string]string{"email": u.Email})
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"user_id": uid})
}

func (s *Server) issueSession(w http.ResponseWriter, userID int64, _ string) {
	token, err := auth.NewToken(userID, s.jwtSecret, s.tokenTTL)
	if err != nil {
		// Practically unreachable; log and let the caller's response go out
		// without a cookie.
		s.logger.Error("signing token", "err", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.tokenTTL.Seconds()),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
