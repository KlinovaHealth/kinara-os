package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var ErrInvalidPassword = errors.New("password does not match")

// HashPassword returns a bcrypt hash. The plaintext password is never stored.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword returns nil if password matches hash, ErrInvalidPassword otherwise.
func VerifyPassword(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// GenerateAPIKey returns a random key in the format "kinara_<base64url>" and its SHA-256 hash.
// The key is returned once; only the hash is stored in the database.
func GenerateAPIKey() (key, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	key = "kinara_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(key))
	hash = hex.EncodeToString(sum[:])
	return key, hash, nil
}

// HashAPIKey returns the SHA-256 hex hash of an API key (used for lookup).
func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// GenerateRefreshToken returns a cryptographically random 48-byte hex string
// and its SHA-256 hash. The hex string is returned to the client; only the hash
// is stored in the sessions table.
func GenerateRefreshToken() (token, hash string, err error) {
	raw := make([]byte, 48)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	token = fmt.Sprintf("%x", raw)
	sum := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(sum[:])
	return token, hash, nil
}

// HashRefreshToken hashes a refresh token for DB lookup.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
