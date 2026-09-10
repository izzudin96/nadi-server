package auth

import "golang.org/x/crypto/bcrypt"

// HashSecret returns a bcrypt hash of a secret (a device API key or a user
// password). Secrets are never stored in plaintext.
func HashSecret(secret string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckSecret reports whether secret matches a stored hash.
func CheckSecret(hash, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}
