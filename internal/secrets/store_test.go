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
	"context"
	"testing"

	"simple-secrets/pkg/crypto"
)

// TestStoreBasicOperations tests the basic store operations
func TestStoreBasicOperations(t *testing.T) {
	// Create a master key for testing
	masterKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Create crypto service
	cryptoService := NewCryptoService(masterKey)

	// Create file repository (in-memory for testing would be better, but we'll use temp dir)
	tempDir := t.TempDir()
	repo := NewFileRepository(tempDir)

	// Create store
	store := NewStore(repo, cryptoService)

	ctx := context.Background()

	// Test Put and Get
	key := "test-key"
	value := "test-value"

	err = store.Put(ctx, key, value)
	if err != nil {
		t.Fatalf("Failed to put secret: %v", err)
	}

	retrievedValue, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if retrievedValue != value {
		t.Errorf("Expected value %q, got %q", value, retrievedValue)
	}

	// Test List
	metadata, err := store.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	if len(metadata) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(metadata))
	}

	// Test Disable and Enable
	err = store.Disable(ctx, key)
	if err != nil {
		t.Fatalf("Failed to disable secret: %v", err)
	}

	// Should not be able to get disabled secret
	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Expected error when getting disabled secret, got none")
	}

	// Should appear in disabled list
	disabledMetadata, err := store.ListDisabled(ctx)
	if err != nil {
		t.Fatalf("Failed to list disabled secrets: %v", err)
	}

	if len(disabledMetadata) != 1 {
		t.Errorf("Expected 1 disabled secret, got %d", len(disabledMetadata))
	}

	// Should not appear in enabled list
	enabledMetadata, err := store.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list enabled secrets: %v", err)
	}

	if len(enabledMetadata) != 0 {
		t.Errorf("Expected 0 enabled secrets, got %d", len(enabledMetadata))
	}

	// Re-enable
	err = store.Enable(ctx, key)
	if err != nil {
		t.Fatalf("Failed to enable secret: %v", err)
	}

	// Should be able to get again
	retrievedValue, err = store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get re-enabled secret: %v", err)
	}

	if retrievedValue != value {
		t.Errorf("Expected value %q after re-enable, got %q", value, retrievedValue)
	}

	// Test Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}

	// Should not be able to get deleted secret
	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Expected error when getting deleted secret, got none")
	}
}

// TestStoreGenerate tests the generate functionality
func TestStoreGenerate(t *testing.T) {
	// Create a master key for testing
	masterKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Create crypto service
	cryptoService := NewCryptoService(masterKey)

	// Create file repository
	tempDir := t.TempDir()
	repo := NewFileRepository(tempDir)

	// Create store
	store := NewStore(repo, cryptoService)

	ctx := context.Background()

	// Test Generate
	key := "generated-key"
	length := 16

	generatedValue, err := store.Generate(ctx, key, length)
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}

	if len(generatedValue) != length {
		t.Errorf("Expected generated value length %d, got %d", length, len(generatedValue))
	}

	// Should be able to retrieve the generated value
	retrievedValue, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get generated secret: %v", err)
	}

	if retrievedValue != generatedValue {
		t.Errorf("Expected generated value %q, got %q", generatedValue, retrievedValue)
	}
}

// TestStoreValidation tests input validation
func TestStoreValidation(t *testing.T) {
	// Create a master key for testing
	masterKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Create crypto service
	cryptoService := NewCryptoService(masterKey)

	// Create file repository
	tempDir := t.TempDir()
	repo := NewFileRepository(tempDir)

	// Create store
	store := NewStore(repo, cryptoService)

	ctx := context.Background()

	// Test empty key validation
	err = store.Put(ctx, "", "value")
	if err == nil {
		t.Error("Expected error for empty key, got none")
	}

	_, err = store.Get(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key, got none")
	}

	_, err = store.Generate(ctx, "", 16)
	if err == nil {
		t.Error("Expected error for empty key, got none")
	}

	err = store.Delete(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key, got none")
	}

	err = store.Enable(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key, got none")
	}

	err = store.Disable(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key, got none")
	}
}
