package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestConsolidatedListCommands(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = append(os.Environ(), "HOME="+tmp)
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
			contains: "", // Empty list is fine for new store
		},
		{
			name:     "list backups",
			args:     []string{"list", "backups"},
			wantErr:  false,
			contains: "(no rotation backups available)",
		},
		{
			name:     "list users",
			args:     []string{"list", "users"},
			wantErr:  false,
			contains: "admin",
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
			cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
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
			cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	cmd = exec.Command(cliBin, "put", "backup-test", "modified-value")
	cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("put modified failed: %v", err)
	}

	// Create a rotation backup
	cmd = exec.Command(cliBin, "rotate", "master-key", "--yes")
	cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
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
			cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
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
