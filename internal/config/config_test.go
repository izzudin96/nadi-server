package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear relevant env vars so defaults apply.
	for _, k := range []string{"NADI_ADDR", "DATABASE_URL", "JWT_SECRET"} {
		os.Unsetenv(k)
	}

	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL should not be empty")
	}
	if cfg.JWTSecret == "" {
		t.Error("JWTSecret should not be empty")
	}
}

func TestLoadAllowRegistrationDefault(t *testing.T) {
	os.Unsetenv("NADI_ALLOW_REGISTRATION")
	if cfg := Load(); cfg.AllowRegistration != "auto" {
		t.Errorf("AllowRegistration = %q, want auto", cfg.AllowRegistration)
	}
}

func TestValidateRejectsDefaultSecretInProduction(t *testing.T) {
	cfg := Config{JWTSecret: defaultJWTSecret, CookieSecure: true, AllowRegistration: "auto"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for default JWT secret with secure cookies")
	}

	cfg.JWTSecret = "a-strong-secret"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestValidateRejectsBadRegistrationPolicy(t *testing.T) {
	cfg := Config{JWTSecret: "strong", AllowRegistration: "sometimes"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid NADI_ALLOW_REGISTRATION")
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("NADI_ADDR", ":9999")
	os.Setenv("DATABASE_URL", "postgres://example")
	os.Setenv("JWT_SECRET", "secret")
	defer func() {
		os.Unsetenv("NADI_ADDR")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg := Load()
	if cfg.Addr != ":9999" {
		t.Errorf("Addr = %q, want :9999", cfg.Addr)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "secret" {
		t.Errorf("JWTSecret = %q, want secret", cfg.JWTSecret)
	}
}
