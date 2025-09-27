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
	"testing"
	"time"

	"simple-secrets/internal"
)

func TestParseRole(t *testing.T) {
	tests := []struct {
		name        string
		roleStr     string
		expected    internal.Role
		expectError bool
	}{
		{
			name:        "valid_admin_role",
			roleStr:     "admin",
			expected:    internal.RoleAdmin,
			expectError: false,
		},
		{
			name:        "valid_reader_role",
			roleStr:     "reader",
			expected:    internal.RoleReader,
			expectError: false,
		},
		{
			name:        "invalid_role_returns_error",
			roleStr:     "invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty_string_returns_error",
			roleStr:     "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "case_sensitive_admin_uppercase",
			roleStr:     "ADMIN",
			expected:    "",
			expectError: true,
		},
		{
			name:        "case_sensitive_reader_uppercase",
			roleStr:     "READER",
			expected:    "",
			expectError: true,
		},
		{
			name:        "case_sensitive_mixed_case",
			roleStr:     "Admin",
			expected:    "",
			expectError: true,
		},
		{
			name:        "whitespace_only_returns_error",
			roleStr:     "   ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "role_with_leading_whitespace",
			roleStr:     " admin",
			expected:    "",
			expectError: true,
		},
		{
			name:        "role_with_trailing_whitespace",
			roleStr:     "admin ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "numeric_role_returns_error",
			roleStr:     "123",
			expected:    "",
			expectError: true,
		},
		{
			name:        "special_characters_return_error",
			roleStr:     "admin!",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRole(tt.roleStr)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseRole(%q) expected error, but got none", tt.roleStr)
				}
				if result != tt.expected {
					t.Errorf("parseRole(%q) expected role %q, got %q", tt.roleStr, tt.expected, result)
				}
			} else {
				if err != nil {
					t.Errorf("parseRole(%q) unexpected error: %v", tt.roleStr, err)
				}
				if result != tt.expected {
					t.Errorf("parseRole(%q) expected role %q, got %q", tt.roleStr, tt.expected, result)
				}
			}
		})
	}
}

func TestGetTokenRotationDisplay(t *testing.T) {
	// Fixed timestamp for consistent testing
	fixedTime := time.Date(2025, 9, 27, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name      string
		timestamp *time.Time
		expected  string
	}{
		{
			name:      "nil_timestamp_shows_legacy_user",
			timestamp: nil,
			expected:  "Unknown (legacy user)",
		},
		{
			name:      "valid_timestamp_formatted_correctly",
			timestamp: &fixedTime,
			expected:  "2025-09-27 14:30:45",
		},
		{
			name:      "zero_time_formatted",
			timestamp: &time.Time{},
			expected:  "0001-01-01 00:00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTokenRotationDisplay(tt.timestamp)
			if result != tt.expected {
				t.Errorf("getTokenRotationDisplay() expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSafeDisplayFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple_string",
			input:    "test-key",
			expected: `"test-key"`,
		},
		{
			name:     "empty_string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "string_with_spaces",
			input:    "test key with spaces",
			expected: `"test key with spaces"`,
		},
		{
			name:     "string_with_quotes",
			input:    `key"with"quotes`,
			expected: `"key\"with\"quotes"`,
		},
		{
			name:     "string_with_newlines",
			input:    "key\nwith\nnewlines",
			expected: `"key\nwith\nnewlines"`,
		},
		{
			name:     "string_with_tabs",
			input:    "key\twith\ttabs",
			expected: `"key\twith\ttabs"`,
		},
		{
			name:     "string_with_backslashes",
			input:    `key\with\backslashes`,
			expected: `"key\\with\\backslashes"`,
		},
		{
			name:     "string_with_control_characters",
			input:    "key\x00\x01\x02",
			expected: `"key\x00\x01\x02"`,
		},
		{
			name:     "unicode_string",
			input:    "key-with-émojis-🔑",
			expected: `"key-with-émojis-🔑"`,
		},
		{
			name:     "string_with_carriage_return",
			input:    "key\rwith\rcarriage",
			expected: `"key\rwith\rcarriage"`,
		},
		{
			name:     "mixed_special_characters",
			input:    "key\"with\nmixed\tspecial\rchars\\",
			expected: `"key\"with\nmixed\tspecial\rchars\\"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeDisplayFormat(tt.input)
			if result != tt.expected {
				t.Errorf("safeDisplayFormat(%q) expected %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}
