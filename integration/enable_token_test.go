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

// TestEnableTokenIntegration tests the new enable token/user functionality
func TestEnableTokenIntegration(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Set up admin user
	adminToken := env.AdminToken()

	// Create a test user to disable/enable
	t.Run("create_test_user", func(t *testing.T) {
		output, err := env.CLI().Raw("create-user", "test-user", "reader", "--token", adminToken)
		testing_framework.Assert(t, output, err).Success()
	})

	// Extract the token from user creation (for later validation)
	var userToken string
	t.Run("extract_user_token", func(t *testing.T) {
		output, err := env.CLI().Raw("create-user", "token-test-user", "reader", "--token", adminToken)
		testing_framework.Assert(t, output, err).Success()

		// Extract token from output (format: "User created with token: TOKEN")
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "token:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					userToken = strings.TrimSpace(parts[1])
					break
				}
			}
		}
		if userToken == "" {
			t.Fatalf("could not extract user token from output: %s", output)
		}
	})

	// Disable user by username
	t.Run("disable_user_by_username", func(t *testing.T) {
		input := "yes\n"
		output, err := env.CLI().RawWithInput(input, "disable", "user", "test-user", "--token", adminToken)
		testing_framework.Assert(t, output, err).Success()
		if !strings.Contains(string(output), "Token disabled for user 'test-user'") {
			t.Fatalf("expected disable confirmation, got: %s", output)
		}
	})

	// Try to enable user (should work)
	t.Run("enable_user_by_username", func(t *testing.T) {
		input := "y\n"
		output, err := env.CLI().RawWithInput(input, "enable", "user", "test-user", "--token", adminToken)
		testing_framework.Assert(t, output, err).Success()
		if !strings.Contains(string(output), "New token generated for user 'test-user'") {
			t.Fatalf("expected enable confirmation, got: %s", output)
		}
	})

	// Test disable by token value, then enable
	t.Run("disable_by_token_value_then_enable", func(t *testing.T) {
		if userToken == "" {
			t.Skip("no user token available for testing")
		}

		// Disable by token value
		input := "yes\n"
		output, err := env.CLI().RawWithInput(input, "disable", "token", userToken, "--token", adminToken)
		testing_framework.Assert(t, output, err).Success()
		if !strings.Contains(string(output), "Token disabled for user 'token-test-user'") {
			t.Fatalf("expected disable by token confirmation, got: %s", output)
		}

		// Enable the disabled user
		input = "y\n"
		output, err = env.CLI().RawWithInput(input, "enable", "user", "token-test-user", "--token", adminToken)
		testing_framework.Assert(t, output, err).Success()
		if !strings.Contains(string(output), "New token generated for user 'token-test-user'") {
			t.Fatalf("expected enable after token disable confirmation, got: %s", output)
		}
	})

	// Test error cases
	t.Run("enable_nonexistent_user", func(t *testing.T) {
		input := "y\n"
		output, err := env.CLI().RawWithInput(input, "enable", "user", "nonexistent", "--token", adminToken)
		if err == nil {
			t.Fatalf("expected error when enabling nonexistent user, but got none. Output: %s", output)
		}
	})

	t.Run("enable_active_user", func(t *testing.T) {
		// Create an active user
		output, err := env.CLI().Raw("create-user", "active-user", "reader", "--token", adminToken)
		testing_framework.Assert(t, output, err).Success()

		// Try to enable (platform services allow token regeneration for active users)
		input := "y\n"
		output, err = env.CLI().RawWithInput(input, "enable", "user", "active-user", "--token", adminToken)
		if err != nil {
			t.Fatalf("unexpected error when enabling active user: %v. Output: %s", err, output)
		}
		// Platform services allow token regeneration for active users
		if !strings.Contains(string(output), "New token generated") {
			t.Fatalf("expected new token generation message, got: %s", output)
		}
	})

	// Test permission requirements
	t.Run("enable_requires_admin", func(t *testing.T) {
		// Create reader user
		readerOutput, err := env.CLI().Raw("create-user", "reader-user", "reader", "--token", adminToken)
		testing_framework.Assert(t, readerOutput, err).Success()

		// Extract reader token
		var readerToken string
		lines := strings.Split(string(readerOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, "token:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					readerToken = strings.TrimSpace(parts[1])
					break
				}
			}
		}

		if readerToken == "" {
			t.Fatalf("could not extract reader token")
		}

		// Try to enable with reader token (should fail)
		input := "y\n"
		output, err := env.CLI().RawWithInput(input, "enable", "user", "test-user", "--token", readerToken)
		if err == nil {
			t.Fatalf("expected permission error when non-admin tries to enable, but got none. Output: %s", output)
		}
		if !strings.Contains(string(output), "Permission denied: manage-users") {
			t.Fatalf("expected permission denied error, got: %s", output)
		}
	})
}
