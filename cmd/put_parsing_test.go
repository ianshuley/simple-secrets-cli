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

func TestParsePositiveInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "valid_positive_integer",
			input:    "42",
			expected: 42,
		},
		{
			name:     "zero_returns_zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "negative_returns_negative",
			input:    "-5",
			expected: -5,
		},
		{
			name:     "non_numeric_returns_zero",
			input:    "abc",
			expected: 0,
		},
		{
			name:     "empty_string_returns_zero",
			input:    "",
			expected: 0,
		},
		{
			name:     "whitespace_returns_zero",
			input:    "   ",
			expected: 0,
		},
		{
			name:     "large_number",
			input:    "1024",
			expected: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePositiveInteger(tt.input)
			if result != tt.expected {
				t.Errorf("parsePositiveInteger(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsTokenFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		position int
		expected bool
	}{
		{
			name:     "token_flag_with_value",
			args:     []string{"--token", "mytoken"},
			position: 0,
			expected: true,
		},
		{
			name:     "not_token_flag",
			args:     []string{"--generate", "key"},
			position: 0,
			expected: false,
		},
		{
			name:     "token_flag_at_end_no_value",
			args:     []string{"key", "--token"},
			position: 1,
			expected: false, // No value after --token
		},
		{
			name:     "token_flag_with_empty_next",
			args:     []string{"--token", ""},
			position: 0,
			expected: true, // Has next arg (even if empty)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only test if position is within bounds
			if tt.position >= len(tt.args) {
				t.Skip("Position out of bounds - function would panic")
				return
			}

			result := isTokenFlag(tt.args, tt.position)
			if result != tt.expected {
				t.Errorf("isTokenFlag(%v, %d) = %t, want %t", tt.args, tt.position, result, tt.expected)
			}
		})
	}
}

func TestIsLengthFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		position int
		expected bool
	}{
		{
			name:     "length_flag_with_value",
			args:     []string{"--length", "64"},
			position: 0,
			expected: true,
		},
		{
			name:     "not_length_flag",
			args:     []string{"--generate", "key"},
			position: 0,
			expected: false,
		},
		{
			name:     "length_flag_at_end_no_value",
			args:     []string{"key", "--length"},
			position: 1,
			expected: false, // No value after --length
		},
		{
			name:     "short_length_flag_with_value",
			args:     []string{"-l", "32"},
			position: 0,
			expected: true,
		},
		{
			name:     "short_length_flag_at_end_no_value",
			args:     []string{"key", "-l"},
			position: 1,
			expected: false, // No value after -l
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only test if position is within bounds
			if tt.position >= len(tt.args) {
				t.Skip("Position out of bounds - function would panic")
				return
			}

			result := isLengthFlag(tt.args, tt.position)
			if result != tt.expected {
				t.Errorf("isLengthFlag(%v, %d) = %t, want %t", tt.args, tt.position, result, tt.expected)
			}
		})
	}
}

func TestHasTokenValue(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		flagPosition int
		expected     bool
	}{
		{
			name:         "has_value",
			args:         []string{"--token", "mytoken"},
			flagPosition: 0,
			expected:     true,
		},
		{
			name:         "no_value_at_end",
			args:         []string{"key", "--token"},
			flagPosition: 1,
			expected:     false,
		},
		{
			name:         "next_is_another_flag_still_has_value",
			args:         []string{"--token", "--generate"},
			flagPosition: 0,
			expected:     true, // Just checks if next position exists
		},
		{
			name:         "position_out_of_bounds",
			args:         []string{"--token"},
			flagPosition: 5,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasTokenValue(tt.args, tt.flagPosition)
			if result != tt.expected {
				t.Errorf("hasTokenValue(%v, %d) = %t, want %t", tt.args, tt.flagPosition, result, tt.expected)
			}
		})
	}
}

func TestHasLengthValue(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		flagPosition int
		expected     bool
	}{
		{
			name:         "has_value",
			args:         []string{"--length", "64"},
			flagPosition: 0,
			expected:     true,
		},
		{
			name:         "no_value_at_end",
			args:         []string{"key", "--length"},
			flagPosition: 1,
			expected:     false,
		},
		{
			name:         "next_is_another_flag_still_has_value",
			args:         []string{"--length", "--generate"},
			flagPosition: 0,
			expected:     true, // Just checks if next position exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasLengthValue(tt.args, tt.flagPosition)
			if result != tt.expected {
				t.Errorf("hasLengthValue(%v, %d) = %t, want %t", tt.args, tt.flagPosition, result, tt.expected)
			}
		})
	}
}

func TestProcessTokenFlag(t *testing.T) {
	tests := []struct {
		name                  string
		args                  []string
		flagPosition          int
		expectedToken         string
		expectedExplicitlySet bool
		expectedNextPosition  int
	}{
		{
			name:                  "token_with_value",
			args:                  []string{"--token", "mytoken", "key"},
			flagPosition:          0,
			expectedToken:         "mytoken",
			expectedExplicitlySet: true,
			expectedNextPosition:  1, // Position of the token value
		},
		{
			name:                  "token_with_next_arg",
			args:                  []string{"--token", "key"},
			flagPosition:          0,
			expectedToken:         "key", // Takes next arg as token value
			expectedExplicitlySet: true,
			expectedNextPosition:  1, // Position of the token value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token string
			var tokenExplicitlySet bool

			nextPos := processTokenFlag(tt.args, tt.flagPosition, &token, &tokenExplicitlySet)

			if token != tt.expectedToken {
				t.Errorf("processTokenFlag() token = %q, want %q", token, tt.expectedToken)
			}
			if tokenExplicitlySet != tt.expectedExplicitlySet {
				t.Errorf("processTokenFlag() tokenExplicitlySet = %t, want %t", tokenExplicitlySet, tt.expectedExplicitlySet)
			}
			if nextPos != tt.expectedNextPosition {
				t.Errorf("processTokenFlag() nextPos = %d, want %d", nextPos, tt.expectedNextPosition)
			}
		})
	}
}

func TestProcessLengthFlag(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		flagPosition         int
		expectedLength       int
		expectedNextPosition int
	}{
		{
			name:                 "length_with_valid_value",
			args:                 []string{"--length", "128", "key"},
			flagPosition:         0,
			expectedLength:       128,
			expectedNextPosition: 1,
		},
		{
			name:                 "length_with_invalid_value",
			args:                 []string{"--length", "abc", "key"},
			flagPosition:         0,
			expectedLength:       32, // Default for zero/invalid
			expectedNextPosition: 1,
		},
		{
			name:                 "length_with_next_arg",
			args:                 []string{"--length", "key"},
			flagPosition:         0,
			expectedLength:       32, // parsePositiveInteger("key") = 0, so default
			expectedNextPosition: 1,  // Still processes next arg
		},
		{
			name:                 "short_length_with_valid_value",
			args:                 []string{"-l", "64", "key"},
			flagPosition:         0,
			expectedLength:       64,
			expectedNextPosition: 1,
		},
		{
			name:                 "short_length_with_invalid_value",
			args:                 []string{"-l", "xyz", "key"},
			flagPosition:         0,
			expectedLength:       32, // Default for zero/invalid
			expectedNextPosition: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length := 32 // Default

			nextPos := processLengthFlag(tt.args, tt.flagPosition, &length)

			if length != tt.expectedLength {
				t.Errorf("processLengthFlag() length = %d, want %d", length, tt.expectedLength)
			}
			if nextPos != tt.expectedNextPosition {
				t.Errorf("processLengthFlag() nextPos = %d, want %d", nextPos, tt.expectedNextPosition)
			}
		})
	}
}

func TestShouldShowHelp(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "help_flag_long",
			args:     []string{"--help"},
			expected: true,
		},
		{
			name:     "help_flag_short",
			args:     []string{"-h"},
			expected: true,
		},
		{
			name:     "help_with_other_args",
			args:     []string{"key", "--help"},
			expected: true,
		},
		{
			name:     "no_help_flag",
			args:     []string{"key", "value"},
			expected: false,
		},
		{
			name:     "empty_args",
			args:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldShowHelp(tt.args)
			if result != tt.expected {
				t.Errorf("shouldShowHelp(%v) = %t, want %t", tt.args, result, tt.expected)
			}
		})
	}
}

func TestIsEmptyToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "empty_string",
			token:    "",
			expected: true,
		},
		{
			name:     "whitespace_only",
			token:    "   ",
			expected: true,
		},
		{
			name:     "valid_token",
			token:    "abc123",
			expected: false,
		},
		{
			name:     "token_with_spaces",
			token:    " abc123 ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmptyToken(tt.token)
			if result != tt.expected {
				t.Errorf("isEmptyToken(%q) = %t, want %t", tt.token, result, tt.expected)
			}
		})
	}
}

func TestCreateEmptyTokenError(t *testing.T) {
	err := createEmptyTokenError()
	if err == nil {
		t.Error("createEmptyTokenError() should return an error")
		return
	}

	errorMsg := err.Error()
	expectedContent := []string{
		"authentication required",
		"token cannot be empty",
		"simple-secrets --token",
		"SIMPLE_SECRETS_TOKEN",
	}

	for _, content := range expectedContent {
		if !strings.Contains(errorMsg, content) {
			t.Errorf("createEmptyTokenError() message should contain %q, got: %s", content, errorMsg)
		}
	}
}

func TestExtractArgumentsAndFlags(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		expectedFilteredArgs []string
		expectedToken        string
		expectedTokenSet     bool
		expectedGenerate     bool
		expectedLength       int
	}{
		{
			name:                 "simple_put_command",
			args:                 []string{"key", "value"},
			expectedFilteredArgs: []string{"key", "value"},
			expectedToken:        "",
			expectedTokenSet:     false,
			expectedGenerate:     false,
			expectedLength:       32,
		},
		{
			name:                 "generate_command",
			args:                 []string{"--generate", "key"},
			expectedFilteredArgs: []string{"key"},
			expectedToken:        "",
			expectedTokenSet:     false,
			expectedGenerate:     true,
			expectedLength:       32,
		},
		{
			name:                 "generate_with_length",
			args:                 []string{"--generate", "--length", "64", "key"},
			expectedFilteredArgs: []string{"key"},
			expectedToken:        "",
			expectedTokenSet:     false,
			expectedGenerate:     true,
			expectedLength:       64,
		},
		{
			name:                 "with_token_flag",
			args:                 []string{"--token", "mytoken", "key", "value"},
			expectedFilteredArgs: []string{"key", "value"},
			expectedToken:        "mytoken",
			expectedTokenSet:     true,
			expectedGenerate:     false,
			expectedLength:       32,
		},
		{
			name:                 "all_flags_together",
			args:                 []string{"--token", "mytoken", "--generate", "--length", "128", "key"},
			expectedFilteredArgs: []string{"key"},
			expectedToken:        "mytoken",
			expectedTokenSet:     true,
			expectedGenerate:     true,
			expectedLength:       128,
		},
		{
			name:                 "short_generate_flag",
			args:                 []string{"-g", "key"},
			expectedFilteredArgs: []string{"key"},
			expectedToken:        "",
			expectedTokenSet:     false,
			expectedGenerate:     true,
			expectedLength:       32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token string
			var tokenExplicitlySet bool
			var generate bool
			length := 32

			filteredArgs := extractArgumentsAndFlags(tt.args, &token, &tokenExplicitlySet, &generate, &length)

			if !stringSlicesEqual(filteredArgs, tt.expectedFilteredArgs) {
				t.Errorf("extractArgumentsAndFlags() filteredArgs = %v, want %v", filteredArgs, tt.expectedFilteredArgs)
			}
			if token != tt.expectedToken {
				t.Errorf("extractArgumentsAndFlags() token = %q, want %q", token, tt.expectedToken)
			}
			if tokenExplicitlySet != tt.expectedTokenSet {
				t.Errorf("extractArgumentsAndFlags() tokenExplicitlySet = %t, want %t", tokenExplicitlySet, tt.expectedTokenSet)
			}
			if generate != tt.expectedGenerate {
				t.Errorf("extractArgumentsAndFlags() generate = %t, want %t", generate, tt.expectedGenerate)
			}
			if length != tt.expectedLength {
				t.Errorf("extractArgumentsAndFlags() length = %d, want %d", length, tt.expectedLength)
			}
		})
	}
}

// Helper function to compare string slices
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
