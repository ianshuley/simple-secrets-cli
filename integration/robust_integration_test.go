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
)

// TestConsolidatedListCommandsRefactored demonstrates a robust approach to integration testing
func TestConsolidatedListCommandsRefactored(t *testing.T) {
	// Create isolated test helper
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains string
	}{
		{
			name:     "list keys",
			args:     []string{"list", "keys"},
			wantErr:  false,
			contains: "", // Empty list is fine for new store
		},
		{
			name:     "list backups",
			args:     []string{"list", "backups"},
			wantErr:  false,
			contains: "(no rotation backups available)",
		},
		{
			name:     "list users",
			args:     []string{"list", "users"},
			wantErr:  false,
			contains: "admin",
		},
		{
			name:     "list invalid",
			args:     []string{"list", "invalid"},
			wantErr:  true,
			contains: "unknown list type",
		},
		{
			name:     "list no args",
			args:     []string{"list"},
			wantErr:  true,
			contains: "accepts 1 arg(s), received 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := helper.RunCommand(tc.args...)

			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %v, but got none. Output: %s", tc.args, output)
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %v: %v\nOutput: %s", tc.args, err, output)
			}

			if tc.contains != "" && !strings.Contains(string(output), tc.contains) {
				t.Fatalf("expected output to contain %q, got: %s", tc.contains, output)
			}
		})
	}
}

// TestConsolidatedDisableEnableCommandsRefactored demonstrates robust disable/enable testing
func TestConsolidatedDisableEnableCommandsRefactored(t *testing.T) {
	// Create isolated test helper
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Test putting a secret first
	t.Run("put_test_secret", func(t *testing.T) {
		output, err := helper.RunCommand("put", "test-secret", "test-value")
		if err != nil {
			t.Fatalf("failed to put secret: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "stored") {
			t.Fatalf("expected success message, got: %s", output)
		}
	})

	// Test disabling the secret
	t.Run("disable_secret", func(t *testing.T) {
		output, err := helper.RunCommand("disable", "secret", "test-secret")
		if err != nil {
			t.Fatalf("failed to disable secret: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "disabled") {
			t.Fatalf("expected disable confirmation, got: %s", output)
		}
	})

	// Test that disabled secret is excluded from list keys
	t.Run("list_keys_excludes_disabled", func(t *testing.T) {
		output, err := helper.RunCommand("list", "keys")
		if err != nil {
			t.Fatalf("failed to list keys: %v\nOutput: %s", err, output)
		}
		if strings.Contains(string(output), "test-secret") {
			t.Fatalf("disabled secret should not appear in keys list: %s", output)
		}
	})

	// Test that disabled secret appears in list disabled
	t.Run("list_disabled_shows_secret", func(t *testing.T) {
		output, err := helper.RunCommand("list", "disabled")
		if err != nil {
			t.Fatalf("failed to list disabled: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "test-secret") {
			t.Fatalf("disabled secret should appear in disabled list: %s", output)
		}
	})

	// Test that getting disabled secret fails
	t.Run("get_disabled_secret_fails", func(t *testing.T) {
		output, err := helper.RunCommand("get", "test-secret")
		if err == nil {
			t.Fatalf("expected error when getting disabled secret, but got none. Output: %s", output)
		}
		if !strings.Contains(string(output), "not found") {
			t.Fatalf("expected not found error message, got: %s", output)
		}
	})

	// Test enabling the secret
	t.Run("enable_secret", func(t *testing.T) {
		output, err := helper.RunCommand("enable", "secret", "test-secret")
		if err != nil {
			t.Fatalf("failed to enable secret: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "enabled") {
			t.Fatalf("expected enable confirmation, got: %s", output)
		}
	})

	// Test that enabled secret appears in keys list again
	t.Run("list_keys_includes_enabled_secret", func(t *testing.T) {
		output, err := helper.RunCommand("list", "keys")
		if err != nil {
			t.Fatalf("failed to list keys: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "test-secret") {
			t.Fatalf("enabled secret should appear in keys list: %s", output)
		}
	})

	// Test that getting enabled secret works
	t.Run("get_enabled_secret_works", func(t *testing.T) {
		output, err := helper.RunCommand("get", "test-secret")
		if err != nil {
			t.Fatalf("failed to get enabled secret: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "test-value") {
			t.Fatalf("expected secret value, got: %s", output)
		}
	})

	// Test error cases
	t.Run("disable_nonexistent_secret", func(t *testing.T) {
		output, err := helper.RunCommand("disable", "secret", "nonexistent")
		if err == nil {
			t.Fatalf("expected error when disabling nonexistent secret, but got none. Output: %s", output)
		}
	})

	t.Run("enable_nonexistent_secret", func(t *testing.T) {
		output, err := helper.RunCommand("enable", "secret", "nonexistent")
		if err == nil {
			t.Fatalf("expected error when enabling nonexistent secret, but got none. Output: %s", output)
		}
	})

	// Test authentication requirements
	t.Run("disable_without_token", func(t *testing.T) {
		output, err := helper.RunCommandWithoutToken("disable", "secret", "test-secret")
		if err == nil {
			t.Fatalf("expected error when disabling without token, but got none. Output: %s", output)
		}
		if !strings.Contains(string(output), "authentication required") && !strings.Contains(string(output), "invalid token") {
			t.Fatalf("expected authentication error, got: %s", output)
		}
	})

	t.Run("enable_without_token", func(t *testing.T) {
		output, err := helper.RunCommandWithoutToken("enable", "secret", "test-secret")
		if err == nil {
			t.Fatalf("expected error when enabling without token, but got none. Output: %s", output)
		}
		if !strings.Contains(string(output), "authentication required") && !strings.Contains(string(output), "invalid token") {
			t.Fatalf("expected authentication error, got: %s", output)
		}
	})
}
