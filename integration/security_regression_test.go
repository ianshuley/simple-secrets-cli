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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simple-secrets/integration/testing_framework"
)

// TestEmptyTokenBypassVulnerability tests the specific security issue where --token "" was accepted
func TestEmptyTokenBypassVulnerability(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name   string
		args   []string
		errMsg string
	}{
		{
			name:   "explicit_empty_token_list",
			args:   []string{"--token", "", "list", "keys"},
			errMsg: "authentication required: token cannot be empty",
		},
		{
			name:   "explicit_empty_token_put",
			args:   []string{"--token", "", "put", "key", "value"},
			errMsg: "authentication required: token cannot be empty",
		},
		{
			name:   "explicit_empty_token_delete",
			args:   []string{"--token", "", "delete", "key"},
			errMsg: "authentication required: token cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := env.RunRawCommand(tt.args, env.CleanEnvironment(), "")
			testing_framework.Assert(t, output, err).
				Failure().
				Contains(tt.errMsg)
		})
	}
}

// TestDirectoryPermissionsVulnerability tests that directories are created with 700 permissions
func TestDirectoryPermissionsVulnerability(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// The environment automatically creates the config directory during setup
	configDir := filepath.Join(env.TempDir(), ".simple-secrets")
	stat, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}

	perm := stat.Mode().Perm()
	expected := os.FileMode(0700)
	if perm != expected {
		t.Errorf("config directory has insecure permissions. Expected %s, got %s", expected, perm)
	}
}

// TestInputValidationVulnerabilities tests that malicious key names are rejected
func TestInputValidationVulnerabilities(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	maliciousKeys := []struct {
		name      string
		key       string
		putErr    string // Expected error for put command
		getErr    string // Expected error for get command
		deleteErr string // Expected error for delete command
	}{
		{
			name:      "path_traversal_dotdot",
			key:       "../../../etc/passwd",
			putErr:    "key name cannot contain path separators",
			getErr:    "secret not found",
			deleteErr: "file does not exist",
		},
		{
			name:      "absolute_path",
			key:       "/etc/passwd",
			putErr:    "key name cannot contain path separators",
			getErr:    "secret not found",
			deleteErr: "file does not exist",
		},
		{
			name:      "backslash_path",
			key:       "..\\..\\windows\\system32",
			putErr:    "key name cannot contain path separators",
			getErr:    "secret not found",
			deleteErr: "file does not exist",
		},
	}

	for _, tt := range maliciousKeys {
		t.Run(tt.name, func(t *testing.T) {
			// Test with put command - should validate input
			output, err := env.CLI().Put(tt.key, "value")
			testing_framework.Assert(t, output, err).
				Failure().
				Contains(tt.putErr)

			// Test with get command - different behavior
			output, err = env.CLI().Get(tt.key)
			testing_framework.Assert(t, output, err).
				Failure().
				Contains(tt.getErr)

			// Test with delete command - different behavior
			output, err = env.CLI().Delete(tt.key)
			testing_framework.Assert(t, output, err).
				Failure().
				Contains(tt.deleteErr)
		})
	}

	// Separate test for null byte which may cause different behavior
	t.Run("null_byte", func(t *testing.T) {
		key := "key\x00"

		// Put should reject it or handle it gracefully
		output, err := env.CLI().Put(key, "value")
		testing_framework.Assert(t, output, err).Failure()

		// Get and delete should also fail gracefully
		output, err = env.CLI().Get(key)
		testing_framework.Assert(t, output, err).Failure()

		output, err = env.CLI().Delete(key)
		testing_framework.Assert(t, output, err).Failure()
	})
}

// TestPermissionEscalationVulnerability tests that users cannot escalate their permissions
func TestPermissionEscalationVulnerability(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Create a reader user
	userOutput, err := env.CLI().Users().Create("reader", "reader")
	testing_framework.Assert(t, userOutput, err).Success()

	readerToken := testing_framework.ParseToken(string(userOutput))
	if readerToken == "" {
		t.Fatalf("Could not extract reader token")
	}

	// Test that reader cannot create users (admin operation)
	output, err := env.RunRawCommand([]string{"create-user", "hacker", "admin", "--token", readerToken}, env.CleanEnvironment(), "")
	testing_framework.Assert(t, output, err).
		Failure().
		Contains("permission denied")

	// Test that reader cannot disable other users
	output, err = env.RunRawCommand([]string{"disable", "user", "admin", "--token", readerToken}, env.CleanEnvironment(), "")
	testing_framework.Assert(t, output, err).
		Failure().
		Contains("permission denied")

	// Test that reader cannot rotate master key
	output, err = env.RunRawCommand([]string{"rotate", "master-key", "--yes", "--token", readerToken}, env.CleanEnvironment(), "")
	testing_framework.Assert(t, output, err).
		Failure().
		Contains("permission denied")
}

// TestTokenSecurityVulnerability tests token handling security
func TestTokenSecurityVulnerability(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Test that expired/invalid tokens are rejected
	invalidTokens := []string{
		"invalid-token",
		"",
		"a",
		"token-with-invalid-chars!@#$%",
		strings.Repeat("a", 1000), // extremely long token
	}

	for i, token := range invalidTokens {
		t.Run(fmt.Sprintf("invalid_token_%d", i), func(t *testing.T) {
			output, err := env.RunRawCommand([]string{"list", "keys", "--token", token}, env.CleanEnvironment(), "")
			testing_framework.Assert(t, output, err).Failure()
		})
	}
}
