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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnvironment manages isolated test environments with automatic first-run initialization.
// It provides clean isolation between tests with separate temp directories, environments,
// and admin tokens. This eliminates test interdependencies and ensures reliable test execution.
type TestEnvironment struct {
	t          *testing.T
	tempDir    string
	binaryPath string
	adminToken string
	cli        *CLIRunner
}

// NewEnvironment creates a new isolated test environment with first-run setup complete.
// Each environment gets its own temp directory, clean environment variables,
// and a pre-initialized admin token for immediate use.
func NewEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Create isolated temp directory
	tempDir := t.TempDir()

	// Use the shared binary for efficiency
	binaryPath := "../simple-secrets"

	env := &TestEnvironment{
		t:          t,
		tempDir:    tempDir,
		binaryPath: binaryPath,
	}

	// Initialize with first-run and capture admin token
	env.initializeFirstRun()

	// Create CLI runner
	env.cli = &CLIRunner{
		env: env,
	}

	return env
}

// SetupCleanEnvironment creates a test environment without first-run initialization.
// This is useful for testing first-run scenarios where you need a truly clean environment.
func (e *TestEnvironment) SetupCleanEnvironment(t *testing.T, tempDir, binaryPath string) {
	t.Helper()

	e.t = t
	e.tempDir = tempDir
	e.binaryPath = binaryPath
	e.adminToken = "" // No token - clean environment

	// Create CLI runner
	e.cli = &CLIRunner{
		env: e,
	}
}

// CLI returns a type-safe command runner for this environment
func (e *TestEnvironment) CLI() *CLIRunner {
	return e.cli
}

// TempDir returns the isolated temporary directory path
func (e *TestEnvironment) TempDir() string {
	return e.tempDir
}

// AdminToken returns the admin authentication token
func (e *TestEnvironment) AdminToken() string {
	return e.adminToken
}

// BinaryPath returns the path to the CLI binary
func (e *TestEnvironment) BinaryPath() string {
	return e.binaryPath
}

// ConfigDir returns the configuration directory path for this environment
func (e *TestEnvironment) ConfigDir() string {
	return filepath.Join(e.tempDir, ".simple-secrets")
}

// CleanEnvironment returns environment variables for this isolated test
func (e *TestEnvironment) CleanEnvironment() []string {
	return []string{
		"HOME=" + e.tempDir,
		"SIMPLE_SECRETS_CONFIG_DIR=" + filepath.Join(e.tempDir, ".simple-secrets"),
		"PATH=" + os.Getenv("PATH"), // Keep PATH for finding go/git etc
		"SIMPLE_SECRETS_TEST=1",     // Disable first-run protection in tests
		// Note: SIMPLE_SECRETS_TOKEN is set separately by each method as needed
	}
}

// FirstRunProtectionEnvironment returns environment variables with test mode disabled
// This enables first-run protection for testing scenarios where we want to verify
// the protection mechanisms work correctly
func (e *TestEnvironment) FirstRunProtectionEnvironment() []string {
	return []string{
		"HOME=" + e.tempDir,
		"SIMPLE_SECRETS_CONFIG_DIR=" + filepath.Join(e.tempDir, ".simple-secrets"),
		"PATH=" + os.Getenv("PATH"), // Keep PATH for finding go/git etc
		// Note: SIMPLE_SECRETS_TEST is NOT set, enabling first-run protection
	}
}

// Cleanup performs any necessary cleanup (called via defer)
func (e *TestEnvironment) Cleanup() {
	// Test temp directories are automatically cleaned up by Go testing framework
	// This method exists for consistency and potential future cleanup needs
}

// initializeFirstRun performs first-run initialization and captures the admin token
func (e *TestEnvironment) initializeFirstRun() {
	e.t.Helper()

	// Run setup command with "Y" input to accept first-run initialization
	cmd := exec.Command(e.binaryPath, "setup")
	cmd.Env = e.CleanEnvironment()
	cmd.Stdin = strings.NewReader("Y\n")

	output, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("first-run setup failed: %v\n%s", err, output)
	}

	// Extract the admin token from setup output
	token := ParseToken(string(output))
	if token == "" {
		e.t.Fatalf("could not extract admin token from first-run output: %s", output)
	}

	e.adminToken = token
}

// RunRawCommand executes a command with custom arguments and environment
// This is the low-level interface for special cases that need custom control
func (e *TestEnvironment) RunRawCommand(args []string, env []string, stdin string) ([]byte, error) {
	e.t.Helper()

	cmd := exec.Command(e.binaryPath, args...)
	if env != nil {
		cmd.Env = env
	} else {
		cmd.Env = append(e.CleanEnvironment(), "SIMPLE_SECRETS_TOKEN="+e.adminToken)
	}

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	return cmd.CombinedOutput()
}

// NewCLIRunnerWithToken creates a CLI runner with a custom authentication token
func NewCLIRunnerWithToken(env *TestEnvironment, token string) *CLIRunner {
	return &CLIRunner{
		env:         env,
		customToken: token,
	}
}
