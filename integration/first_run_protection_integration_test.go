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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// firstRunTestEnv returns environment for testing first-run protection (without test mode disabled)
func firstRunTestEnv(tmp string) []string {
	return append(os.Environ(),
		"HOME="+tmp,
		"SIMPLE_SECRETS_CONFIG_DIR="+tmp+"/.simple-secrets",
		// Note: No SIMPLE_SECRETS_TEST=1 so first-run protection is active
	)
}

func TestFirstRunProtectionIntegration(t *testing.T) {
	// Create isolated test directory
	testDir := t.TempDir()
	configDir := filepath.Join(testDir, ".simple-secrets")
	os.MkdirAll(configDir, 0700)

	// Build the binary
	binaryPath := buildBinary(t)

	t.Run("BlocksWhenMasterKeyExists", func(t *testing.T) {
		// Setup: create only master.key to simulate partial installation
		masterKeyPath := filepath.Join(configDir, "master.key")
		os.WriteFile(masterKeyPath, []byte("fake-master-key"), 0600)
		defer os.Remove(masterKeyPath)

		// Try to run a command that would trigger first-run
		cmd := exec.Command(binaryPath, "list", "keys")
		cmd.Env = firstRunTestEnv(testDir)
		output, err := cmd.CombinedOutput()

		// Should fail with protection error
		if err == nil {
			t.Fatalf("expected command to fail when master.key exists but users.json missing")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "existing simple-secrets installation detected") {
			t.Fatalf("expected protection error message, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "master.key") {
			t.Fatalf("expected error to mention master.key, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "restore it from backup") {
			t.Fatalf("expected recovery guidance, got: %s", outputStr)
		}
	})

	t.Run("BlocksWhenSecretsJsonExists", func(t *testing.T) {
		// Clean up any files from previous tests
		os.RemoveAll(configDir)
		os.MkdirAll(configDir, 0700)

		// Setup: create only secrets.json to simulate partial installation
		secretsPath := filepath.Join(configDir, "secrets.json")
		os.WriteFile(secretsPath, []byte(`{"secrets":[]}`), 0600)
		defer os.Remove(secretsPath)

		// Try to run a command that would trigger first-run
		cmd := exec.Command(binaryPath, "list", "keys")
		cmd.Env = firstRunTestEnv(testDir)
		output, err := cmd.CombinedOutput()

		// Should fail with protection error
		if err == nil {
			t.Fatalf("expected command to fail when secrets.json exists but users.json missing")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "existing simple-secrets installation detected") {
			t.Fatalf("expected protection error message, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "secrets.json") {
			t.Fatalf("expected error to mention secrets.json, got: %s", outputStr)
		}
	})

	t.Run("AllowsCleanFirstRun", func(t *testing.T) {
		// Clean up any existing files
		os.RemoveAll(configDir)

		// Try to run a command that should trigger clean first-run
		cmd := exec.Command(binaryPath, "list", "keys")
		cmd.Env = firstRunTestEnv(testDir)
		output, _ := cmd.CombinedOutput()

		// Should succeed with first-run message (though it will exit 1 due to no token)
		outputStr := string(output)
		if !strings.Contains(outputStr, "users.json not found – creating default admin user") {
			t.Fatalf("expected first-run creation message, got: %s", outputStr)
		}

		if !strings.Contains(outputStr, "Created default admin user") {
			t.Fatalf("expected admin creation confirmation, got: %s", outputStr)
		}

		// Verify files were created
		if _, err := os.Stat(filepath.Join(configDir, "users.json")); err != nil {
			t.Fatalf("users.json not created: %v", err)
		}

		if _, err := os.Stat(filepath.Join(configDir, "roles.json")); err != nil {
			t.Fatalf("roles.json not created: %v", err)
		}
	})

	t.Run("BlocksWithMultipleCriticalFiles", func(t *testing.T) {
		// Clean up first
		os.RemoveAll(configDir)
		os.MkdirAll(configDir, 0700)

		// Create both master.key and secrets.json
		masterKeyPath := filepath.Join(configDir, "master.key")
		secretsPath := filepath.Join(configDir, "secrets.json")
		os.WriteFile(masterKeyPath, []byte("fake-master-key"), 0600)
		os.WriteFile(secretsPath, []byte(`{"secrets":[]}`), 0600)

		// Try to run a command that would trigger first-run
		cmd := exec.Command(binaryPath, "create-user", "test", "admin")
		cmd.Env = firstRunTestEnv(testDir)
		output, err := cmd.CombinedOutput()

		// Should fail with protection error
		if err == nil {
			t.Fatalf("expected command to fail when critical files exist but users.json missing")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "existing simple-secrets installation detected") {
			t.Fatalf("expected protection error message, got: %s", outputStr)
		}

		// Should mention the first file found (master.key comes first in our check)
		if !strings.Contains(outputStr, "master.key") {
			t.Fatalf("expected error to mention master.key, got: %s", outputStr)
		}
	})
}

// buildBinary creates a test binary and returns its path
func buildBinary(t *testing.T) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "simple-secrets-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = ".." // Go up to project root

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	return binaryPath
}
