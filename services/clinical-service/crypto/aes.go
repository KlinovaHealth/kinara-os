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
	ErrInvalidKeySize  = errors.New("encryption key must be exactly 32 bytes")
	ErrCiphertextShort = errors.New("ciphertext too short")
	ErrDecryptFailed   = errors.New("decryption failed: authentication tag mismatch")
)

// Encryptor wraps AES-256-GCM. Each call to Encrypt generates a fresh random nonce,
// so two encryptions of the same plaintext produce different ciphertexts.
type Encryptor struct {
	key []byte
}

func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}
	k := make([]byte, 32)
	copy(k, key)
	return &Encryptor{key: k}, nil
}

// Encrypt returns base64(nonce || ciphertext).
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
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt.
func (e *Encryptor) Decrypt(encoded string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
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
	if len(data) < gcm.NonceSize() {
		return nil, ErrCiphertextShort
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return plaintext, nil
}

func (e *Encryptor) EncryptString(s string) (string, error) {
	return e.Encrypt([]byte(s))
}

func (e *Encryptor) DecryptString(encoded string) (string, error) {
	b, err := e.Decrypt(encoded)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// EncryptOptional encrypts s if non-empty; returns nil otherwise.
func (e *Encryptor) EncryptOptional(s string) (*string, error) {
	if s == "" {
		return nil, nil
	}
	enc, err := e.EncryptString(s)
	if err != nil {
		return nil, err
	}
	return &enc, nil
}

// DecryptOptional decrypts a nullable encrypted column.
func (e *Encryptor) DecryptOptional(enc *string) (string, error) {
	if enc == nil {
		return "", nil
	}
	return e.DecryptString(*enc)
}
