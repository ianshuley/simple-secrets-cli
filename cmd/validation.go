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
	"fmt"
	"strings"
)

// ValidationConfig controls what validation rules are applied to an input string
type ValidationConfig struct {
	EntityType          string // "key name", "username", etc. for error messages
	AllowEmpty          bool   // Whether empty strings are allowed
	AllowControlChars   bool   // Whether control characters are allowed
	AllowedControlChars []rune // Specific control chars that are allowed (e.g., tab, newline)
	AllowPathTraversal  bool   // Whether path separators and .. are allowed
	AllowShellMetachars bool   // Whether shell metacharacters are allowed (prevents command injection)
}

// ValidateSecureInput performs comprehensive validation on user input strings
// to prevent security issues like path traversal, control character injection, and command injection
func ValidateSecureInput(input string, config ValidationConfig) error {
	// First check for potential post-shell-expansion injection patterns
	if err := detectShellInjectionArtifacts(input, config); err != nil {
		return err
	}

	if err := validateEmpty(input, config); err != nil {
		return err
	}

	if !config.AllowControlChars {
		if err := validateControlCharacters(input, config); err != nil {
			return err
		}
	}

	if !config.AllowPathTraversal {
		if err := validatePathTraversal(input, config); err != nil {
			return err
		}
	}

	if !config.AllowShellMetachars {
		if err := validateShellMetacharacters(input, config); err != nil {
			return err
		}
	}

	return nil
}

func validateEmpty(input string, config ValidationConfig) error {
	if !config.AllowEmpty && strings.TrimSpace(input) == "" {
		return fmt.Errorf("%s cannot be empty", config.EntityType)
	}
	return nil
}

func validateControlCharacters(input string, config ValidationConfig) error {
	allowedMap := make(map[rune]bool)
	for _, r := range config.AllowedControlChars {
		allowedMap[r] = true
	}

	for _, r := range input {
		isControlChar := r < 32 || r == 127
		if isControlChar && !allowedMap[r] {
			return fmt.Errorf("%s cannot contain control characters", config.EntityType)
		}
	}
	return nil
}

func validatePathTraversal(input string, config ValidationConfig) error {
	if strings.Contains(input, "..") || strings.Contains(input, "/") || strings.Contains(input, "\\") {
		return fmt.Errorf("%s cannot contain path separators or path traversal sequences", config.EntityType)
	}
	return nil
}

func validateShellMetacharacters(input string, config ValidationConfig) error {
	// Shell metacharacters that can be used for command injection
	shellMetachars := []string{
		"$(", // Command substitution
		"`",  // Backtick command substitution
		";",  // Command separator
		"|",  // Pipe
		"&",  // Background process/AND
		">",  // Redirection
		"<",  // Redirection
		"*",  // Globbing
		"?",  // Globbing
		"[",  // Globbing
		"]",  // Globbing
		"{",  // Brace expansion
		"}",  // Brace expansion
		"~",  // Home directory expansion
		"!",  // History expansion (bash)
		"#",  // Comments (when at start or after space)
	}

	for _, char := range shellMetachars {
		if strings.Contains(input, char) {
			return fmt.Errorf("%s cannot contain shell metacharacters (found: %q)", config.EntityType, char)
		}
	}

	// Special check for $ followed by any character (not just $( )
	// This catches both $(cmd) and ${var} and $var patterns
	if strings.Contains(input, "$") {
		return fmt.Errorf("%s cannot contain shell metacharacters (found: \"$\")", config.EntityType)
	}

	return nil
}

// detectShellInjectionArtifacts detects patterns that suggest shell command injection
// may have already occurred before the application received the input
func detectShellInjectionArtifacts(input string, config ValidationConfig) error {
	if config.AllowShellMetachars {
		return nil // Skip detection if shell metacharacters are explicitly allowed
	}

	// Only check for obvious command injection artifacts, not legitimate path characters
	// Focus on detecting output patterns rather than input patterns
	suspiciousPatterns := []struct {
		pattern string
		message string
	}{
		// Common error messages that suggest injection
		{"Permission denied", "potential shell command injection detected (Permission denied output)"},
		{"command not found", "potential shell command injection detected (command not found output)"},
		{"rm: it is dangerous", "potential shell command injection detected (rm command output)"},
		{"cannot remove", "potential shell command injection detected (rm/file operation output)"},
		
		// Detect echo command output patterns (common in injection)
		{"-INJECTED", "potential shell command injection detected (echo command output pattern)"},
		{"-SAFE", "potential shell command injection detected (echo command output pattern)"},
		{"echo:", "potential shell command injection detected (echo command error output)"},
		
		// Detect other obvious command outputs
		{"cannot access", "potential shell command injection detected (file access error)"},
		{"Text file busy", "potential shell command injection detected (file busy error)"},
		
		// Common command outputs
		{"drwx", "potential shell command injection detected (ls -l output pattern)"},
		{"total ", "potential shell command injection detected (ls total output)"},
	}

	for _, suspicious := range suspiciousPatterns {
		if strings.Contains(input, suspicious.pattern) {
			return fmt.Errorf("security warning: %s. Use single quotes to prevent shell expansion", suspicious.message)
		}
	}

	// Detect null byte truncation warning patterns
	if strings.Contains(input, "null byte truncation") {
		return fmt.Errorf("security warning: input appears to contain shell injection artifacts")
	}

	return nil
}

// Predefined validation configurations for common use cases

// UsernameValidationConfig provides secure validation for usernames
var UsernameValidationConfig = ValidationConfig{
	EntityType:          "username",
	AllowEmpty:          false,
	AllowControlChars:   false,
	AllowedControlChars: nil,
	AllowPathTraversal:  false,
	AllowShellMetachars: false, // Prevent command injection
}

// SecretKeyValidationConfig provides secure validation for secret keys
var SecretKeyValidationConfig = ValidationConfig{
	EntityType:          "key name",
	AllowEmpty:          false,
	AllowControlChars:   false,
	AllowedControlChars: []rune{0x09, 0x0A, 0x0D}, // tab, LF, CR
	AllowPathTraversal:  false,
	AllowShellMetachars: false, // Prevent command injection
}
