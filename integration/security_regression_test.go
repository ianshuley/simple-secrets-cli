package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmptyTokenBypassVulnerability tests the specific security issue where --token "" was accepted
func TestEmptyTokenBypassVulnerability(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "explicit_empty_token_list",
			args:    []string{"--token", "", "list", "keys"},
			wantErr: true,
			errMsg:  "authentication required: token cannot be empty",
		},
		{
			name:    "explicit_empty_token_put",
			args:    []string{"--token", "", "put", "key", "value"},
			wantErr: true,
			errMsg:  "authentication required: token cannot be empty",
		},
		{
			name:    "explicit_empty_token_delete",
			args:    []string{"--token", "", "delete", "key"},
			wantErr: true,
			errMsg:  "authentication required: token cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = append(os.Environ(), "HOME="+tmp)
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded. Output: %s", out)
					return
				}
				if !strings.Contains(string(out), tt.errMsg) {
					t.Errorf("expected error message to contain %q, got: %s", tt.errMsg, out)
				}
			}
		})
	}
}

// TestDirectoryPermissionsVulnerability tests that directories are created with 700 permissions
func TestDirectoryPermissionsVulnerability(t *testing.T) {
	tmp := t.TempDir()

	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	configDir := filepath.Join(tmp, ".simple-secrets")
	stat, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}

	perm := stat.Mode().Perm()
	expected := os.FileMode(0700)
	if perm != expected {
		t.Errorf("config directory has insecure permissions. Expected %s, got %s", expected, perm)
	}
}
