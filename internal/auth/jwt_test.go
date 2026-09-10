package auth

import (
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	tok, err := NewToken(42, "secret", time.Hour)
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	uid, err := ParseToken(tok, "secret")
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if uid != 42 {
		t.Fatalf("uid = %d, want 42", uid)
	}
}

func TestTokenWrongSecret(t *testing.T) {
	tok, _ := NewToken(1, "secret-a", time.Hour)
	if _, err := ParseToken(tok, "secret-b"); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestTokenExpired(t *testing.T) {
	tok, _ := NewToken(1, "secret", -time.Minute)
	if _, err := ParseToken(tok, "secret"); err == nil {
		t.Fatal("expected error for expired token")
	}
}
