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
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

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
			cmd := exec.Command(cliBin, tt.args...)
			envWithToken := append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
			cmd.Env = envWithToken
			out, err := cmd.CombinedOutput()

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
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Add a secret first so rotation has something to work with
	cmd = exec.Command(cliBin, "put", "test-key", "test-value")
	envWithToken := append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
	cmd.Env = envWithToken
	_, err = cmd.CombinedOutput()
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
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
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
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Add and delete a secret to create backup
	cmd = exec.Command(cliBin, "put", "backup-test", "original-value")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	cmd = exec.Command(cliBin, "put", "backup-test", "modified-value")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("put modified failed: %v", err)
	}

	// Create a rotation backup
	cmd = exec.Command(cliBin, "rotate", "master-key", "--yes")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
	_, err = cmd.CombinedOutput()
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
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
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
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Test that legacy restore-database command still works
	cmd = exec.Command(cliBin, "restore-database", "--help")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("legacy restore-database command should still work: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "restore-database") {
		t.Errorf("legacy restore-database help should mention restore-database: %s", out)
	}
}

func TestConsolidatedCommandHelpText(t *testing.T) {
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
				"token <username>", "secret <key>",
				"simple-secrets disable token alice",
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
			cmd := exec.Command(cliBin, tt.args...)
			out, err := cmd.CombinedOutput()
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
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains string
	}{
		// Test secret disable/enable workflow
		{
			name:     "put test secret",
			args:     []string{"put", "test-key", "test-value", "--token", token},
			wantErr:  false,
			contains: `Secret "test-key" stored`,
		},
		{
			name:     "disable secret",
			args:     []string{"disable", "secret", "test-key", "--token", token},
			wantErr:  false,
			contains: "Secret 'test-key' has been disabled",
		},
		{
			name:     "list keys excludes disabled",
			args:     []string{"list", "keys", "--token", token},
			wantErr:  false,
			contains: "", // Should not contain test-key
		},
		{
			name:     "list disabled shows secret",
			args:     []string{"list", "disabled", "--token", token},
			wantErr:  false,
			contains: "test-key",
		},
		{
			name:     "get disabled secret fails",
			args:     []string{"get", "test-key", "--token", token},
			wantErr:  true,
			contains: "not found",
		},
		{
			name:     "enable secret",
			args:     []string{"enable", "secret", "test-key", "--token", token},
			wantErr:  false,
			contains: "Secret 'test-key' has been re-enabled",
		},
		{
			name:     "list keys includes enabled secret",
			args:     []string{"list", "keys", "--token", token},
			wantErr:  false,
			contains: "test-key",
		},
		{
			name:     "get enabled secret works",
			args:     []string{"get", "test-key", "--token", token},
			wantErr:  false,
			contains: "test-value",
		},
		// Test token disable workflow
		{
			name:     "create test user",
			args:     []string{"create-user", "testuser", "reader", "--token", token},
			wantErr:  false,
			contains: "Generated token:",
		},
		{
			name:     "disable token invalid type",
			args:     []string{"disable", "invalid", "testuser", "--token", token},
			wantErr:  true,
			contains: "unknown disable type",
		},
		{
			name:     "enable invalid type",
			args:     []string{"enable", "invalid", "test-key", "--token", token},
			wantErr:  true,
			contains: "unknown enable type",
		},
		// Error cases
		{
			name:     "disable nonexistent secret",
			args:     []string{"disable", "secret", "nonexistent", "--token", token},
			wantErr:  true,
			contains: "not found",
		},
		{
			name:     "enable nonexistent secret",
			args:     []string{"enable", "secret", "nonexistent", "--token", token},
			wantErr:  true,
			contains: "not found",
		},
		{
			name:     "disable without token",
			args:     []string{"disable", "secret", "test-key"},
			wantErr:  true,
			contains: "authentication required",
		},
		{
			name:     "enable without token",
			args:     []string{"enable", "secret", "test-key"},
			wantErr:  true,
			contains: "authentication required",
		},
	}

	// Run tests sequentially to maintain state
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = testEnv(tmp)
			out, err := cmd.CombinedOutput()

			if tt.wantErr && err == nil {
				t.Errorf("expected error but command succeeded: %s", out)
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v\n%s", err, out)
				return
			}

			if tt.contains != "" && !strings.Contains(string(out), tt.contains) {
				t.Errorf("output should contain %q but got: %s", tt.contains, out)
			}
		})
	}
}

