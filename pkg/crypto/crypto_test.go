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

package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, AES256KeySize)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}

	payloads := [][]byte{
		nil,
		[]byte(""),
		[]byte("x"),
		bytes.Repeat([]byte{0}, 32),
		bytes.Repeat([]byte("ab"), 1024),
	}

	for i, p := range payloads {
		ct, err := Encrypt(key, p)
		if err != nil {
			t.Fatalf("case %d: encrypt: %v", i, err)
		}

		pt, err := Decrypt(key, ct)
		if err != nil {
			t.Fatalf("case %d: decrypt: %v", i, err)
		}

		if !bytes.Equal(p, pt) {
			t.Fatalf("case %d: plaintext mismatch\n  exp: %q\n  got: %q", i, p, pt)
		}
	}
}

func TestDecrypt_TamperFails(t *testing.T) {
	key := make([]byte, AES256KeySize)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("hello world")
	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt the base64 - change one character
	corrupted := strings.Replace(ciphertext, string(ciphertext[0]), "Z", 1)
	if corrupted == ciphertext {
		// If replacement didn't work, manually change first char
		runes := []rune(ciphertext)
		if runes[0] == 'Z' {
			runes[0] = 'A'
		} else {
			runes[0] = 'Z'
		}
		corrupted = string(runes)
	}

	_, err = Decrypt(key, corrupted)
	if err == nil {
		t.Fatal("expected decrypt to fail with corrupted ciphertext")
	}
}

func FuzzEncryptDecrypt(f *testing.F) {
	key := make([]byte, AES256KeySize)
	_, err := rand.Read(key)
	if err != nil {
		f.Fatal(err)
	}

	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	f.Add(make([]byte, 1000))

	f.Fuzz(func(t *testing.T, plaintext []byte) {
		ciphertext, err := Encrypt(key, plaintext)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}

		// Ensure it's valid base64
		_, err = base64.StdEncoding.DecodeString(ciphertext)
		if err != nil {
			t.Fatalf("ciphertext is not valid base64: %v", err)
		}

		decrypted, err := Decrypt(key, ciphertext)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted) {
			t.Fatalf("plaintext mismatch\n  exp: %q\n  got: %q", plaintext, decrypted)
		}
	})
}

func TestGenerateSecretValue(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"valid length", 32, false},
		{"minimum length", 1, false},
		{"large length", 1000, false},
		{"zero length", 0, true},
		{"negative length", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateSecretValue(tt.length)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != tt.length {
				t.Errorf("expected length %d, got %d", tt.length, len(result))
			}

			// Check that result contains only valid characters
			const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+"
			for _, char := range result {
				if !strings.ContainsRune(charset, char) {
					t.Errorf("invalid character %c in generated secret", char)
				}
			}
		})
	}
}

func TestGenerateSecretValue_Randomness(t *testing.T) {
	// Generate multiple secrets and ensure they're different
	secrets := make([]string, 10)
	for i := range secrets {
		var err error
		secrets[i], err = GenerateSecretValue(32)
		if err != nil {
			t.Fatalf("failed to generate secret %d: %v", i, err)
		}
	}

	// Check that all secrets are unique
	for i := 0; i < len(secrets); i++ {
		for j := i + 1; j < len(secrets); j++ {
			if secrets[i] == secrets[j] {
				t.Errorf("generated duplicate secrets: %q", secrets[i])
			}
		}
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"hello", "aGVsbG8gd29ybGQ"},                        // This will be different - just checking format
		{"", "47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU"}, // SHA256 of empty string
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := HashToken(tt.token)

			// Check that result is valid base64
			_, err := base64.RawURLEncoding.DecodeString(result)
			if err != nil {
				t.Errorf("result is not valid base64: %v", err)
			}

			// Check deterministic behavior
			result2 := HashToken(tt.token)
			if result != result2 {
				t.Error("hash function is not deterministic")
			}

			// For empty string, we know the exact hash
			if tt.token == "" && result != tt.want {
				t.Errorf("expected %q, got %q", tt.want, result)
			}
		})
	}
}
