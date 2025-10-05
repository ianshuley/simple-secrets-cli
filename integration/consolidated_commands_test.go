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

// testEnv returns a clean environment for the given temp directory
func testEnv(tmp string) []string {
	return append(os.Environ(),
		"HOME="+tmp,
		"SIMPLE_SECRETS_CONFIG_DIR="+tmp+"/.simple-secrets",
		"SIMPLE_SECRETS_TEST=1", // Disable first-run protection in tests
		"SIMPLE_SECRETS_TOKEN=", // Clear any existing token
	)
}

func TestConsolidatedListCommands(t *testing.T) {
	// Create isolated test helper
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains string
	}{
		{
			name:     "list keys",
			args:     []string{"list", "keys"},
			wantErr:  false,
			contains: "(no secrets)",
		},
		{
			name:     "list backups",
			args:     []string{"list", "backups"},
			wantErr:  false,
			contains: "(no rotation backups available)",
		},
		{
			name:    "list users",
			args:    []string{"list", "users"},
			wantErr: false,
		},
		{
			name:    "list invalid",
			args:    []string{"list", "invalid"},
			wantErr: true,
		},
		{
			name:    "list no args",
			args:    []string{"list"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := helper.RunCommand(tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded: %s", out)
				}
				return
			}

			if err != nil {
				t.Errorf("command failed: %v\n%s", err, out)
				return
			}

			if tt.contains != "" && !strings.Contains(string(out), tt.contains) {
				t.Errorf("output should contain %q but got: %s", tt.contains, out)
			}
		})
	}
}

func TestConsolidatedRotateCommands(t *testing.T) {
	// Create isolated test helper
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Add a secret first so rotation has something to work with
	_, err := helper.RunCommand("put", "test-key", "test-value")
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		stdin   string
	}{
		{
			name:    "rotate master-key with confirmation",
			args:    []string{"rotate", "master-key"},
			wantErr: false,
			stdin:   "yes\n",
		},
		{
			name:    "rotate master-key with --yes flag",
			args:    []string{"rotate", "master-key", "--yes"},
			wantErr: false,
		},
		{
			name:    "rotate master-key abort",
			args:    []string{"rotate", "master-key"},
			wantErr: false,
			stdin:   "no\n",
		},
		{
			name:    "rotate token admin",
			args:    []string{"rotate", "token", "admin"},
			wantErr: false,
		},
		{
			name:    "rotate token non-existent user",
			args:    []string{"rotate", "token", "nonexistent"},
			wantErr: true,
		},
		{
			name:    "rotate invalid type",
			args:    []string{"rotate", "invalid"},
			wantErr: true,
		},
		{
			name:    "rotate no args",
			args:    []string{"rotate"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(helper.GetBinaryPath(), tt.args...)
			cmd.Env = append(helper.cleanEnv(), "SIMPLE_SECRETS_TOKEN="+helper.GetToken())
			if tt.stdin != "" {
				cmd.Stdin = strings.NewReader(tt.stdin)
			}
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded: %s", out)
				}
				return
			}

			if err != nil {
				t.Errorf("command failed: %v\n%s", err, out)
			}
		})
	}
}

func TestConsolidatedRestoreCommands(t *testing.T) {
	// Create isolated test helper
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Add and delete a secret to create backup
	_, err := helper.RunCommand("put", "backup-test", "original-value")
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	_, err = helper.RunCommand("put", "backup-test", "modified-value")
	if err != nil {
		t.Fatalf("put modified failed: %v", err)
	}

	// Create a rotation backup
	_, err = helper.RunCommand("rotate", "master-key", "--yes")
	if err != nil {
		t.Fatalf("rotate failed: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		stdin   string
	}{
		{
			name:    "restore secret",
			args:    []string{"restore", "secret", "backup-test"},
			wantErr: false,
		},
		{
			name:    "restore secret non-existent",
			args:    []string{"restore", "secret", "nonexistent"},
			wantErr: true,
		},
		{
			name:    "restore database abort",
			args:    []string{"restore", "database"},
			wantErr: true,
		},
		{
			name:    "restore invalid type",
			args:    []string{"restore", "invalid"},
			wantErr: true,
		},
		{
			name:    "restore no args",
			args:    []string{"restore"},
			wantErr: true,
		},
		{
			name:    "restore secret no key",
			args:    []string{"restore", "secret"},
			wantErr: true,
		},
		{
			name:    "restore database no backup",
			args:    []string{"restore", "database"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For tests that need stdin, use exec.Command with helper's token and env
			cmd := exec.Command(helper.GetBinaryPath(), tt.args...)
			cmd.Env = append([]string{
				"HOME=" + helper.GetTempDir(),
				"SIMPLE_SECRETS_CONFIG_DIR=" + filepath.Join(helper.GetTempDir(), ".simple-secrets"),
				"PATH=" + os.Getenv("PATH"),
			}, "SIMPLE_SECRETS_TOKEN="+helper.GetToken())
			if tt.stdin != "" {
				cmd.Stdin = strings.NewReader(tt.stdin)
			}
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded: %s", out)
				}
				return
			}

			if err != nil {
				t.Errorf("command failed: %v\n%s", err, out)
			}
		})
	}
}

func TestLegacyCommandsStillWork(t *testing.T) {
	// Create isolated test helper
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Test that legacy restore-database command still works
	out, err := helper.RunCommand("restore-database", "--help")
	if err != nil {
		t.Errorf("legacy restore-database command should still work: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "restore-database") {
		t.Errorf("legacy restore-database help should mention restore-database: %s", out)
	}
}

func TestConsolidatedCommandHelpText(t *testing.T) {
	// Create isolated test helper (help commands don't need authentication)
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name: "list help",
			args: []string{"list", "--help"},
			contains: []string{
				"List different types of data",
				"keys", "backups", "users",
				"simple-secrets list keys",
			},
		},
		{
			name: "rotate help",
			args: []string{"rotate", "--help"},
			contains: []string{
				"Rotate different types of keys",
				"master-key", "token",
				"simple-secrets rotate master-key",
			},
		},
		{
			name: "restore help",
			args: []string{"restore", "--help"},
			contains: []string{
				"Restore different types of data",
				"secret", "database",
				"simple-secrets restore secret",
			},
		},
		{
			name: "disable help",
			args: []string{"disable", "--help"},
			contains: []string{
				"Disable different types of resources",
				"user <username>", "secret <key>",
				"simple-secrets disable user alice",
				"simple-secrets disable secret api-key",
			},
		},
		{
			name: "enable help",
			args: []string{"enable", "--help"},
			contains: []string{
				"Re-enable resources that were previously disabled",
				"secret <key>",
				"simple-secrets enable secret api-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Help commands don't need tokens
			out, err := helper.RunCommandWithoutToken(tt.args...)
			if err != nil {
				t.Errorf("help command failed: %v\n%s", err, out)
				return
			}

			for _, contain := range tt.contains {
				if !strings.Contains(string(out), contain) {
					t.Errorf("help output should contain %q but got: %s", contain, out)
				}
			}
		})
	}
}

func TestConsolidatedDisableEnableCommands(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains string
	}{
		// Test secret disable/enable workflow
		{
			name:     "put test secret",
			args:     []string{"put", "test-key", "test-value"},
			wantErr:  false,
			contains: `Secret "test-key" stored`,
		},
		{
			name:     "disable secret",
			args:     []string{"disable", "secret", "test-key"},
			wantErr:  false,
			contains: "Secret 'test-key' has been disabled",
		},
		{
			name:     "list keys excludes disabled",
			args:     []string{"list", "keys"},
			wantErr:  false,
			contains: "", // Should not contain test-key
		},
		{
			name:     "list disabled shows secret",
			args:     []string{"list", "disabled"},
			wantErr:  false,
			contains: "test-key",
		},
		{
			name:     "get disabled secret fails",
			args:     []string{"get", "test-key"},
			wantErr:  true,
			contains: "Secret is disabled",
		},
		{
			name:     "enable secret",
			args:     []string{"enable", "secret", "test-key"},
			wantErr:  false,
			contains: "Secret 'test-key' has been re-enabled",
		},
		{
			name:     "list keys includes enabled secret",
			args:     []string{"list", "keys"},
			wantErr:  false,
			contains: "test-key",
		},
		{
			name:     "get enabled secret works",
			args:     []string{"get", "test-key"},
			wantErr:  false,
			contains: "test-value",
		},
		// Test token disable workflow
		{
			name:     "create test user",
			args:     []string{"create-user", "testuser", "reader"},
			wantErr:  false,
			contains: "Generated token:",
		},
		{
			name:     "disable token invalid type",
			args:     []string{"disable", "invalid", "testuser"},
			wantErr:  true,
			contains: "unknown disable type",
		},
		{
			name:     "enable invalid type",
			args:     []string{"enable", "invalid", "test-key"},
			wantErr:  true,
			contains: "unknown enable type",
		},
		// Error cases
		{
			name:     "disable nonexistent secret",
			args:     []string{"disable", "secret", "nonexistent"},
			wantErr:  true,
			contains: "secret not found",
		},
		{
			name:     "enable nonexistent secret",
			args:     []string{"enable", "secret", "nonexistent"},
			wantErr:  true,
			contains: "secret not found",
		},
	}

	// Run tests sequentially to maintain state
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := helper.RunCommand(tt.args...)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but command succeeded: %s", output)
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v\n%s", err, output)
				return
			}

			if tt.contains != "" && !strings.Contains(string(output), tt.contains) {
				t.Errorf("output should contain %q but got: %s", tt.contains, string(output))
			}
		})
	}
}
