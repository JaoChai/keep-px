package crypto

import (
	"strings"
	"testing"
)

func testKey() []byte {
	return []byte("01234567890123456789012345678901") // 32 bytes
}

func TestRoundTrip(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey())
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	original := "EAABsbCS1iHgBO1234567890abcdef"
	encrypted, err := enc.Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !strings.HasPrefix(encrypted, "enc:") {
		t.Fatalf("encrypted should have enc: prefix, got %q", encrypted)
	}
	if encrypted == original {
		t.Fatal("encrypted should differ from original")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if decrypted != original {
		t.Fatalf("expected %q, got %q", original, decrypted)
	}
}

func TestEmptyString(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey())
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	encrypted, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	if encrypted != "" {
		t.Fatalf("expected empty, got %q", encrypted)
	}

	decrypted, err := enc.Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("expected empty, got %q", decrypted)
	}
}

func TestTamperedCiphertext(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey())
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	original := "EAABsbCS1iHgBO1234567890abcdef"
	encrypted, err := enc.Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with the encrypted string (keep enc: prefix)
	tampered := encrypted[:len(encrypted)-2] + "XX"
	_, err = enc.Decrypt(tampered)
	if err == nil {
		t.Fatal("Decrypt tampered enc: token should return error")
	}
}

func TestInvalidKeyLength(t *testing.T) {
	_, err := NewTokenEncryptor([]byte("tooshort"))
	if err == nil {
		t.Fatal("expected error for short key")
	}

	_, err = NewTokenEncryptor([]byte("this-key-is-way-too-long-for-aes-256-gcm-encryption"))
	if err == nil {
		t.Fatal("expected error for long key")
	}
}

func TestIsEncrypted(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey())
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	encrypted, err := enc.Encrypt("some-token")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if !IsEncrypted(encrypted) {
		t.Fatal("encrypted token should be detected as encrypted")
	}

	if IsEncrypted("EAABsbCS1iHgBO_plaintext_token") {
		t.Fatal("plaintext token should not be detected as encrypted")
	}

	if IsEncrypted("") {
		t.Fatal("empty string should not be detected as encrypted")
	}
}

func TestDecryptPlaintext(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey())
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	// A plaintext FB token should be returned as-is
	plaintext := "EAABsbCS1iHgBO_plaintext_token"
	decrypted, err := enc.Decrypt(plaintext)
	if err != nil {
		t.Fatalf("Decrypt plaintext: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestDecryptLongBase64Plaintext(t *testing.T) {
	enc, err := NewTokenEncryptor(testKey())
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	// FB access tokens are often long valid base64 strings (≥29 bytes decoded).
	// These should be returned as-is (not error), since they're plaintext.
	longToken := "EAABsbCS1iHgBOzBCxZAhJ1gZBYq5ZBrJ3xK7mz9ZC"
	decrypted, err := enc.Decrypt(longToken)
	if err != nil {
		t.Fatalf("Decrypt long base64 plaintext should not error: %v", err)
	}
	if decrypted != longToken {
		t.Fatalf("expected %q, got %q", longToken, decrypted)
	}
}
