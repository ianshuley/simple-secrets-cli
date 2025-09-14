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
package internal

// SecretsManager defines the interface for secrets management operations.
// This interface abstracts the underlying storage implementation and ensures
// all operations are thread-safe through the implementing type's synchronization.
type SecretsManager interface {
	// Core CRUD operations
	Put(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	ListKeys() []string

	// Secret state management
	DisableSecret(key string) error
	EnableSecret(key string) error
	IsEnabled(key string) bool

	// Master key operations
	RotateMasterKey(backupDir string) error

	// Backup and restore operations
	CreateBackup(backupDir string) error
	RestoreFromBackup(backupDir string) error
}

// UserManager defines the interface for user authentication and authorization.
// This interface provides thread-safe access to user data and permissions
// through the implementing type's synchronization mechanisms.
type UserManager interface {
	// Authentication
	Lookup(token string) (*User, error)

	// User data access
	Users() []*User
	Permissions() RolePermissions

	// User management (for future expansion)
	CreateUser(username, role string) (string, error) // returns token
	DeleteUser(username string) error
	UpdateUserRole(username, newRole string) error
}

// StorageBackend defines the interface for different storage implementations.
// This abstraction allows for different storage backends (filesystem, cloud, etc.)
// while maintaining thread-safety guarantees.
type StorageBackend interface {
	// File operations
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm FileMode) error
	AtomicWriteFile(path string, data []byte, perm FileMode) error

	// Directory operations
	MkdirAll(path string, perm FileMode) error
	RemoveAll(path string) error
	Exists(path string) bool

	// Listing operations
	ListDir(path string) ([]string, error)
}

// FileMode represents file permissions (abstraction over os.FileMode)
type FileMode uint32

const (
	// Standard file permissions
	FileMode0600 FileMode = 0600
	FileMode0644 FileMode = 0644
	FileMode0755 FileMode = 0755
)

// SecretStore represents the composition of secrets and user management
// This interface combines both managers for comprehensive secret store operations
type SecretStore interface {
	SecretsManager
	UserManager

	// Composite operations that span both managers
	Initialize() error
	Close() error
}
