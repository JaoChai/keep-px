package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// encPrefix marks tokens encrypted by this package.
const encPrefix = "enc:"

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
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *TokenEncryptor) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	// New format: "enc:" prefix — strict decryption
	if strings.HasPrefix(encoded, encPrefix) {
		return e.decryptBase64(encoded[len(encPrefix):])
	}

	// Legacy format: try base64 + GCM, fallback to plaintext
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded, nil // not valid base64: plaintext token
	}
	if len(data) < e.gcm.NonceSize()+16 {
		return encoded, nil // too short: plaintext token
	}
	nonce, ciphertext := data[:e.gcm.NonceSize()], data[e.gcm.NonceSize():]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// GCM failed: plaintext FB token that happens to be valid base64
		return encoded, nil
	}
	return string(plaintext), nil
}

// decryptBase64 decodes base64 and decrypts AES-GCM. Errors are strict.
func (e *TokenEncryptor) decryptBase64(b64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	if len(data) < e.gcm.NonceSize()+16 {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:e.gcm.NonceSize()], data[e.gcm.NonceSize():]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}
	return string(plaintext), nil
}

// IsEncrypted checks if a token has the "enc:" prefix (encrypted by this package).
func IsEncrypted(token string) bool {
	return strings.HasPrefix(token, encPrefix)
}
