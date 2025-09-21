/*package api

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

// Package api provides composable interfaces for building external tools,
// APIs, and integrations on top of simple-secrets. These interfaces follow
// the principle of accepting interfaces and returning concrete types.
package api

// User represents an authenticated user with role-based permissions
type User struct {
	Username string
	Role     string
}

// SecretReader provides read-only access to secrets.
// Perfect for monitoring, ansible fact gathering, or read-only API endpoints.
type SecretReader interface {
	// Get retrieves a secret value by key
	Get(key string) (string, error)

	// List returns all available secret keys (enabled secrets only)
	List() []string

	// IsEnabled checks if a secret is currently enabled
	IsEnabled(key string) bool
}

// SecretWriter provides write operations for secrets.
// Use alongside SecretReader for full secret management in ansible or APIs.
type SecretWriter interface {
	// Put stores a secret value with the given key
	Put(key, value string) error

	// Delete removes a secret permanently
	Delete(key string) error

	// Enable activates a previously disabled secret
	Enable(key string) error

	// Disable temporarily deactivates a secret without deleting it
	Disable(key string) error
}

// Authenticator handles user authentication and authorization.
// Essential for any API or multi-user tooling built on simple-secrets.
type Authenticator interface {
	// Authenticate validates a token and returns the associated user
	Authenticate(token string) (*User, error)

	// CanRead checks if a user has read permissions
	CanRead(user *User) bool

	// CanWrite checks if a user has write permissions
	CanWrite(user *User) bool

	// CanAdmin checks if a user has administrative permissions
	CanAdmin(user *User) bool
}

// UserManager provides user account management operations.
// Needed for administrative APIs and user provisioning tools.
type UserManager interface {
	// CreateUser creates a new user and returns their authentication token
	CreateUser(username, role string) (token string, err error)

	// DeleteUser removes a user account
	DeleteUser(username string) error

	// ListUsers returns all user accounts
	ListUsers() ([]*User, error)

	// RotateToken generates a new authentication token for a user
	RotateToken(username string) (newToken string, err error)
}

// AdminOperations provides high-level administrative functions.
// These are typically used by administrative APIs or deployment automation.
type AdminOperations interface {
	// Backup creates a backup of all secrets and configuration
	Backup(backupDir string) error

	// Restore restores secrets and configuration from a backup
	Restore(backupDir string) error

	// RotateMasterKey rotates the master encryption key (re-encrypts all secrets)
	RotateMasterKey(backupDir string) error
}

// SecretsService combines read and write operations for full secret management.
// This is a convenience interface for tools that need both read and write access.
type SecretsService interface {
	SecretReader
	SecretWriter
}

// SecureSecretsService adds authentication to secret operations.
// Perfect for building authenticated APIs or secure ansible modules.
type SecureSecretsService interface {
	SecretReader
	SecretWriter
	Authenticator
}

// FullService provides complete access to all simple-secrets functionality.
// Use this for administrative tools or comprehensive API implementations.
type FullService interface {
	SecretReader
	SecretWriter
	Authenticator
	UserManager
	AdminOperations
}
