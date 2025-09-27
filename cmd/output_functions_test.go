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
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout during function execution
func captureOutput(fn func()) string {
	// Save original stdout
	originalStdout := os.Stdout

	// Create pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Channel to capture the output
	outputChan := make(chan string)

	// Start goroutine to read from pipe
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outputChan <- buf.String()
	}()

	// Execute the function
	fn()

	// Close writer and restore stdout
	w.Close()
	os.Stdout = originalStdout

	// Get captured output
	output := <-outputChan
	r.Close()

	return output
}

func TestPrintFirstRunMessage(t *testing.T) {
	output := captureOutput(func() {
		PrintFirstRunMessage()
	})

	// Verify key components of the first run message
	expectedParts := []string{
		"First run detected. Default admin user created.",
		"To use your new token, re-run this command in one of these ways:",
		"--token <your-token> (as a flag)",
		"SIMPLE_SECRETS_TOKEN=<your-token>",
		"config.json",
		"chmod 600 ~/.simple-secrets/config.json",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("PrintFirstRunMessage() output missing expected part: %q", part)
		}
	}

	// Verify the output is not empty
	if strings.TrimSpace(output) == "" {
		t.Error("PrintFirstRunMessage() produced empty output")
	}

	// Verify output contains helpful instructions
	if !strings.Contains(output, "env var") || !strings.Contains(output, "secure permissions") {
		t.Error("PrintFirstRunMessage() missing key instructional content")
	}
}

func TestPrintTokenAtEnd(t *testing.T) {
	testToken := "test-token-12345"

	output := captureOutput(func() {
		PrintTokenAtEnd(testToken)
	})

	// Verify the token is displayed
	if !strings.Contains(output, testToken) {
		t.Errorf("PrintTokenAtEnd() output missing token: %q", testToken)
	}

	// Verify key components of the token display
	expectedParts := []string{
		"🔑 Your authentication token:",
		testToken,
		"📋 Please store this token securely",
		"password manager",
		"It will not be shown again!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("PrintTokenAtEnd() output missing expected part: %q", part)
		}
	}

	// Verify the output is not empty
	if strings.TrimSpace(output) == "" {
		t.Error("PrintTokenAtEnd() produced empty output")
	}

	// Verify security warning is present
	if !strings.Contains(output, "securely") {
		t.Error("PrintTokenAtEnd() missing security warning")
	}
}

func TestPrintTokenAtEndWithEmptyToken(t *testing.T) {
	emptyToken := ""

	output := captureOutput(func() {
		PrintTokenAtEnd(emptyToken)
	})

	// Even with empty token, function should still produce output
	if strings.TrimSpace(output) == "" {
		t.Error("PrintTokenAtEnd() with empty token produced no output")
	}

	// Should still contain the structure elements
	expectedParts := []string{
		"🔑 Your authentication token:",
		"📋 Please store this token securely",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("PrintTokenAtEnd() with empty token missing expected part: %q", part)
		}
	}
}

func TestPrintTokenAtEndWithSpecialCharacters(t *testing.T) {
	specialToken := "token-with-special-chars!@#$%^&*()"

	output := captureOutput(func() {
		PrintTokenAtEnd(specialToken)
	})

	// Verify the special token is displayed correctly
	if !strings.Contains(output, specialToken) {
		t.Errorf("PrintTokenAtEnd() output missing special token: %q", specialToken)
	}

	// Function should handle special characters without errors
	if strings.TrimSpace(output) == "" {
		t.Error("PrintTokenAtEnd() with special characters produced empty output")
	}
}

func TestPrintUserCreationSuccess(t *testing.T) {
	testUsername := "testuser"
	testToken := "test-token-67890"

	output := captureOutput(func() {
		printUserCreationSuccess(testUsername, testToken)
	})

	// Verify both username and token are displayed
	if !strings.Contains(output, testUsername) {
		t.Errorf("printUserCreationSuccess() output missing username: %q", testUsername)
	}

	if !strings.Contains(output, testToken) {
		t.Errorf("printUserCreationSuccess() output missing token: %q", testToken)
	}

	// Verify key message components
	expectedParts := []string{
		"User",
		"created",
		"Generated token:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("printUserCreationSuccess() output missing expected part: %q", part)
		}
	}

	// Verify the output is not empty
	if strings.TrimSpace(output) == "" {
		t.Error("printUserCreationSuccess() produced empty output")
	}
}

func TestPrintUserCreationSuccessWithSpecialUsername(t *testing.T) {
	specialUsername := "user-with-special@chars.com"
	testToken := "token123"

	output := captureOutput(func() {
		printUserCreationSuccess(specialUsername, testToken)
	})

	// Should handle special characters in username
	if !strings.Contains(output, specialUsername) {
		t.Errorf("printUserCreationSuccess() output missing special username: %q", specialUsername)
	}

	if !strings.Contains(output, testToken) {
		t.Errorf("printUserCreationSuccess() output missing token: %q", testToken)
	}

	// Should still produce structured output
	if !strings.Contains(output, "created") {
		t.Error("printUserCreationSuccess() with special username missing 'created' message")
	}
}

func TestPrintUserCreationSuccessWithEmptyValues(t *testing.T) {
	emptyUsername := ""
	emptyToken := ""

	output := captureOutput(func() {
		printUserCreationSuccess(emptyUsername, emptyToken)
	})

	// Even with empty values, function should produce some output
	if strings.TrimSpace(output) == "" {
		t.Error("printUserCreationSuccess() with empty values produced no output")
	}

	// Should still contain structure
	if !strings.Contains(output, "created") || !strings.Contains(output, "Generated token:") {
		t.Error("printUserCreationSuccess() with empty values missing expected structure")
	}
}
