package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// TestValidateKeyName tests the input validation logic for secret keys
func TestValidateKeyName(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_simple_key",
			key:         "validkey",
			expectError: false,
		},
		{
			name:        "valid_key_with_underscore",
			key:         "valid_key",
			expectError: false,
		},
		{
			name:        "valid_key_with_dash",
			key:         "valid-key",
			expectError: false,
		},
		{
			name:        "valid_key_with_tab",
			key:         "test\tkey",
			expectError: false,
		},
		{
			name:        "valid_key_with_newline",
			key:         "test\nkey",
			expectError: false,
		},
		{
			name:        "valid_key_with_carriage_return",
			key:         "test\rkey",
			expectError: false,
		},
		{
			name:        "empty_key",
			key:         "",
			expectError: true,
			errorMsg:    "key name cannot be empty",
		},
		{
			name:        "whitespace_only_key",
			key:         "   ",
			expectError: true,
			errorMsg:    "key name cannot be empty",
		},
		{
			name:        "null_byte_injection",
			key:         "test\x00key",
			expectError: true,
			errorMsg:    "key name cannot contain null bytes",
		},
		{
			name:        "control_character_injection",
			key:         "test\x01key",
			expectError: true,
			errorMsg:    "key name cannot contain control characters",
		},
		{
			name:        "control_character_del",
			key:         "test\x1fkey",
			expectError: true,
			errorMsg:    "key name cannot contain control characters",
		},
		{
			name:        "path_separator_forward_slash",
			key:         "test/key",
			expectError: true,
			errorMsg:    "key name cannot contain path separators or path traversal sequences",
		},
		{
			name:        "path_separator_backslash",
			key:         "test\\key",
			expectError: true,
			errorMsg:    "key name cannot contain path separators or path traversal sequences",
		},
		{
			name:        "path_traversal_dots",
			key:         "test..key",
			expectError: true,
			errorMsg:    "key name cannot contain path separators or path traversal sequences",
		},
		{
			name:        "path_traversal_sequence",
			key:         "../secrets",
			expectError: true,
			errorMsg:    "key name cannot contain path separators or path traversal sequences",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeyName(tt.key)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for key %q, but got none", tt.key)
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for key %q, but got: %v", tt.key, err)
				}
			}
		})
	}
}

// validateKeyName extracts the validation logic to be testable
func validateKeyName(key string) error {
	// This mirrors the validation logic in put.go
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("key name cannot be empty")
	}

	// Check for null bytes and other problematic characters
	if strings.Contains(key, "\x00") {
		return fmt.Errorf("key name cannot contain null bytes")
	}

	// Check for control characters (0x00-0x1F except \t, \n, \r)
	for _, r := range key {
		if r < 0x20 && r != 0x09 && r != 0x0A && r != 0x0D {
			return fmt.Errorf("key name cannot contain control characters")
		}
	}

	// Check for path traversal attempts
	if strings.Contains(key, "..") || strings.Contains(key, "/") || strings.Contains(key, "\\") {
		return fmt.Errorf("key name cannot contain path separators or path traversal sequences")
	}

	return nil
}
