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
	"strings"
	"testing"
)

func TestValidateSecureInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		config      ValidationConfig
		wantErr     bool
		errContains string
	}{
		// Username validation tests
		{
			name:   "valid_username",
			input:  "alice",
			config: UsernameValidationConfig,
		},
		{
			name:        "empty_username",
			input:       "",
			config:      UsernameValidationConfig,
			wantErr:     true,
			errContains: "username cannot be empty",
		},
		{
			name:        "username_path_traversal_dots",
			input:       "../etc/passwd",
			config:      UsernameValidationConfig,
			wantErr:     true,
			errContains: "username cannot contain path separators or path traversal sequences",
		},
		{
			name:        "username_path_traversal_slash",
			input:       "user/with/slash",
			config:      UsernameValidationConfig,
			wantErr:     true,
			errContains: "username cannot contain path separators or path traversal sequences",
		},
		{
			name:        "username_control_chars",
			input:       "user\x01name",
			config:      UsernameValidationConfig,
			wantErr:     true,
			errContains: "username cannot contain control characters",
		},

		// Secret key validation tests
		{
			name:   "valid_secret_key",
			input:  "api-key",
			config: SecretKeyValidationConfig,
		},
		{
			name:        "empty_secret_key",
			input:       "   ",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot be empty",
		},
		{
			name:   "secret_key_with_allowed_control_chars",
			input:  "key\twith\ntab\rand\rreturn",
			config: SecretKeyValidationConfig,
		},
		{
			name:        "secret_key_with_disallowed_control_chars",
			input:       "key\x01with\x02control",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain control characters",
		},
		{
			name:        "secret_key_path_traversal",
			input:       "../../etc/passwd",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain path separators or path traversal sequences",
		},

		// Shell metacharacter injection tests
		{
			name:        "command_substitution_dollar_paren",
			input:       "key$(whoami)",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "command_substitution_backtick",
			input:       "key`whoami`",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "dollar_variable_expansion",
			input:       "key$HOME",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "dollar_brace_expansion",
			input:       "key${USER}",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "pipe_character",
			input:       "key|grep secret",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "semicolon_separator",
			input:       "key;echo test",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "ampersand_background",
			input:       "key&wget example.com",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "redirect_output",
			input:       "key>file.txt",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "redirect_input",
			input:       "key<file.txt",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "glob_asterisk",
			input:       "key*wildcard",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "glob_question",
			input:       "key?wildcard",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "brace_expansion",
			input:       "key{a,b,c}",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "bracket_glob",
			input:       "key[0-9]",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "tilde_home_expansion",
			input:       "key~user",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "exclamation_history",
			input:       "key!!",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},
		{
			name:        "hash_comment",
			input:       "key#comment",
			config:      SecretKeyValidationConfig,
			wantErr:     true,
			errContains: "key name cannot contain shell metacharacters",
		},

		// Custom configuration tests
		{
			name:  "custom_config_allow_empty",
			input: "",
			config: ValidationConfig{
				EntityType:         "test field",
				AllowEmpty:         true,
				AllowControlChars:  false,
				AllowPathTraversal: false,
			},
		},
		{
			name:  "custom_config_allow_path_traversal",
			input: "../some/path",
			config: ValidationConfig{
				EntityType:         "test field",
				AllowEmpty:         false,
				AllowControlChars:  false,
				AllowPathTraversal: true,
			},
		},
		{
			name:  "custom_config_allow_control_chars",
			input: "test\x01value",
			config: ValidationConfig{
				EntityType:          "test field",
				AllowEmpty:          false,
				AllowControlChars:   true,
				AllowPathTraversal:  false,
				AllowShellMetachars: false,
			},
		},
		{
			name:  "custom_config_allow_shell_metachars",
			input: "key$(dangerous)",
			config: ValidationConfig{
				EntityType:          "test field",
				AllowEmpty:          false,
				AllowControlChars:   false,
				AllowPathTraversal:  false,
				AllowShellMetachars: true, // Allow shell metacharacters
			},
		},
		{
			name:  "custom_config_disallow_shell_metachars",
			input: "key$(dangerous)",
			config: ValidationConfig{
				EntityType:          "test field",
				AllowEmpty:          false,
				AllowControlChars:   false,
				AllowPathTraversal:  false,
				AllowShellMetachars: false, // Disallow shell metacharacters
			},
			wantErr:     true,
			errContains: "test field cannot contain shell metacharacters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecureInput(tt.input, tt.config)

			if tt.wantErr {
				assertExpectedError(t, err, tt.errContains)
				return
			}

			assertNoError(t, err)
		})
	}
}

func TestValidationConfigDefaults(t *testing.T) {
	// Test that our predefined configs have the expected values
	t.Run("username_config", func(t *testing.T) {
		config := UsernameValidationConfig
		if config.EntityType != "username" {
			t.Errorf("expected EntityType 'username', got %q", config.EntityType)
		}
		if config.AllowEmpty {
			t.Error("expected AllowEmpty to be false for usernames")
		}
		if config.AllowControlChars {
			t.Error("expected AllowControlChars to be false for usernames")
		}
		if config.AllowPathTraversal {
			t.Error("expected AllowPathTraversal to be false for usernames")
		}
		if config.AllowShellMetachars {
			t.Error("expected AllowShellMetachars to be false for usernames")
		}
	})

	t.Run("secret_key_config", func(t *testing.T) {
		config := SecretKeyValidationConfig
		if config.EntityType != "key name" {
			t.Errorf("expected EntityType 'key name', got %q", config.EntityType)
		}
		if config.AllowEmpty {
			t.Error("expected AllowEmpty to be false for secret keys")
		}
		if config.AllowControlChars {
			t.Error("expected AllowControlChars to be false for secret keys")
		}
		if config.AllowPathTraversal {
			t.Error("expected AllowPathTraversal to be false for secret keys")
		}
		// Check that tab, LF, CR are allowed
		expectedAllowed := []rune{0x09, 0x0A, 0x0D}
		if len(config.AllowedControlChars) != len(expectedAllowed) {
			t.Errorf("expected %d allowed control chars, got %d", len(expectedAllowed), len(config.AllowedControlChars))
		}
		for i, r := range expectedAllowed {
			if i >= len(config.AllowedControlChars) || config.AllowedControlChars[i] != r {
				t.Errorf("expected allowed control char %#x at index %d", r, i)
			}
		}
	})
}

// contains is a simple helper for string containment checks
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// assertExpectedError validates that an error occurred and optionally contains expected text
func assertExpectedError(t *testing.T, err error, expectedContent string) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error but got none")
		return
	}

	if expectedContent != "" && !contains(err.Error(), expectedContent) {
		t.Errorf("expected error to contain %q, got: %v", expectedContent, err)
	}
}

// assertNoError validates that no error occurred
func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}
}
