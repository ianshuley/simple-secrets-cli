/*
Copyright © 2025 Ian Shuley

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Crypto constants to eliminate magic numbers
const (
	AES256KeySize = 32 // AES-256 key size in bytes
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

// GenerateSecretValue creates a cryptographically secure random secret
func GenerateSecretValue(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive, got %d", length)
	}

	// Character set: A-Z, a-z, 0-9, and URL-safe symbols
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+"

	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i, b := range randomBytes {
		result[i] = charset[int(b)%len(charset)]
	}

	return string(result), nil
}
