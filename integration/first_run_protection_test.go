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
	"path/filepath"
	"strings"
	"testing"

	"simple-secrets/integration/testing_framework"
)

func TestFirstRunProtectionIntegration(t *testing.T) {
	t.Run("BlocksWhenMasterKeyExists", func(t *testing.T) {
		env := testing_framework.NewEnvironment(t)
		defer env.Cleanup()

		// Setup: create only master.key to simulate partial installation
		masterKeyPath := filepath.Join(env.ConfigDir(), "master.key")
		err := os.WriteFile(masterKeyPath, []byte("fake-master-key"), 0600)
		if err != nil {
			t.Fatalf("failed to create master.key: %v", err)
		}

		// Try to run a command that would trigger first-run with protection enabled
		output, err := env.CLI().RawWithFirstRunProtection("list", "keys")

		// Should fail with master key corruption error
		if err == nil {
			t.Fatalf("expected command to fail when master.key exists but users.json missing")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "failed to decode master key") {
			t.Fatalf("expected master key decode error message, got: %s", outputStr)
		}
	})

	t.Run("BlocksWhenSecretsJsonExists", func(t *testing.T) {
		env := testing_framework.NewEnvironment(t)
		defer env.Cleanup()

		// Setup: create only secrets.json to simulate partial installation
		secretsPath := filepath.Join(env.ConfigDir(), "secrets.json")
		err := os.WriteFile(secretsPath, []byte(`{}`), 0600)
		if err != nil {
			t.Fatalf("failed to create secrets.json: %v", err)
		}

		// Try to run a command that would trigger first-run with protection enabled
		output, err := env.CLI().RawWithFirstRunProtection("list", "keys")

		// Should fail with database corruption error
		if err == nil {
			t.Fatalf("expected command to fail when secrets.json exists but users.json missing")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "authentication required: no token found") {
			t.Fatalf("expected authentication error due to missing users.json, got: %s", outputStr)
		}
	})

	t.Run("AllowsCleanFirstRun", func(t *testing.T) {
		// This test needs a truly clean environment without any initialization
		tempDir := t.TempDir()

		// Create a minimal test environment manually to avoid auto-initialization
		env := &testing_framework.TestEnvironment{}
		env.SetupCleanEnvironment(t, tempDir, "../simple-secrets")
		defer env.Cleanup()

		// Try to run a command that should trigger clean first-run
		output, _ := env.CLI().RawWithFirstRunProtection("list", "keys")

		// Should show first-run setup message
		outputStr := string(output)
		if !strings.Contains(outputStr, "master key not found - run setup first") {
			t.Fatalf("expected setup first message, got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "Usage:") {
			t.Fatalf("expected usage information after error, got: %s", outputStr)
		}
	})

	t.Run("BlocksWithMultipleCriticalFiles", func(t *testing.T) {
		env := testing_framework.NewEnvironment(t)
		defer env.Cleanup()

		// Create both master.key and secrets.json
		masterKeyPath := filepath.Join(env.ConfigDir(), "master.key")
		secretsPath := filepath.Join(env.ConfigDir(), "secrets.json")

		err := os.WriteFile(masterKeyPath, []byte("fake-master-key"), 0600)
		if err != nil {
			t.Fatalf("failed to create master.key: %v", err)
		}

		err = os.WriteFile(secretsPath, []byte(`{}`), 0600)
		if err != nil {
			t.Fatalf("failed to create secrets.json: %v", err)
		}

		// Try to run a command that would trigger first-run with protection enabled
		output, err := env.CLI().RawWithFirstRunProtection("create-user", "test", "admin")

		// Should fail with master key corruption error
		if err == nil {
			t.Fatalf("expected command to fail when critical files exist but users.json missing")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "failed to decode master key") {
			t.Fatalf("expected master key decode error message, got: %s", outputStr)
		}
	})
}
