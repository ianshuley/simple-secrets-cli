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

package rotation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"simple-secrets/pkg/rotation"
	"simple-secrets/pkg/secrets"
	"simple-secrets/pkg/users"
)

// mockSecretsStore is a test double for the secrets store
type mockSecretsStore struct{}

func (m *mockSecretsStore) Put(ctx context.Context, key, value string) error { return nil }
func (m *mockSecretsStore) Get(ctx context.Context, key string) (string, error) {
	return "test-value", nil
}
func (m *mockSecretsStore) Generate(ctx context.Context, key string, length int) (string, error) {
	return "generated-secret", nil
}
func (m *mockSecretsStore) Delete(ctx context.Context, key string) error { return nil }
func (m *mockSecretsStore) List(ctx context.Context) ([]secrets.SecretMetadata, error) {
	return []secrets.SecretMetadata{}, nil
}
func (m *mockSecretsStore) Enable(ctx context.Context, key string) error  { return nil }
func (m *mockSecretsStore) Disable(ctx context.Context, key string) error { return nil }
func (m *mockSecretsStore) ListDisabled(ctx context.Context) ([]secrets.SecretMetadata, error) {
	return []secrets.SecretMetadata{}, nil
}

// mockUsersStore is a test double for the users store
type mockUsersStore struct{}

func (m *mockUsersStore) Create(ctx context.Context, username, role string) (*users.User, string, error) {
	return &users.User{Username: username, Role: role}, "new-token", nil
}
func (m *mockUsersStore) GetByUsername(ctx context.Context, username string) (*users.User, error) {
	return &users.User{Username: username}, nil
}
func (m *mockUsersStore) GetByToken(ctx context.Context, token string) (*users.User, error) {
	return &users.User{Username: "testuser"}, nil
}
func (m *mockUsersStore) List(ctx context.Context) ([]*users.User, error) {
	return []*users.User{}, nil
}
func (m *mockUsersStore) Update(ctx context.Context, user *users.User) error {
	return nil
}
func (m *mockUsersStore) Delete(ctx context.Context, username string) error {
	return nil
}
func (m *mockUsersStore) Enable(ctx context.Context, username string) error {
	return nil
}
func (m *mockUsersStore) Disable(ctx context.Context, username string) error {
	return nil
}
func (m *mockUsersStore) RotateToken(ctx context.Context, username string) (string, error) {
	return "rotated-token", nil
}
func (m *mockUsersStore) AddToken(ctx context.Context, username, tokenName string) (*users.Token, string, error) {
	return &users.Token{Name: tokenName}, "new-token", nil
}
func (m *mockUsersStore) RevokeToken(ctx context.Context, username, tokenID string) error {
	return nil
}
func (m *mockUsersStore) ListTokens(ctx context.Context, username string) ([]*users.Token, error) {
	return []*users.Token{}, nil
}

// setupTempDir creates a temporary directory for testing
func setupTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "rotation_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

// createTestBackup creates a test backup directory with required files
func createTestBackup(t *testing.T, backupDir, name string) {
	backupPath := filepath.Join(backupDir, name)
	if err := os.MkdirAll(backupPath, 0700); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create required files
	keyFile := filepath.Join(backupPath, "master.key")
	secretsFile := filepath.Join(backupPath, "secrets.json")

	if err := os.WriteFile(keyFile, []byte("test-key"), 0600); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := os.WriteFile(secretsFile, []byte(`{"test":"value"}`), 0600); err != nil {
		t.Fatalf("Failed to create secrets file: %v", err)
	}
}

func TestNewService(t *testing.T) {
	secretsStore := &mockSecretsStore{}
	usersStore := &mockUsersStore{}
	dataDir := "/test/data"

	service := NewService(secretsStore, usersStore, dataDir)

	if service == nil {
		t.Fatalf("Expected service to be created, got nil")
	}

	// Test that it implements the interface
	var _ rotation.Service = service
}

func TestNewServiceWithConfig(t *testing.T) {
	secretsStore := &mockSecretsStore{}
	usersStore := &mockUsersStore{}
	dataDir := "/test/data"
	config := &rotation.RotationConfig{
		BackupRetentionCount: 5,
		AutoCleanup:          false,
		BackupDir:            "custom-backups",
	}

	service := NewServiceWithConfig(secretsStore, usersStore, dataDir, config)

	if service == nil {
		t.Fatalf("Expected service to be created, got nil")
	}

	// Test that it implements the interface
	var _ rotation.Service = service
}

func TestServiceImpl_CreateBackup(t *testing.T) {
	tempDir := setupTempDir(t)

	// Create test data files
	keyPath := filepath.Join(tempDir, "master.key")
	secretsPath := filepath.Join(tempDir, "secrets.json")
	usersPath := filepath.Join(tempDir, "users.json")

	if err := os.WriteFile(keyPath, []byte("test-key"), 0600); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := os.WriteFile(secretsPath, []byte(`{"test":"secret"}`), 0600); err != nil {
		t.Fatalf("Failed to create secrets file: %v", err)
	}
	if err := os.WriteFile(usersPath, []byte(`{"users":[]}`), 0600); err != nil {
		t.Fatalf("Failed to create users file: %v", err)
	}

	service := &ServiceImpl{
		secretsStore: &mockSecretsStore{},
		usersStore:   &mockUsersStore{},
		config:       rotation.DefaultRotationConfig(),
		dataDir:      tempDir,
	}

	backupDir := filepath.Join(tempDir, "test-backup")
	err := service.CreateBackup(context.Background(), backupDir)

	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup files were created
	backupKeyPath := filepath.Join(backupDir, "master.key")
	backupSecretsPath := filepath.Join(backupDir, "secrets.json")
	backupUsersPath := filepath.Join(backupDir, "users.json")

	if _, err := os.Stat(backupKeyPath); os.IsNotExist(err) {
		t.Errorf("Backup key file was not created")
	}
	if _, err := os.Stat(backupSecretsPath); os.IsNotExist(err) {
		t.Errorf("Backup secrets file was not created")
	}
	if _, err := os.Stat(backupUsersPath); os.IsNotExist(err) {
		t.Errorf("Backup users file was not created")
	}
}

func TestServiceImpl_ListBackups(t *testing.T) {
	tempDir := setupTempDir(t)
	backupDir := filepath.Join(tempDir, "backups")

	// Create test backups
	createTestBackup(t, backupDir, "rotate-20240901-143022")
	createTestBackup(t, backupDir, "manual-20240902-100000")
	createTestBackup(t, backupDir, "pre-restore-20240903-120000")

	service := &ServiceImpl{
		secretsStore: &mockSecretsStore{},
		usersStore:   &mockUsersStore{},
		config:       rotation.DefaultRotationConfig(),
		dataDir:      tempDir,
	}

	backups, err := service.ListBackups(context.Background())

	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(backups))
	}

	// Verify backups are sorted by timestamp (newest first)
	for i := 0; i < len(backups)-1; i++ {
		if backups[i].Timestamp.Before(backups[i+1].Timestamp) {
			t.Errorf("Backups are not sorted by timestamp (newest first)")
			break
		}
	}

	// Verify backup types are correctly identified
	found := make(map[rotation.BackupType]bool)
	for _, backup := range backups {
		found[backup.Type] = true
	}

	if !found[rotation.BackupTypeRotation] {
		t.Errorf("Expected to find rotation backup")
	}
	if !found[rotation.BackupTypeManual] {
		t.Errorf("Expected to find manual backup")
	}
	if !found[rotation.BackupTypePreRestore] {
		t.Errorf("Expected to find pre-restore backup")
	}
}

func TestServiceImpl_ListBackups_EmptyDirectory(t *testing.T) {
	tempDir := setupTempDir(t)

	service := &ServiceImpl{
		secretsStore: &mockSecretsStore{},
		usersStore:   &mockUsersStore{},
		config:       rotation.DefaultRotationConfig(),
		dataDir:      tempDir,
	}

	backups, err := service.ListBackups(context.Background())

	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("Expected 0 backups for empty directory, got %d", len(backups))
	}
}

func TestServiceImpl_ValidateBackup(t *testing.T) {
	tempDir := setupTempDir(t)
	backupDir := filepath.Join(tempDir, "backups")

	service := &ServiceImpl{
		secretsStore: &mockSecretsStore{},
		usersStore:   &mockUsersStore{},
		config:       rotation.DefaultRotationConfig(),
		dataDir:      tempDir,
	}

	t.Run("valid backup", func(t *testing.T) {
		createTestBackup(t, backupDir, "valid-backup")
		backupPath := filepath.Join(backupDir, "valid-backup")

		err := service.ValidateBackup(context.Background(), backupPath)
		if err != nil {
			t.Errorf("Expected valid backup to pass validation, got error: %v", err)
		}
	})

	t.Run("invalid backup - missing files", func(t *testing.T) {
		invalidBackupPath := filepath.Join(backupDir, "invalid-backup")
		if err := os.MkdirAll(invalidBackupPath, 0700); err != nil {
			t.Fatalf("Failed to create invalid backup directory: %v", err)
		}
		// Don't create the required files

		err := service.ValidateBackup(context.Background(), invalidBackupPath)
		if err == nil {
			t.Errorf("Expected invalid backup to fail validation, but it passed")
		}
	})
}

func TestServiceImpl_CleanupOldBackups(t *testing.T) {
	tempDir := setupTempDir(t)
	backupDir := filepath.Join(tempDir, "backups")

	// Create multiple rotation backups with different timestamps
	createTestBackup(t, backupDir, "rotate-20240901-100000")
	createTestBackup(t, backupDir, "rotate-20240902-100000")
	createTestBackup(t, backupDir, "rotate-20240903-100000")
	createTestBackup(t, backupDir, "manual-20240904-100000") // Should not be cleaned

	service := &ServiceImpl{
		secretsStore: &mockSecretsStore{},
		usersStore:   &mockUsersStore{},
		config:       rotation.DefaultRotationConfig(),
		dataDir:      tempDir,
	}

	t.Run("cleanup keeps specified number", func(t *testing.T) {
		err := service.CleanupOldBackups(context.Background(), 2)
		if err != nil {
			t.Fatalf("CleanupOldBackups failed: %v", err)
		}

		// Check remaining backups
		backups, err := service.ListBackups(context.Background())
		if err != nil {
			t.Fatalf("ListBackups failed: %v", err)
		}

		// Should have 3 backups total: 2 rotation + 1 manual
		if len(backups) != 3 {
			t.Errorf("Expected 3 backups after cleanup, got %d", len(backups))
		}

		// Count rotation backups specifically
		rotationCount := 0
		for _, backup := range backups {
			if backup.Type == rotation.BackupTypeRotation {
				rotationCount++
			}
		}
		if rotationCount != 2 {
			t.Errorf("Expected 2 rotation backups after cleanup, got %d", rotationCount)
		}
	})

	t.Run("invalid keep count", func(t *testing.T) {
		err := service.CleanupOldBackups(context.Background(), 0)
		if err == nil {
			t.Errorf("Expected error for invalid keep count, got nil")
		}
	})
}

func TestServiceImpl_TokenRotation(t *testing.T) {
	service := &ServiceImpl{
		secretsStore: &mockSecretsStore{},
		usersStore:   &mockUsersStore{},
		config:       rotation.DefaultRotationConfig(),
		dataDir:      "/test",
	}

	t.Run("rotate user token", func(t *testing.T) {
		token, err := service.RotateUserToken(context.Background(), "testuser")
		if err != nil {
			t.Fatalf("RotateUserToken failed: %v", err)
		}
		if token != "rotated-token" {
			t.Errorf("Expected 'rotated-token', got %q", token)
		}
	})

	t.Run("rotate self token", func(t *testing.T) {
		token, err := service.RotateSelfToken(context.Background(), "currentuser")
		if err != nil {
			t.Fatalf("RotateSelfToken failed: %v", err)
		}
		if token != "rotated-token" {
			t.Errorf("Expected 'rotated-token', got %q", token)
		}
	})
}

func TestServiceImpl_RestoreFromBackup(t *testing.T) {
	tempDir := setupTempDir(t)
	backupDir := filepath.Join(tempDir, "backups")

	// Create original data files
	keyPath := filepath.Join(tempDir, "master.key")
	secretsPath := filepath.Join(tempDir, "secrets.json")

	if err := os.WriteFile(keyPath, []byte("current-key"), 0600); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := os.WriteFile(secretsPath, []byte(`{"current":"secret"}`), 0600); err != nil {
		t.Fatalf("Failed to create secrets file: %v", err)
	}

	// Create a test backup
	createTestBackup(t, backupDir, "test-backup")

	service := &ServiceImpl{
		secretsStore: &mockSecretsStore{},
		usersStore:   &mockUsersStore{},
		config:       rotation.DefaultRotationConfig(),
		dataDir:      tempDir,
	}

	t.Run("restore from specific backup", func(t *testing.T) {
		err := service.RestoreFromBackup(context.Background(), "test-backup")
		if err != nil {
			t.Fatalf("RestoreFromBackup failed: %v", err)
		}

		// Verify a pre-restore backup was created
		backups, err := service.ListBackups(context.Background())
		if err != nil {
			t.Fatalf("ListBackups failed: %v", err)
		}

		foundPreRestore := false
		for _, backup := range backups {
			if backup.Type == rotation.BackupTypePreRestore {
				foundPreRestore = true
				break
			}
		}
		if !foundPreRestore {
			t.Errorf("Expected pre-restore backup to be created")
		}
	})

	t.Run("restore from nonexistent backup", func(t *testing.T) {
		err := service.RestoreFromBackup(context.Background(), "nonexistent-backup")
		if err == nil {
			t.Errorf("Expected error for nonexistent backup, got nil")
		}
	})
}
