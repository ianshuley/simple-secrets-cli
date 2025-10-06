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
	"testing"
	"time"

	secretsmodels "simple-secrets/pkg/secrets"
)

func TestCreateRotationMetadata(t *testing.T) {
	tests := []struct {
		name           string
		setupSecret    *secretsmodels.Secret
		key            string
		plaintext      []byte
		expectNewCount bool
	}{
		{
			name:           "new_secret_during_rotation",
			setupSecret:    nil,
			key:            "new-secret",
			plaintext:      []byte("secret-value"),
			expectNewCount: true,
		},
		{
			name: "existing_secret_rotation",
			setupSecret: &secretsmodels.Secret{
				Key:   "existing-secret",
				Value: []byte("encrypted"),
				Metadata: secretsmodels.SecretMetadata{
					Key:           "existing-secret",
					CreatedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					ModifiedAt:    time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
					LastRotatedAt: timePtr(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)),
					RotationCount: 2,
					Disabled:      false,
					Size:          12,
				},
			},
			key:            "existing-secret",
			plaintext:      []byte("secret-value"),
			expectNewCount: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup store
			store := &SecretsStore{
				secrets: make(map[string]secretsmodels.Secret),
			}

			// Add existing secret if provided
			if tt.setupSecret != nil {
				store.secrets[tt.key] = *tt.setupSecret
			}

			// Record time before operation
			beforeRotation := time.Now()

			// Execute
			result := store.createRotationMetadata(tt.key, tt.plaintext)

			// Record time after operation
			afterRotation := time.Now()

			// Validate basic fields
			if result.Key != tt.key {
				t.Errorf("Expected key %s, got %s", tt.key, result.Key)
			}
			if result.Size != len(tt.plaintext) {
				t.Errorf("Expected size %d, got %d", len(tt.plaintext), result.Size)
			}

			// Validate rotation tracking
			if result.LastRotatedAt == nil {
				t.Error("Expected LastRotatedAt to be set")
			} else {
				rotationTime := *result.LastRotatedAt
				if rotationTime.Before(beforeRotation) || rotationTime.After(afterRotation) {
					t.Errorf("Rotation time %v should be between %v and %v", rotationTime, beforeRotation, afterRotation)
				}
			}

			if tt.expectNewCount {
				// New secret case
				if result.RotationCount != 1 {
					t.Errorf("Expected rotation count 1 for new secret, got %d", result.RotationCount)
				}
				// For new secrets, CreatedAt and ModifiedAt should be recent
				if result.CreatedAt.Before(beforeRotation) || result.CreatedAt.After(afterRotation) {
					t.Errorf("CreatedAt %v should be between %v and %v", result.CreatedAt, beforeRotation, afterRotation)
				}
				if result.ModifiedAt.Before(beforeRotation) || result.ModifiedAt.After(afterRotation) {
					t.Errorf("ModifiedAt %v should be between %v and %v", result.ModifiedAt, beforeRotation, afterRotation)
				}
			} else {
				// Existing secret case
				originalMeta := tt.setupSecret.Metadata
				expectedCount := originalMeta.RotationCount + 1
				if result.RotationCount != expectedCount {
					t.Errorf("Expected rotation count %d, got %d", expectedCount, result.RotationCount)
				}
				// Original timestamps should be preserved
				if !result.CreatedAt.Equal(originalMeta.CreatedAt) {
					t.Errorf("CreatedAt should be preserved: expected %v, got %v", originalMeta.CreatedAt, result.CreatedAt)
				}
				if !result.ModifiedAt.Equal(originalMeta.ModifiedAt) {
					t.Errorf("ModifiedAt should be preserved: expected %v, got %v", originalMeta.ModifiedAt, result.ModifiedAt)
				}
				// Other metadata should be preserved
				if result.Disabled != originalMeta.Disabled {
					t.Errorf("Disabled flag should be preserved: expected %v, got %v", originalMeta.Disabled, result.Disabled)
				}
			}
		})
	}
}

func TestCreateRotationMetadata_PreservesDisabledState(t *testing.T) {
	store := &SecretsStore{
		secrets: map[string]secretsmodels.Secret{
			"disabled-secret": {
				Key:   "disabled-secret",
				Value: []byte("encrypted"),
				Metadata: secretsmodels.SecretMetadata{
					Key:           "disabled-secret",
					CreatedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					ModifiedAt:    time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
					LastRotatedAt: timePtr(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)),
					RotationCount: 1,
					Disabled:      true, // Secret is disabled
					Size:          12,
				},
			},
		},
	}

	result := store.createRotationMetadata("disabled-secret", []byte("new-value"))

	if !result.Disabled {
		t.Error("Expected disabled state to be preserved during rotation")
	}
	if result.RotationCount != 2 {
		t.Errorf("Expected rotation count to increment to 2, got %d", result.RotationCount)
	}
}

func TestCreateRotationMetadata_UpdatesSize(t *testing.T) {
	store := &SecretsStore{
		secrets: map[string]secretsmodels.Secret{
			"test-secret": {
				Key:   "test-secret",
				Value: []byte("old-encrypted"),
				Metadata: secretsmodels.SecretMetadata{
					Key:           "test-secret",
					CreatedAt:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					ModifiedAt:    time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
					LastRotatedAt: timePtr(time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)),
					RotationCount: 1,
					Disabled:      false,
					Size:          10, // Old size
				},
			},
		},
	}

	newPlaintext := []byte("much-longer-secret-value")
	result := store.createRotationMetadata("test-secret", newPlaintext)

	expectedSize := len(newPlaintext)
	if result.Size != expectedSize {
		t.Errorf("Expected size to be updated to %d, got %d", expectedSize, result.Size)
	}
}

// timePtr is a helper to create *time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
