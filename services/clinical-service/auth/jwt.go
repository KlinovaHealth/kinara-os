package auth

import (
	"crypto/rsa"
	"errors"
	"os"

	"github.com/golang-jwt/jwt/v5"
	pkgauth "github.com/klinova/kinara-os/pkg/auth"
)

// Claims is the JWT payload for Kinara OS tokens.
type Claims = pkgauth.Claims

// Validator verifies RS256-signed JWTs using a PEM public key.
type Validator struct {
	publicKey *rsa.PublicKey
}

func NewValidator(publicKeyPath string) (*Validator, error) {
	pem, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return nil, err
	}
	return &Validator{publicKey: key}, nil
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
