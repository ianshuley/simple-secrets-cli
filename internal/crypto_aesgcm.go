package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// encrypt: AES-256-GCM → base64(ciphertext with nonce prefix)
func encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
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

	ct := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// decrypt: base64 → (nonce||ciphertext) → AES-256-GCM
func decrypt(key []byte, encrypted string) ([]byte, error) {
	ct, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	n := gcm.NonceSize()
	if len(ct) < n {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, body := ct[:n], ct[n:]
	return gcm.Open(nil, nonce, body, nil)
}
