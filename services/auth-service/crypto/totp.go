package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	totpDigits   = 6
	totpPeriod   = 30 // seconds
	totpWindow   = 1  // ±1 period tolerance for clock drift
	totpIssuer   = "KinaraGovernanceOS"
)

// GenerateTOTPSecret returns a base32-encoded random TOTP secret (RFC 4226 §4).
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

// TOTPAuthURI returns the otpauth:// URI for QR code generation.
func TOTPAuthURI(secret, username string) string {
	return fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		totpIssuer, username, secret, totpIssuer, totpDigits, totpPeriod,
	)
}

// VerifyTOTP returns true if code is valid for the given base32 secret.
// Checks current window and ±1 adjacent windows for clock drift tolerance.
func VerifyTOTP(secret, code string) bool {
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(
		strings.ToUpper(strings.TrimSpace(secret)),
	)
	if err != nil {
		return false
	}
	counter := time.Now().Unix() / int64(totpPeriod)
	for delta := -int64(totpWindow); delta <= int64(totpWindow); delta++ {
		if computeHOTP(secretBytes, counter+delta) == code {
			return true
		}
	}
	return false
}

// computeHOTP implements RFC 4226 HOTP with SHA-1 and 6 digits.
func computeHOTP(secret []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	h := mac.Sum(nil)

	offset := h[len(h)-1] & 0x0f
	code := binary.BigEndian.Uint32(h[offset:offset+4]) & 0x7fffffff
	code = code % uint32(math.Pow10(totpDigits))
	return fmt.Sprintf("%0*d", totpDigits, code)
}
