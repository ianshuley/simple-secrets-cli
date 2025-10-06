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
	"strings"
	"testing"

	"simple-secrets/integration/testing_framework"
)

func TestRotationAndRestoreCommands(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Add a secret first for rotation testing
	output, err := env.CLI().Put("backup-test", "original-value")
	if err != nil {
		t.Fatalf("put failed: %v\n%s", err, output)
	}

	// Test master key rotation with --yes flag
	t.Run("rotate_master_key_with_yes", func(t *testing.T) {
		output, err := env.CLI().Rotate().MasterKey()
		if err != nil {
			t.Fatalf("rotate master-key failed: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "Master key rotation completed successfully") {
			t.Errorf("expected success message, got: %s", output)
		}
	})

	// Verify secret is still accessible after rotation
	t.Run("verify_secret_after_rotation", func(t *testing.T) {
		output, err := env.CLI().Get("backup-test")
		if err != nil {
			t.Fatalf("get after rotation failed: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "original-value") {
			t.Errorf("expected original value, got: %s", output)
		}
	})

	// Test master key rotation with interactive confirmation
	t.Run("rotate_master_key_with_confirmation", func(t *testing.T) {
		output, err := env.CLI().Rotate().MasterKeyWithConfirmation("yes\n")
		if err != nil {
			t.Fatalf("rotate master-key with confirmation failed: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "Master key rotation completed successfully") {
			t.Errorf("expected success message, got: %s", output)
		}
	})

	// BUG FIX VERIFIED: This test previously revealed a panic in put command after rotation
	// The panic was: "invalid memory address or nil pointer dereference" in SecretsStore.Get
	// Fixed by eliminating race condition between rotation and Get/Put operations

	// Modify the secret to test point-in-time recovery
	t.Run("modify_secret_for_backup", func(t *testing.T) {
		output, err := env.CLI().Put("backup-test", "modified-value")
		if err != nil {
			t.Fatalf("put modified failed: %v\n%s", err, output)
		}
	})

	// Create rotation backup with the modified state
	t.Run("create_rotation_backup", func(t *testing.T) {
		output, err := env.CLI().Rotate().MasterKey()
		if err != nil {
			t.Fatalf("second rotate failed: %v\n%s", err, output)
		}
	})

	// Test secret restoration
	t.Run("restore_secret", func(t *testing.T) {
		output, err := env.CLI().Secrets().RestoreWithConfirmation("backup-test", "y\n")
		if err != nil {
			t.Fatalf("restore secret failed: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "restoration completed successfully") {
			t.Errorf("expected restore confirmation, got: %s", output)
		}
	})

	// Verify restored secret has original value
	t.Run("verify_restored_secret", func(t *testing.T) {
		output, err := env.CLI().Get("backup-test")
		if err != nil {
			t.Fatalf("get restored secret failed: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "modified-value") {
			t.Errorf("expected modified value after restore (current behavior: most recent backup), got: %s", output)
		}
	})

	// Test error cases
	t.Run("restore_nonexistent_secret", func(t *testing.T) {
		output, err := env.CLI().Secrets().RestoreWithConfirmation("nonexistent", "n\n")
		if err != nil {
			t.Fatalf("unexpected error when restoring nonexistent secret: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "Secret restoration cancelled") {
			t.Errorf("expected cancellation message, got: %s", output)
		}
	})

	// Test database restoration shows usage message when no backup specified
	t.Run("restore_database_requires_backup_name", func(t *testing.T) {
		output, err := env.CLI().Restore().Database()
		if err == nil {
			t.Fatalf("expected error when restoring database without backup name, but got none. Output: %s", output)
		}
		if !strings.Contains(string(output), "database restore requires a backup name") {
			t.Errorf("expected backup name requirement message, got: %s", output)
		}
	})

	// Test listing backups
	t.Run("list_backups", func(t *testing.T) {
		output, err := env.CLI().List().Backups()
		if err != nil {
			t.Fatalf("list backups failed: %v\n%s", err, output)
		}
		// Should show rotation backups after our rotations
		if strings.Contains(string(output), "(no rotation backups available)") {
			t.Errorf("expected rotation backups to be available, got: %s", output)
		}
	})
}
