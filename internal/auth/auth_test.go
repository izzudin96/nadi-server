package auth

import "testing"

func TestHashAndCheckSecret(t *testing.T) {
	hash, err := HashSecret("s3cret-key")
	if err != nil {
		t.Fatalf("HashSecret() error = %v", err)
	}
	if hash == "s3cret-key" {
		t.Fatal("hash should not equal the plaintext secret")
	}
	if !CheckSecret(hash, "s3cret-key") {
		t.Fatal("CheckSecret() = false for correct secret")
	}
	if CheckSecret(hash, "wrong") {
		t.Fatal("CheckSecret() = true for wrong secret")
	}
}

func TestHashSecretUnique(t *testing.T) {
	h1, _ := HashSecret("same")
	h2, _ := HashSecret("same")
	if h1 == h2 {
		t.Fatal("bcrypt should produce a unique salt per hash")
	}
}
