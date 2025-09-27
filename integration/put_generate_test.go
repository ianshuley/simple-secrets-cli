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
)

func TestPutGenerateFlag(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	cli := env.CLI()

	// Test basic generation
	t.Run("basic_generate", func(t *testing.T) {
		output, err := cli.Put("test-key", "--generate")
		if err != nil {
			t.Fatalf("put with --generate failed: %v", err)
		}

		// Should contain confirmation message
		if !strings.Contains(string(output), `Secret "test-key" stored.`) {
			t.Errorf("Expected storage confirmation, got: %s", output)
		}

		// Should contain the generated secret on second line
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines of output, got %d: %v", len(lines), lines)
		}

		generatedSecret := lines[1]
		if len(generatedSecret) != 32 {
			t.Errorf("Expected 32 character secret, got %d: %s", len(generatedSecret), generatedSecret)
		}

		// Verify the secret was actually stored by retrieving it
		retrievedOutput, err := cli.Get("test-key")
		if err != nil {
			t.Fatalf("failed to retrieve generated secret: %v", err)
		}

		retrievedSecret := strings.TrimSpace(string(retrievedOutput))
		if retrievedSecret != generatedSecret {
			t.Errorf("Retrieved secret %q doesn't match generated secret %q", retrievedSecret, generatedSecret)
		}
	})

	// Test short flag variant
	t.Run("short_flag", func(t *testing.T) {
		output, err := cli.Put("test-key-short", "-g")
		if err != nil {
			t.Fatalf("put with -g failed: %v", err)
		}

		if !strings.Contains(string(output), `Secret "test-key-short" stored.`) {
			t.Errorf("Expected storage confirmation, got: %s", output)
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines of output, got %d: %v", len(lines), lines)
		}

		generatedSecret := lines[1]
		if len(generatedSecret) != 32 {
			t.Errorf("Expected 32 character secret, got %d: %s", len(generatedSecret), generatedSecret)
		}
	})

	// Test custom length
	t.Run("custom_length", func(t *testing.T) {
		output, err := cli.Put("test-key-64", "--generate", "--length", "64")
		if err != nil {
			t.Fatalf("put with --generate --length failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines of output, got %d: %v", len(lines), lines)
		}

		generatedSecret := lines[1]
		if len(generatedSecret) != 64 {
			t.Errorf("Expected 64 character secret, got %d: %s", len(generatedSecret), generatedSecret)
		}
	})

	// Test short flags combined (-g -l)
	t.Run("short_flags_combined", func(t *testing.T) {
		output, err := cli.Put("test-key-short-length", "-g", "-l", "48")
		if err != nil {
			t.Fatalf("put with -g -l failed: %v", err)
		}

		if !strings.Contains(string(output), `Secret "test-key-short-length" stored.`) {
			t.Errorf("Expected storage confirmation, got: %s", output)
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines of output, got %d: %v", len(lines), lines)
		}

		generatedSecret := lines[1]
		if len(generatedSecret) != 48 {
			t.Errorf("Expected 48 character secret, got %d: %s", len(generatedSecret), generatedSecret)
		}
	})

	// Test character set compliance
	t.Run("character_set", func(t *testing.T) {
		output, err := cli.Put("test-charset", "--generate")
		if err != nil {
			t.Fatalf("put with --generate failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		generatedSecret := lines[1]

		// Expected character set: A-Z, a-z, 0-9, !@#$%^&*()-_=+
		expectedCharset := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+"

		for _, char := range generatedSecret {
			if !strings.ContainsRune(expectedCharset, char) {
				t.Errorf("Generated secret contains invalid character: %c", char)
			}
		}
	})

	// Test uniqueness
	t.Run("uniqueness", func(t *testing.T) {
		secrets := make([]string, 5)

		for i := 0; i < 5; i++ {
			keyName := "test-unique-" + string(rune('a'+i))
			output, err := cli.Put(keyName, "--generate")
			if err != nil {
				t.Fatalf("put with --generate failed on iteration %d: %v", i, err)
			}

			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			secrets[i] = lines[1]
		}

		// Check all secrets are unique
		for i := 0; i < len(secrets); i++ {
			for j := i + 1; j < len(secrets); j++ {
				if secrets[i] == secrets[j] {
					t.Errorf("Generated duplicate secrets: %q", secrets[i])
				}
			}
		}
	})
}

func TestPutGenerateErrors(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	cli := env.CLI()

	// Test error: both generate and manual value
	t.Run("generate_with_manual_value", func(t *testing.T) {
		output, err := cli.Put("test-key", "manual-value", "--generate")
		if err == nil {
			t.Fatal("Expected error when providing both --generate and manual value")
		}

		// The error message is in the output, not the error object for CLI commands
		outputStr := string(output)
		if !strings.Contains(outputStr, "cannot provide both --generate flag and manual value") {
			t.Errorf("Expected specific error message, got: %s", outputStr)
		}
	})

	// Test error: generate without key
	t.Run("generate_without_key", func(t *testing.T) {
		output, err := cli.PutRaw("--generate")
		if err == nil {
			t.Fatal("Expected error when using --generate without key")
		}

		errorMsg := string(output)
		if !strings.Contains(errorMsg, "requires key argument when using --generate flag") {
			t.Errorf("Expected specific error message, got: %s", errorMsg)
		}
	})

	// Test that normal put still works
	t.Run("normal_put_still_works", func(t *testing.T) {
		_, err := cli.Put("normal-key", "normal-value")
		if err != nil {
			t.Fatalf("Normal put failed: %v", err)
		}

		output, err := cli.Get("normal-key")
		if err != nil {
			t.Fatalf("Failed to get normal secret: %v", err)
		}

		if strings.TrimSpace(string(output)) != "normal-value" {
			t.Errorf("Expected 'normal-value', got: %s", output)
		}
	})
}
