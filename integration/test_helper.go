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

// TestHelper manages isolated test environments with automatic first-run initialization.
// It provides a shared binary approach for efficiency while ensuring each test runs in
// complete isolation with its own temp directory, clean environment, and admin token.
// This eliminates test interdependencies and makes tests both faster and more reliable.
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

	// Use explicit setup command (new behavior)
	cmd := exec.Command(h.binaryPath, "setup")
	cmd.Env = h.cleanEnv()

	// Provide "Y" as input to the first-run prompt
	cmd.Stdin = strings.NewReader("Y\n")

	output, err := cmd.CombinedOutput()
	if err != nil {
		h.t.Fatalf("first run failed: %v\n%s", err, output)
	}

	// Extract token from output
	h.token = extractTokenFromOutput(string(output))
	if h.token == "" {
		h.t.Fatalf("could not extract admin token from first-run output: %s", output)
	}
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
