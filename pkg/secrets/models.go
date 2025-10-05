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

import "time"

// Secret represents a stored secret with its encrypted value and metadata
type Secret struct {
	Key      string         `json:"key"`
	Value    []byte         `json:"value"` // Encrypted value
	Metadata SecretMetadata `json:"metadata"`
}

// SecretMetadata contains information about a secret's lifecycle and status
type SecretMetadata struct {
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Disabled   bool      `json:"disabled"`
	Size       int       `json:"size"` // Size of plaintext value for display
}

// IsDisabled returns true if the secret is currently disabled
func (s *Secret) IsDisabled() bool {
	return s.Metadata.Disabled
}

// IsEnabled returns true if the secret is currently enabled
func (s *Secret) IsEnabled() bool {
	return !s.Metadata.Disabled
}

// Enable marks the secret as enabled
func (s *Secret) Enable() {
	s.Metadata.Disabled = false
	s.Metadata.ModifiedAt = time.Now()
}

// Disable marks the secret as disabled
func (s *Secret) Disable() {
	s.Metadata.Disabled = true
	s.Metadata.ModifiedAt = time.Now()
}

// UpdateValue updates the secret's encrypted value and metadata
func (s *Secret) UpdateValue(encryptedValue []byte, plaintextSize int) {
	s.Value = encryptedValue
	s.Metadata.ModifiedAt = time.Now()
	s.Metadata.Size = plaintextSize
}

// NewSecret creates a new secret with the given key and encrypted value
func NewSecret(key string, encryptedValue []byte, plaintextSize int) *Secret {
	now := time.Now()
	return &Secret{
		Key:   key,
		Value: encryptedValue,
		Metadata: SecretMetadata{
			CreatedAt:  now,
			ModifiedAt: now,
			Disabled:   false,
			Size:       plaintextSize,
		},
	}
}
