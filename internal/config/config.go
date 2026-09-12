package config

import (
	"errors"
	"fmt"
	"os"
)

const defaultJWTSecret = "dev-secret-change-me"

// Config holds server settings, loaded from environment variables (12-factor)
// with dev-friendly defaults that match docker-compose.yml.
type Config struct {
	Addr         string
	DatabaseURL  string
	JWTSecret    string
	CookieSecure bool
	// AllowRegistration controls self-service signup:
	//   "auto"  — open only while there are zero users (first-run bootstrap)
	//   "true"  — always open
	//   "false" — always closed (manage users with -create-user)
	AllowRegistration string
}

func Load() Config {
	return Config{
		Addr:              getenv("NADI_ADDR", ":8080"),
		DatabaseURL:       getenv("DATABASE_URL", "postgres://nadi:nadi@localhost:5432/nadi?sslmode=disable"),
		JWTSecret:         getenv("JWT_SECRET", defaultJWTSecret),
		CookieSecure:      getenv("NADI_SECURE_COOKIES", "false") == "true",
		AllowRegistration: getenv("NADI_ALLOW_REGISTRATION", "auto"),
	}
}

// Validate rejects unsafe production configuration. A secure-cookie server is
// treated as production, so it must not run with the baked-in dev secret.
func (c Config) Validate() error {
	if c.CookieSecure && c.JWTSecret == defaultJWTSecret {
		return errors.New("JWT_SECRET must be set to a strong random value when NADI_SECURE_COOKIES=true")
	}
	switch c.AllowRegistration {
	case "auto", "true", "false":
	default:
		return fmt.Errorf("NADI_ALLOW_REGISTRATION must be one of auto|true|false, got %q", c.AllowRegistration)
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
