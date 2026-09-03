// Package auth provides shared JWT claims and clinic-scope enforcement
// used across all Kinara OS microservices.
package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the canonical JWT payload for Kinara Governance OS.
// All services validate tokens using this struct.
// EntityType and TenantID are signed by the auth-service at login from the user's DB record;
// they are never supplied or overridden by the client.
type Claims struct {
	jwt.RegisteredClaims
	UserID     uuid.UUID  `json:"uid"`
	Username   string     `json:"username"`
	Role       string     `json:"role"`
	Scopes     []string   `json:"scopes"`
	EntityType string     `json:"entity_type"`        // "klinova" | "vha"
	TenantID   uuid.UUID  `json:"tenant_id"`           // UUID of the owning tenant row
	DeviceID   *uuid.UUID `json:"device_id,omitempty"`
	ClinicID   *uuid.UUID `json:"clinic_id,omitempty"`
	Scope      string     `json:"scope,omitempty"`     // "clinic:<uuid>" for device sessions
}

// IsDeviceSession returns true when the token was issued for a scoped device.
func (c *Claims) IsDeviceSession() bool {
	return c.DeviceID != nil && c.ClinicID != nil && strings.HasPrefix(c.Scope, "clinic:")
}

// IsAllowedRole returns true if the claim role matches any of the given roles.
func (c *Claims) IsAllowedRole(roles ...string) bool {
	for _, r := range roles {
		if c.Role == r {
			return true
		}
	}
	return false
}

// HasScope returns true if the given scope string appears in Scopes.
func (c *Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Validator validates JWT access tokens using an RSA public key.
type Validator struct {
	publicKey *rsa.PublicKey
}

// NewValidator loads the RSA public key from disk and returns a Validator.
func NewValidator(publicKeyPath string) (*Validator, error) {
	data, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		return nil, err
	}
	return &Validator{publicKey: pub}, nil
}

// Validate parses and verifies a JWT string, returning its Claims if valid.
func (v *Validator) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return v.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
