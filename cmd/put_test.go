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
	"strings"
	"testing"
)

func TestGenerateSecretValue(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		wantErr     bool
		errContains string
	}{
		{
			name:   "default_length_32",
			length: 32,
		},
		{
			name:   "custom_length_64",
			length: 64,
		},
		{
			name:   "small_length_8",
			length: 8,
		},
		{
			name:        "zero_length",
			length:      0,
			wantErr:     true,
			errContains: "length must be positive",
		},
		{
			name:        "negative_length",
			length:      -5,
			wantErr:     true,
			errContains: "length must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateSecretValue(tt.length)

			if tt.wantErr {
				if err == nil {
					t.Errorf("generateSecretValue() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("generateSecretValue() error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("generateSecretValue() unexpected error: %v", err)
				return
			}

			if len(result) != tt.length {
				t.Errorf("generateSecretValue() length = %d, want %d", len(result), tt.length)
			}

			// Verify character set (base62 + symbols)
			const expectedCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+"
			for _, char := range result {
				if !strings.ContainsRune(expectedCharset, char) {
					t.Errorf("generateSecretValue() contains invalid character: %c", char)
				}
			}
		})
	}
}

func TestGenerateSecretValueUniqueness(t *testing.T) {
	// Generate multiple secrets and verify they are different
	secrets := make([]string, 10)
	for i := range secrets {
		secret, err := generateSecretValue(32)
		if err != nil {
			t.Fatalf("generateSecretValue() unexpected error: %v", err)
		}
		secrets[i] = secret
	}

	// Check that all secrets are unique
	for i := 0; i < len(secrets); i++ {
		for j := i + 1; j < len(secrets); j++ {
			if secrets[i] == secrets[j] {
				t.Errorf("generateSecretValue() generated duplicate secrets: %q", secrets[i])
			}
		}
	}
}

func TestIsGenerateFlag(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want bool
	}{
		{"long_flag", "--generate", true},
		{"short_flag", "-g", true},
		{"other_flag", "--token", false},
		{"not_flag", "value", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGenerateFlag(tt.arg); got != tt.want {
				t.Errorf("isGenerateFlag(%q) = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}

func TestValidatePutArgumentsWithGenerate(t *testing.T) {
	tests := []struct {
		name         string
		filteredArgs []string
		generate     bool
		wantKey      string
		wantValue    string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "normal_mode_valid",
			filteredArgs: []string{"api-key", "secret-value"},
			generate:     false,
			wantKey:      "api-key",
			wantValue:    "secret-value",
		},
		{
			name:         "normal_mode_missing_args",
			filteredArgs: []string{"api-key"},
			generate:     false,
			wantErr:      true,
			errContains:  "requires exactly 2 arguments",
		},
		{
			name:         "generate_mode_valid",
			filteredArgs: []string{"api-key"},
			generate:     true,
			wantKey:      "api-key",
			wantValue:    "", // empty value when generating
		},
		{
			name:         "generate_mode_no_key",
			filteredArgs: []string{},
			generate:     true,
			wantErr:      true,
			errContains:  "requires key argument when using --generate flag",
		},
		{
			name:         "generate_mode_with_value",
			filteredArgs: []string{"api-key", "manual-value"},
			generate:     true,
			wantErr:      true,
			errContains:  "cannot provide both --generate flag and manual value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, err := validatePutArguments(tt.filteredArgs, tt.generate)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePutArguments() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validatePutArguments() error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("validatePutArguments() unexpected error: %v", err)
				return
			}

			if key != tt.wantKey {
				t.Errorf("validatePutArguments() key = %q, want %q", key, tt.wantKey)
			}

			if value != tt.wantValue {
				t.Errorf("validatePutArguments() value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}

func TestDetermineAuthTokenWithExplicitFlag(t *testing.T) {
	// Save original environment
	originalToken := os.Getenv("SIMPLE_SECRETS_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("SIMPLE_SECRETS_TOKEN", originalToken)
		} else {
			os.Unsetenv("SIMPLE_SECRETS_TOKEN")
		}
	}()

	tests := []struct {
		name               string
		parsedToken        string
		wasTokenFlagUsed   bool
		envToken           string
		wantToken          string
		wantErr            bool
		errContains        string
	}{
		{
			name:             "explicit_flag_takes_precedence",
			parsedToken:      "flag-token",
			wasTokenFlagUsed: true,
			envToken:         "env-token",
			wantToken:        "flag-token",
		},
		{
			name:             "empty_explicit_flag_errors",
			parsedToken:      "",
			wasTokenFlagUsed: true,
			envToken:         "env-token",
			wantErr:          true,
			errContains:      "authentication required: token cannot be empty",
		},
		{
			name:             "whitespace_explicit_flag_errors",
			parsedToken:      "   ",
			wasTokenFlagUsed: true,
			envToken:         "env-token",
			wantErr:          true,
			errContains:      "authentication required: token cannot be empty",
		},
		{
			name:             "no_flag_uses_environment",
			parsedToken:      "",
			wasTokenFlagUsed: false,
			envToken:         "env-token-value",
			wantToken:        "env-token-value",
		},
		{
			name:             "no_flag_no_env_errors",
			parsedToken:      "",
			wasTokenFlagUsed: false,
			envToken:         "",
			wantErr:          true,
			errContains:      "authentication required: no token found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envToken != "" {
				os.Setenv("SIMPLE_SECRETS_TOKEN", tt.envToken)
			} else {
				os.Unsetenv("SIMPLE_SECRETS_TOKEN")
			}

			token, err := determineAuthTokenWithExplicitFlag(tt.parsedToken, tt.wasTokenFlagUsed)

			if tt.wantErr {
				if err == nil {
					t.Errorf("determineAuthTokenWithExplicitFlag() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("determineAuthTokenWithExplicitFlag() error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("determineAuthTokenWithExplicitFlag() unexpected error: %v", err)
				return
			}

			if token != tt.wantToken {
				t.Errorf("determineAuthTokenWithExplicitFlag() = %q, want %q", token, tt.wantToken)
			}
		})
	}
}