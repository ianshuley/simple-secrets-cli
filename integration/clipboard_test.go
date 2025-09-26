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

	"github.com/atotto/clipboard"
)

func TestGetWithClipboardAndSilentFlags(t *testing.T) {
	// Create isolated test environment with automatic first-run setup
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	cli := env.CLI()

	// Test data
	testKey := "test_secret"
	testValue := "super_secret_value"

	// Store a test secret
	output, err := cli.Put(testKey, testValue)
	testing_framework.Assert(t, output, err).Success()

	t.Run("get_normal_behavior", func(t *testing.T) {
		// Test normal get behavior (should print to stdout)
		output, err := cli.Get(testKey)
		testing_framework.Assert(t, output, err).Success()
		
		outputStr := strings.TrimSpace(string(output))
		if outputStr != testValue {
			t.Errorf("expected output %q, got %q", testValue, outputStr)
		}
	})

	t.Run("get_with_clipboard_only", func(t *testing.T) {
		// Clear clipboard first
		clipboard.WriteAll("")
		
		// Test get with clipboard flag (should copy to clipboard AND print to stdout)
		output, err := cli.GetWithClipboard(testKey)
		testing_framework.Assert(t, output, err).Success()
		
		// Check stdout output - should contain the secret value
		outputStr := strings.TrimSpace(string(output))
		if !strings.Contains(outputStr, testValue) {
			t.Errorf("expected output to contain %q, got %q", testValue, outputStr)
		}
		
		// In environments without clipboard, we expect a warning but success
		if strings.Contains(outputStr, "Warning: clipboard functionality not available") {
			t.Log("Clipboard not available in test environment - warning shown as expected")
		} else {
			// Check clipboard content if available
			clipboardContent, clipErr := clipboard.ReadAll()
			if clipErr != nil {
				t.Logf("Warning: Could not read clipboard (may not be available in test environment): %v", clipErr)
			} else if clipboardContent != testValue {
				t.Errorf("expected clipboard content %q, got %q", testValue, clipboardContent)
			}
		}
	})

	t.Run("get_with_silent_only", func(t *testing.T) {
		// Test get with silent flag (should suppress stdout, no clipboard)
		output, err := cli.GetSilent(testKey)
		testing_framework.Assert(t, output, err).Success()
		
		// Should have no output
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			t.Errorf("expected no output, got %q", outputStr)
		}
	})

	t.Run("get_with_clipboard_and_silent", func(t *testing.T) {
		// Clear clipboard first
		clipboard.WriteAll("")
		
		// Test get with both clipboard and silent flags (copy to clipboard, no stdout)
		output, err := cli.GetWithClipboardSilent(testKey)
		testing_framework.Assert(t, output, err).Success()
		
		// Should have no stdout output (silent mode)
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			t.Errorf("expected no output in silent mode, got %q", outputStr)
		}
		
		// Check clipboard content if available
		clipboardContent, clipErr := clipboard.ReadAll()
		if clipErr != nil {
			// This is expected in test environments without clipboard
			t.Logf("Clipboard not available in test environment: %v", clipErr)
		} else if clipboardContent != testValue {
			t.Errorf("expected clipboard content %q, got %q", testValue, clipboardContent)
		}
	})

	t.Run("get_nonexistent_secret_with_flags", func(t *testing.T) {
		// Test that error handling works with flags
		nonexistentKey := "nonexistent_secret"
		
		// Test with clipboard flag
		output, err := cli.GetWithClipboard(nonexistentKey)
		testing_framework.Assert(t, output, err).Failure()
		
		// Test with silent flag
		output, err = cli.GetSilent(nonexistentKey)
		testing_framework.Assert(t, output, err).Failure()
		
		// Test with both flags
		output, err = cli.GetWithClipboardSilent(nonexistentKey)
		testing_framework.Assert(t, output, err).Failure()
	})
}