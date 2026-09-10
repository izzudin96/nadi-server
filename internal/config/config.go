package config

import "os"

// Config holds server settings, loaded from environment variables (12-factor)
// with dev-friendly defaults that match docker-compose.yml.
type Config struct {
	Addr         string
	DatabaseURL  string
	JWTSecret    string
	CookieSecure bool
}

func Load() Config {
	return Config{
		Addr:         getenv("NADI_ADDR", ":8080"),
		DatabaseURL:  getenv("DATABASE_URL", "postgres://nadi:nadi@localhost:5432/nadi?sslmode=disable"),
		JWTSecret:    getenv("JWT_SECRET", "dev-secret-change-me"),
		CookieSecure: getenv("NADI_SECURE_COOKIES", "false") == "true",
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
