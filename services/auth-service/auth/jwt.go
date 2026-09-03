package auth

import (
	"crypto/rsa"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	accessTokenTTL       = 15 * time.Minute
	deviceAccessTokenTTL = 5 * time.Minute
	refreshTokenTTL      = 7 * 24 * time.Hour
)

// Claims is the JWT payload used across Kinara Governance OS.
// EntityType and TenantID are set server-side at login from the user's DB record
// and cannot be supplied or overridden by the client.
type Claims struct {
	jwt.RegisteredClaims
	UserID     uuid.UUID  `json:"uid"`
	Username   string     `json:"username"`
	Role       string     `json:"role"`
	Scopes     []string   `json:"scopes"`
	EntityType string     `json:"entity_type"`          // "klinova" | "vha"
	TenantID   uuid.UUID  `json:"tenant_id"`             // UUID of the owning tenant row
	DeviceID   *uuid.UUID `json:"device_id,omitempty"`
	ClinicID   *uuid.UUID `json:"clinic_id,omitempty"`
	Scope      string     `json:"scope,omitempty"`       // "clinic:<uuid>" for device sessions
}

// Issuer signs access tokens with an RSA private key (RS256).
// Other services validate using the corresponding public key.
type Issuer struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
}

func NewIssuer(privateKeyPath, publicKeyPath string) (*Issuer, error) {
	privPEM, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		return nil, err
	}

	pubPEM, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		return nil, err
	}

	return &Issuer{
		privateKey: privKey,
		publicKey:  pubKey,
		issuer:     "kinara-governance-os/auth-service",
	}, nil
}

// IssueAccessToken signs a short-lived JWT (15 min) for API access.
// entityType and tenantID are looked up server-side and cannot be supplied by the client.
func (i *Issuer) IssueAccessToken(userID uuid.UUID, username, role, entityType string, tenantID uuid.UUID, scopes []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			ID:        uuid.New().String(),
		},
		UserID:     userID,
		Username:   username,
		Role:       role,
		Scopes:     scopes,
		EntityType: entityType,
		TenantID:   tenantID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(i.privateKey)
}

// Validate parses and verifies a JWT, returning its claims if valid.
func (i *Issuer) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return i.publicKey, nil
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

func (c *Claims) IsAllowedRole(roles ...string) bool {
	for _, r := range roles {
		if c.Role == r {
			return true
		}
	}
	return false
}

func (c *Claims) HasScope(scope string) bool {
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// IssueDeviceAccessToken signs a short-lived JWT (5 min) scoped to a single clinic.
// Device tokens have a "clinic:<clinic_id>" scope claim; all patient-data services
// must validate this via RequireClinicScope middleware before serving any PHI.
// entityType and tenantID must be looked up from the staff member's DB record before calling.
func (i *Issuer) IssueDeviceAccessToken(staffID, deviceID, clinicID uuid.UUID, role, entityType string, tenantID uuid.UUID) (string, error) {
	now := time.Now()
	clinicScope := "clinic:" + clinicID.String()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   staffID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(deviceAccessTokenTTL)),
			ID:        uuid.New().String(),
		},
		UserID:     staffID,
		Username:   "",
		Role:       role,
		Scopes:     []string{clinicScope},
		EntityType: entityType,
		TenantID:   tenantID,
		DeviceID:   &deviceID,
		ClinicID:   &clinicID,
		Scope:      clinicScope,
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(i.privateKey)
}

// AccessTokenTTLSeconds returns the access token lifetime in seconds.
func AccessTokenTTLSeconds() int {
	return int(accessTokenTTL.Seconds())
}

// DeviceAccessTokenTTLSeconds returns the device session token lifetime in seconds.
func DeviceAccessTokenTTLSeconds() int {
	return int(deviceAccessTokenTTL.Seconds())
}
