// Package crypto provides AES-256-GCM encryption for PHI fields.
// Every encrypt call uses a fresh random nonce; the nonce is prepended
// to the ciphertext and the whole payload is base64-encoded for storage.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrInvalidKeySize  = errors.New("crypto: key must be exactly 32 bytes for AES-256")
	ErrCiphertextShort = errors.New("crypto: ciphertext is too short to contain a valid nonce")
	ErrDecryptFailed   = errors.New("crypto: decryption failed — data may be corrupted or key is wrong")
)

// Encryptor holds the 32-byte AES-256 master key.
type Encryptor struct {
	key []byte
}

// New returns an Encryptor. key must be exactly 32 bytes.
func New(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}
	k := make([]byte, 32)
	copy(k, key)
	return &Encryptor{key: k}, nil
}

// EncryptString encrypts plaintext and returns a base64-encoded string
// of the form: nonce || ciphertext.
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	return e.Encrypt([]byte(plaintext))
}

// Encrypt encrypts raw bytes and returns a base64-encoded string.
func (e *Encryptor) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal appends ciphertext+tag to nonce
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptString decrypts a base64-encoded ciphertext produced by Encrypt.
func (e *Encryptor) DecryptString(encoded string) (string, error) {
	plain, err := e.Decrypt(encoded)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// Decrypt decrypts a base64-encoded ciphertext and returns the raw bytes.
func (e *Encryptor) Decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return plain, nil
}

// EncryptOptional encrypts s if non-empty, returns empty string otherwise.
// Used for nullable PHI fields like email and address.
func (e *Encryptor) EncryptOptional(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	return e.EncryptString(s)
}

// DecryptOptional decrypts s if non-empty, returns empty string otherwise.
func (e *Encryptor) DecryptOptional(s *string) (string, error) {
	if s == nil || *s == "" {
		return "", nil
	}
	return e.DecryptString(*s)
}

// KeyFromHex is a convenience function for tests. Never use hardcoded keys in production.
func KeyFromEnv(raw string) ([]byte, error) {
	b := []byte(raw)
	if len(b) != 32 {
		return nil, ErrInvalidKeySize
	}
	return b, nil
}
