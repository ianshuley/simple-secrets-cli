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

import "simple-secrets/pkg/crypto"

// CryptoService wraps crypto operations with a master key for the secrets domain
type CryptoService struct {
	masterKey []byte
}

// NewCryptoService creates a new crypto service with the given master key
func NewCryptoService(masterKey []byte) *CryptoService {
	return &CryptoService{
		masterKey: masterKey,
	}
}

// Encrypt encrypts plaintext using the master key
func (s *CryptoService) Encrypt(plaintext []byte) ([]byte, error) {
	encryptedB64, err := crypto.Encrypt(s.masterKey, plaintext)
	if err != nil {
		return nil, err
	}
	return []byte(encryptedB64), nil
}

// Decrypt decrypts ciphertext using the master key
func (s *CryptoService) Decrypt(ciphertext []byte) ([]byte, error) {
	return crypto.Decrypt(s.masterKey, string(ciphertext))
}

// GenerateSecretValue generates a new secret value
func (s *CryptoService) GenerateSecretValue(length int) (string, error) {
	return crypto.GenerateSecretValue(length)
}
