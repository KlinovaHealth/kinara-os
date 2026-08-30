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
// Device sessions populate DeviceID, ClinicID, and Scope; human sessions leave them nil/empty.
type Claims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID  `json:"uid"`
	Username string     `json:"username"`
	Role     string     `json:"role"`
	Scopes   []string   `json:"scopes"`
	DeviceID *uuid.UUID `json:"device_id,omitempty"`
	ClinicID *uuid.UUID `json:"clinic_id,omitempty"`
	Scope    string     `json:"scope,omitempty"` // e.g. "clinic:uuid"
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
func (i *Issuer) IssueAccessToken(userID uuid.UUID, username, role string, scopes []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			ID:        uuid.New().String(),
		},
		UserID:   userID,
		Username: username,
		Role:     role,
		Scopes:   scopes,
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
func (i *Issuer) IssueDeviceAccessToken(staffID, deviceID, clinicID uuid.UUID, role string) (string, error) {
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
		UserID:   staffID,
		Username: "",
		Role:     role,
		Scopes:   []string{clinicScope},
		DeviceID: &deviceID,
		ClinicID: &clinicID,
		Scope:    clinicScope,
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
