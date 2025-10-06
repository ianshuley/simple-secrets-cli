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
	"crypto/rand"
	"fmt"
)

// GenerateSecretValue creates a cryptographically secure random secret
// For generated secrets, length is bounded for security and resource management (8-1024 chars)
func GenerateSecretValue(length int) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("generated secret length must be at least 8 characters for security, got %d", length)
	}
	if length > 1024 {
		return "", fmt.Errorf("generated secret length cannot exceed 1024 characters, got %d (use manual storage for larger secrets like SSH keys)", length)
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
