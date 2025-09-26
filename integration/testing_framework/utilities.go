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
package testing_framework

import (
	"strings"
)

// ParseToken extracts authentication tokens from command output.
// It handles multiple output formats from different commands (setup, list keys, etc.)
// and returns the first valid token found.
func ParseToken(output string) string {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Handle setup command format: "TOKEN: <token>"
		if strings.Contains(line, "TOKEN: ") {
			parts := strings.Split(line, "TOKEN: ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}

		// Handle legacy first-run format: "Admin token: <token>"
		if strings.Contains(line, "Admin token: ") {
			parts := strings.Split(line, "Admin token: ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}

		// Handle user creation format: "Generated token: <token>"
		if strings.Contains(line, "Generated token: ") {
			parts := strings.Split(line, "Generated token: ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}

		// Handle token creation format: "Token: <token>"
		if strings.Contains(line, "Token: ") && !strings.Contains(line, "Admin Token: ") {
			parts := strings.Split(line, "Token: ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// AssertContains checks if output contains expected text and provides helpful error messages
func AssertContains(output, expected string) bool {
	return strings.Contains(output, expected)
}

// AssertNotContains checks if output does not contain text
func AssertNotContains(output, forbidden string) bool {
	return !strings.Contains(output, forbidden)
}

// AssertTokenFormat validates that a string looks like a valid token
func AssertTokenFormat(token string) bool {
	// Simple validation - tokens should be non-empty and reasonable length
	return len(token) > 10 && len(token) < 200 && !strings.ContainsAny(token, "\n\r\t")
}

// CleanOutput removes common formatting and whitespace for easier testing
func CleanOutput(output string) string {
	return strings.TrimSpace(output)
}

// SplitLines splits output into clean lines, removing empty lines
func SplitLines(output string) []string {
	lines := strings.Split(output, "\n")
	var cleanLines []string

	for _, line := range lines {
		if cleaned := strings.TrimSpace(line); cleaned != "" {
			cleanLines = append(cleanLines, cleaned)
		}
	}

	return cleanLines
}

// FindLineContaining finds the first line that contains the given text
func FindLineContaining(output, text string) string {
	lines := SplitLines(output)

	for _, line := range lines {
		if strings.Contains(line, text) {
			return line
		}
	}

	return ""
}

// ExtractValue extracts a value after a label (e.g., "Username: alice" -> "alice")
func ExtractValue(line, label string) string {
	if !strings.Contains(line, label) {
		return ""
	}

	parts := strings.SplitN(line, label, 2)
	if len(parts) != 2 {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// ParseTokenFromCreateUser extracts token from create-user command output
func ParseTokenFromCreateUser(output string) string {
	// First try the standard ParseToken function which handles multiple formats
	token := ParseToken(output)
	if token != "" {
		return token
	}

	// Fallback for specific create-user format variations
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "New token: ") {
			parts := strings.Split(line, "New token: ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// ParseTokenFromSelfRotation extracts token from self-rotation command output
func ParseTokenFromSelfRotation(output string) string {
	// Try the standard ParseToken function first
	token := ParseToken(output)
	if token != "" {
		return token
	}

	// Look for rotation-specific patterns
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "New token: ") {
			parts := strings.Split(line, "New token: ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
		if strings.Contains(line, "Rotated token: ") {
			parts := strings.Split(line, "Rotated token: ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}
