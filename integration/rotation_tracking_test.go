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
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	secretsmodels "simple-secrets/pkg/secrets"
)

func TestRotationTrackingIntegration(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	t.Run("rotation_tracking_through_cli", func(t *testing.T) {
		// Store initial secrets with different creation times
		putSecret(t, h, "old-secret", "original-value")

		// Simulate some time passing and user modification
		time.Sleep(10 * time.Millisecond)
		putSecret(t, h, "old-secret", "modified-value")

		// Store another secret
		putSecret(t, h, "another-secret", "another-value")

		// Read initial metadata from file
		initialMetadata := readSecretsMetadata(t, h.GetTempDir())

		// Verify initial state
		oldSecretMeta := initialMetadata["old-secret"]
		if oldSecretMeta.RotationCount != 0 {
			t.Errorf("Expected initial rotation count 0, got %d", oldSecretMeta.RotationCount)
		}
		if oldSecretMeta.LastRotatedAt != nil {
			t.Error("Expected LastRotatedAt to be nil initially")
		}

		// Perform master key rotation
		beforeRotation := time.Now()
		output, err := h.RunCommand("rotate", "master-key", "--yes")
		afterRotation := time.Now()

		if err != nil {
			t.Fatalf("Master key rotation failed: %v\n%s", err, output)
		}

		// Read metadata after rotation
		postRotationMetadata := readSecretsMetadata(t, h.GetTempDir())

		// Verify rotation tracking for existing secrets
		for secretKey, meta := range postRotationMetadata {
			t.Run("secret_"+secretKey, func(t *testing.T) {
				// Rotation count should be incremented
				if meta.RotationCount != 1 {
					t.Errorf("Expected rotation count 1 after first rotation, got %d", meta.RotationCount)
				}

				// LastRotatedAt should be set and recent
				if meta.LastRotatedAt == nil {
					t.Error("Expected LastRotatedAt to be set after rotation")
				} else {
					rotationTime := *meta.LastRotatedAt
					if rotationTime.Before(beforeRotation) || rotationTime.After(afterRotation) {
						t.Errorf("Rotation time %v should be between %v and %v", rotationTime, beforeRotation, afterRotation)
					}
				}

				// CreatedAt should be preserved, ModifiedAt should be updated
				originalMeta := initialMetadata[secretKey]
				if !meta.CreatedAt.Equal(originalMeta.CreatedAt) {
					t.Errorf("CreatedAt should be preserved: expected %v, got %v", originalMeta.CreatedAt, meta.CreatedAt)
				}
				if !meta.ModifiedAt.After(originalMeta.ModifiedAt) {
					t.Errorf("ModifiedAt should be updated during rotation: original %v, post-rotation %v", originalMeta.ModifiedAt, meta.ModifiedAt)
				}
			})
		}

		// Verify secrets are still accessible
		getValue(t, h, "old-secret", "modified-value")
		getValue(t, h, "another-secret", "another-value")

		// Perform second rotation
		time.Sleep(10 * time.Millisecond)
		beforeSecondRotation := time.Now()
		output2, err2 := h.RunCommand("rotate", "master-key", "--yes")
		afterSecondRotation := time.Now()

		if err2 != nil {
			t.Fatalf("Second master key rotation failed: %v\n%s", err2, output2)
		}

		// Verify second rotation tracking
		secondRotationMetadata := readSecretsMetadata(t, h.GetTempDir())

		for secretKey, meta := range secondRotationMetadata {
			t.Run("second_rotation_"+secretKey, func(t *testing.T) {
				// Rotation count should be incremented again
				if meta.RotationCount != 2 {
					t.Errorf("Expected rotation count 2 after second rotation, got %d", meta.RotationCount)
				}

				// LastRotatedAt should be updated
				if meta.LastRotatedAt == nil {
					t.Error("Expected LastRotatedAt to be set after second rotation")
				} else {
					rotationTime := *meta.LastRotatedAt
					if rotationTime.Before(beforeSecondRotation) || rotationTime.After(afterSecondRotation) {
						t.Errorf("Second rotation time %v should be between %v and %v", rotationTime, beforeSecondRotation, afterSecondRotation)
					}
				}

				// CreatedAt should still be preserved, ModifiedAt should be updated
				originalMeta := initialMetadata[secretKey]
				if !meta.CreatedAt.Equal(originalMeta.CreatedAt) {
					t.Errorf("CreatedAt should still be preserved: expected %v, got %v", originalMeta.CreatedAt, meta.CreatedAt)
				}
				if !meta.ModifiedAt.After(beforeSecondRotation) {
					t.Errorf("ModifiedAt should be updated during second rotation: got %v", meta.ModifiedAt)
				}
			})
		}
	})
}

func TestRotationTrackingWithDisabledSecrets(t *testing.T) {
	h := NewTestHelper(t)
	defer h.Cleanup()

	t.Run("disabled_secrets_rotation_tracking", func(t *testing.T) {
		// Store and disable a secret
		putSecret(t, h, "disabled-secret", "secret-value")
		disableSecret(t, h, "disabled-secret")

		// Read initial metadata
		initialMetadata := readSecretsMetadata(t, h.GetTempDir())
		disabledMeta := initialMetadata["disabled-secret"]

		if !disabledMeta.Disabled {
			t.Fatal("Secret should be disabled before rotation test")
		}

		// Perform rotation
		beforeRotation := time.Now()
		output, err := h.RunCommand("rotate", "master-key", "--yes")
		afterRotation := time.Now()

		if err != nil {
			t.Fatalf("Master key rotation failed: %v\n%s", err, output)
		}

		// Verify disabled secret rotation tracking
		postRotationMetadata := readSecretsMetadata(t, h.GetTempDir())
		rotatedDisabledMeta := postRotationMetadata["disabled-secret"]

		// Should still be disabled
		if !rotatedDisabledMeta.Disabled {
			t.Error("Secret should remain disabled after rotation")
		}

		// Should have rotation tracking
		if rotatedDisabledMeta.RotationCount != 1 {
			t.Errorf("Expected rotation count 1, got %d", rotatedDisabledMeta.RotationCount)
		}

		if rotatedDisabledMeta.LastRotatedAt == nil {
			t.Error("Expected LastRotatedAt to be set for disabled secret")
		} else {
			rotationTime := *rotatedDisabledMeta.LastRotatedAt
			if rotationTime.Before(beforeRotation) || rotationTime.After(afterRotation) {
				t.Errorf("Rotation time %v should be between %v and %v", rotationTime, beforeRotation, afterRotation)
			}
		}

		// CreatedAt should be preserved, ModifiedAt should be updated
		if !rotatedDisabledMeta.CreatedAt.Equal(disabledMeta.CreatedAt) {
			t.Errorf("CreatedAt should be preserved for disabled secret")
		}
		if !rotatedDisabledMeta.ModifiedAt.After(beforeRotation) {
			t.Errorf("ModifiedAt should be updated during rotation for disabled secret")
		}
	})
}

// Helper functions for the integration tests

// putSecret stores a secret using the CLI
func putSecret(t *testing.T, h *TestHelper, key, value string) {
	t.Helper()
	output, err := h.RunCommand("put", key, value)
	if err != nil {
		t.Fatalf("Failed to put secret %s: %v\n%s", key, err, output)
	}
}

// getValue retrieves and verifies a secret value using the CLI
func getValue(t *testing.T, h *TestHelper, key, expectedValue string) {
	t.Helper()
	output, err := h.RunCommand("get", key)
	if err != nil {
		t.Fatalf("Failed to get secret %s: %v\n%s", key, err, output)
	}

	actualValue := strings.TrimSpace(string(output))
	if actualValue != expectedValue {
		t.Errorf("Expected value %q for key %s, got %q", expectedValue, key, actualValue)
	}
}

// disableSecret disables a secret using the CLI
func disableSecret(t *testing.T, h *TestHelper, key string) {
	t.Helper()
	output, err := h.RunCommand("disable", "secret", key)
	if err != nil {
		t.Fatalf("Failed to disable secret %s: %v\n%s", key, err, output)
	}
}

// readSecretsMetadata reads and parses the secrets.json file to extract metadata
func readSecretsMetadata(t *testing.T, tempDir string) map[string]secretsmodels.SecretMetadata {
	t.Helper()
	secretsPath := filepath.Join(tempDir, ".simple-secrets", "secrets.json")

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("Failed to read secrets file: %v", err)
	}

	var fileFormat struct {
		Secrets map[string]secretsmodels.Secret `json:"secrets"`
		Version string                          `json:"version"`
	}

	if err := json.Unmarshal(data, &fileFormat); err != nil {
		t.Fatalf("Failed to parse secrets file: %v", err)
	}

	metadata := make(map[string]secretsmodels.SecretMetadata)
	for key, secret := range fileFormat.Secrets {
		metadata[key] = secret.Metadata
	}

	return metadata
}
