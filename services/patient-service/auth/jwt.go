// Package auth handles JWT validation and claim extraction.
// Tokens are issued by the Kinara OS Auth Service and contain
// the caller's user_id, role, and granted service scopes.
package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrMissingToken  = errors.New("auth: authorization token is missing")
	ErrInvalidToken  = errors.New("auth: token is invalid or expired")
	ErrInsufficientScope = errors.New("auth: token does not grant required scope")
)

// Claims is the payload embedded in every Kinara OS JWT.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	Scopes []string  `json:"scopes"`
	jwt.RegisteredClaims
}

// Validator holds the RSA public key used to verify JWT signatures.
type Validator struct {
	publicKey *rsa.PublicKey
}

// NewValidator loads the RSA public key from a PEM file and returns a Validator.
func NewValidator(publicKeyPath string) (*Validator, error) {
	pem, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to read public key file %q: %w", publicKeyPath, err)
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to parse RSA public key: %w", err)
	}

	return &Validator{publicKey: key}, nil
}

// Validate parses and verifies a raw JWT string.
// Returns the embedded Claims on success.
func (v *Validator) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// HasScope returns true if the claims include the specified scope.
func (c *Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// IsAllowedRole returns true if the caller's role is in the allowed set.
func (c *Claims) IsAllowedRole(roles ...string) bool {
	for _, r := range roles {
		if c.Role == r {
			return true
		}
	}
	return false
}
