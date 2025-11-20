package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

// deriveKey returns a 32-byte key derived from ENCRYPTION_KEY env var.
func deriveKey() ([]byte, error) {
	secret := os.Getenv("ENCRYPTION_KEY")
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("ENCRYPTION_KEY not set")
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:], nil
}

// EncryptString encrypts plaintext with AES-GCM and returns base64(nonce|ciphertext).
func EncryptString(plain string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, []byte(plain), nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// DecryptString reverses EncryptString.
func DecryptString(b64 string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// MaskID returns masked version keeping last 4 characters (or fewer when short).
func MaskID(id string) string {
	if id == "" {
		return ""
	}
	n := len(id)
	if n <= 4 {
		return strings.Repeat("*", n)
	}
	return strings.Repeat("*", n-4) + id[n-4:]
}
