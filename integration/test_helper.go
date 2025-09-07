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
	"testing"
)

// TestHelper provides utilities for integration tests
type TestHelper struct {
	t          *testing.T
	tempDir    string
	binaryPath string
	token      string
}

// NewTestHelper creates a new test helper with isolated environment
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	// Create isolated temp directory
	tempDir := t.TempDir()

	// Use the shared binary (much more efficient)
	binaryPath := "../simple-secrets"

	helper := &TestHelper{
		t:          t,
		tempDir:    tempDir,
		binaryPath: binaryPath,
	}

	// Initialize with first-run and capture token
	helper.initializeFirstRun()

	return helper
}

// No need for buildTestBinary - use the shared one!

// initializeFirstRun performs first-run initialization and captures the admin token
func (h *TestHelper) initializeFirstRun() {
	h.t.Helper()

	// Trigger first-run with a clean environment
	cmd := exec.Command(h.binaryPath, "list", "keys")
	cmd.Env = h.cleanEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		h.t.Fatalf("first-run initialization failed: %v\n%s", err, output)
	}

	// Extract token from output
	h.token = h.extractTokenFromOutput(string(output))
	if h.token == "" {
		h.t.Fatalf("could not extract admin token from first-run output: %s", output)
	}
}

// extractTokenFromOutput parses the admin token from first-run output
func (h *TestHelper) extractTokenFromOutput(output string) string {
	lines := splitLines(output)
	for _, line := range lines {
		if containsString(line, "Token:") {
			fields := splitFields(line)
			if len(fields) > 1 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// Helper functions to avoid importing strings package
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitFields(s string) []string {
	var fields []string
	var current []byte
	inField := false

	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			if inField {
				fields = append(fields, string(current))
				current = current[:0]
				inField = false
			}
		} else {
			current = append(current, s[i])
			inField = true
		}
	}

	if inField {
		fields = append(fields, string(current))
	}

	return fields
}

// cleanEnv returns a clean environment with only the test HOME directory
func (h *TestHelper) cleanEnv() []string {
	// Start with minimal environment
	env := []string{
		"HOME=" + h.tempDir,
		"SIMPLE_SECRETS_CONFIG_DIR=" + filepath.Join(h.tempDir, ".simple-secrets"),
		"PATH=" + os.Getenv("PATH"), // Keep PATH for finding go/git etc
	}

	// Add other essential env vars if needed
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		env = append(env, "GOPATH="+gopath)
	}
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		env = append(env, "GOROOT="+goroot)
	}

	return env
}

// RunCommand executes a command with the test environment and token
func (h *TestHelper) RunCommand(args ...string) ([]byte, error) {
	h.t.Helper()

	// Add token to args if not already present
	hasToken := false
	for _, arg := range args {
		if arg == "--token" {
			hasToken = true
			break
		}
	}

	if !hasToken {
		args = append(args, "--token", h.token)
	}

	cmd := exec.Command(h.binaryPath, args...)
	cmd.Env = h.cleanEnv()

	return cmd.CombinedOutput()
}

// RunCommandWithoutToken executes a command without adding authentication
func (h *TestHelper) RunCommandWithoutToken(args ...string) ([]byte, error) {
	h.t.Helper()

	cmd := exec.Command(h.binaryPath, args...)
	cmd.Env = h.cleanEnv()

	return cmd.CombinedOutput()
}

// GetToken returns the admin token for manual command construction
func (h *TestHelper) GetToken() string {
	return h.token
}

// GetTempDir returns the isolated temp directory path
func (h *TestHelper) GetTempDir() string {
	return h.tempDir
}

// GetBinaryPath returns the path to the test binary
func (h *TestHelper) GetBinaryPath() string {
	return h.binaryPath
}

// Cleanup performs any necessary cleanup (automatically called by t.TempDir())
func (h *TestHelper) Cleanup() {
	// TempDir cleanup is automatic, but we could add other cleanup here if needed
}
