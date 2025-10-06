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
package cmd

import (
	"os"
	"testing"

	internal "simple-secrets/internal/auth"
)

func TestDeleteCommandTokenResolution(t *testing.T) {
	// This test ensures the delete command properly resolves tokens from environment variables
	// Regression test for: Delete command environment token parsing issue

	tests := []struct {
		name          string
		envToken      string
		flagToken     string
		expectSuccess bool
		expectEmpty   bool
	}{
		{
			name:          "environment_token_only",
			envToken:      "test-env-token-123",
			flagToken:     "",
			expectSuccess: true,
			expectEmpty:   false,
		},
		{
			name:          "flag_token_only",
			envToken:      "",
			flagToken:     "test-flag-token-456",
			expectSuccess: true,
			expectEmpty:   false,
		},
		{
			name:          "no_token",
			envToken:      "",
			flagToken:     "",
			expectSuccess: false,
			expectEmpty:   true,
		},
		{
			name:          "flag_overrides_env",
			envToken:      "env-token",
			flagToken:     "flag-token",
			expectSuccess: true,
			expectEmpty:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			originalToken := os.Getenv("SIMPLE_SECRETS_TOKEN")
			defer func() {
				if originalToken != "" {
					os.Setenv("SIMPLE_SECRETS_TOKEN", originalToken)
					return
				}
				os.Unsetenv("SIMPLE_SECRETS_TOKEN")
			}()

			if tt.envToken != "" {
				os.Setenv("SIMPLE_SECRETS_TOKEN", tt.envToken)
			}
			if tt.envToken == "" {
				os.Unsetenv("SIMPLE_SECRETS_TOKEN")
			}

			// Test token resolution logic that delete command should use
			resolvedToken, err := internal.ResolveToken(tt.flagToken)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if resolvedToken == "" {
					t.Error("Expected non-empty resolved token")
				}

				// Verify correct token precedence
				if tt.flagToken != "" && resolvedToken != tt.flagToken {
					t.Errorf("Expected flag token %q, got %q", tt.flagToken, resolvedToken)
				}
				if tt.flagToken == "" && tt.envToken != "" && resolvedToken != tt.envToken {
					t.Errorf("Expected env token %q, got %q", tt.envToken, resolvedToken)
				}
				return
			}

			if err == nil {
				t.Error("Expected error but got none")
			}

			// Critical regression test: should not get "empty token" when no token is provided
			// Should get "authentication required" message instead
			if err.Error() == "empty token" {
				t.Error("Got 'empty token' error - this indicates regression in token resolution logic")
			}
		})
	}
}

func TestDeleteCommandEnvironmentTokenIntegration(t *testing.T) {
	// Integration test to ensure delete command properly processes environment tokens
	// This tests the critical fix: environment token resolution in delete command

	// Set a test environment token
	originalToken := os.Getenv("SIMPLE_SECRETS_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("SIMPLE_SECRETS_TOKEN", originalToken)
			return
		}
		os.Unsetenv("SIMPLE_SECRETS_TOKEN")
	}()

	testToken := "test-environment-token-789"
	os.Setenv("SIMPLE_SECRETS_TOKEN", testToken)

	// Test direct ResolveToken call with empty flag (this is what delete command now does)
	// Previously delete command would manually check flags and fail with "empty token"
	// Now it properly delegates to ResolveToken which checks environment
	resolvedToken, err := internal.ResolveToken("")

	// Should successfully resolve to the environment token
	if err != nil {
		t.Errorf("Token resolution failed: %v", err)
	}

	if resolvedToken != testToken {
		t.Errorf("Expected resolved token %q, got %q", testToken, resolvedToken)
	}

	// Critical regression test: ensure we don't get "empty token" when environment is set
	if err != nil && err.Error() == "empty token" {
		t.Error("REGRESSION: Delete command getting 'empty token' error when environment token is set")
	}

	// Test with unset environment (should fail properly, not with "empty token")
	os.Unsetenv("SIMPLE_SECRETS_TOKEN")
	_, err = internal.ResolveToken("")
	if err == nil {
		t.Error("Expected error when no token available")
	}

	// Should get proper "authentication required" message, not "empty token"
	if err != nil && err.Error() == "empty token" {
		t.Error("REGRESSION: Getting 'empty token' instead of proper authentication message")
	}
}
