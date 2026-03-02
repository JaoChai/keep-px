package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type TokenEncryptor struct {
	gcm cipher.AEAD
}

func NewTokenEncryptor(key []byte) (*TokenEncryptor, error) {
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &TokenEncryptor{gcm: gcm}, nil
}

func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *TokenEncryptor) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded, nil // not encrypted, return as-is
	}
	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return encoded, nil // not encrypted, return as-is
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return encoded, nil // decrypt failed, return as-is (graceful fallback)
	}
	return string(plaintext), nil
}

// IsEncrypted checks if a token looks like it's encrypted (base64 encoded, min length).
func IsEncrypted(token string) bool {
	if token == "" {
		return false
	}
	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	// AES-GCM: nonce (12) + ciphertext (>0) + tag (16) = min 29 bytes
	return len(data) >= 29
}
