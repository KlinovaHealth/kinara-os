package auth

import (
	"crypto/rsa"
	"errors"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID     uuid.UUID  `json:"uid"`
	Username   string     `json:"username"`
	Role       string     `json:"role"`
	Scopes     []string   `json:"scopes"`
	EntityType string     `json:"entity_type"` // "klinova" | "vha"
	TenantID   uuid.UUID  `json:"tenant_id"`
	DeviceID   *uuid.UUID `json:"device_id,omitempty"`
	ClinicID   *uuid.UUID `json:"clinic_id,omitempty"`
	Scope      string     `json:"scope,omitempty"` // "clinic:<uuid>" for device sessions
}

func (c *Claims) IsDeviceSession() bool {
	return c.DeviceID != nil && c.ClinicID != nil && strings.HasPrefix(c.Scope, "clinic:")
}

func (c *Claims) IsAllowedRole(allowed ...string) bool {
	for _, r := range allowed {
		if c.Role == r {
			return true
		}
	}
	return false
}

type Validator struct {
	publicKey *rsa.PublicKey
}

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
