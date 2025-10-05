/*package users

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

// Package users provides the public interfaces for the users domain.
// These interfaces are designed for platform extension and are considered stable contracts.
package users

import "context"

// Store is the main interface for user operations that the platform will use.
// This interface abstracts all user management functionality and is designed
// to be extended by the platform for audit logging, caching, etc.
type Store interface {
	// Create creates a new user with the given username and role
	Create(ctx context.Context, username string, role string) (*User, string, error) // returns user and token

	// GetByUsername retrieves a user by username
	GetByUsername(ctx context.Context, username string) (*User, error)

	// GetByToken retrieves a user by token value
	GetByToken(ctx context.Context, token string) (*User, error)

	// List returns all users
	List(ctx context.Context) ([]*User, error)

	// Update updates user information
	Update(ctx context.Context, user *User) error

	// Delete removes a user permanently
	Delete(ctx context.Context, username string) error

	// Enable makes a disabled user active
	Enable(ctx context.Context, username string) error

	// Disable makes a user inactive without deleting
	Disable(ctx context.Context, username string) error

	// RotateToken generates a new token for a user
	RotateToken(ctx context.Context, username string) (string, error)

	// AddToken adds a new named token to a user (multi-token support)
	AddToken(ctx context.Context, username string, tokenName string) (*Token, string, error) // returns token metadata and raw token

	// RevokeToken revokes a specific token
	RevokeToken(ctx context.Context, username string, tokenID string) error

	// ListTokens returns all tokens for a user
	ListTokens(ctx context.Context, username string) ([]*Token, error)
}

// Repository is the storage interface for the users domain.
// This interface defines business operations, not technical file operations.
// Different implementations can use files, databases, cloud storage, etc.
type Repository interface {
	// Store persists a user
	Store(ctx context.Context, user *User) error

	// Retrieve gets a user by username
	Retrieve(ctx context.Context, username string) (*User, error)

	// RetrieveByToken gets a user by token hash
	RetrieveByToken(ctx context.Context, tokenHash string) (*User, error)

	// Delete removes a user permanently
	Delete(ctx context.Context, username string) error

	// List returns all users
	List(ctx context.Context) ([]*User, error)

	// Enable marks a user as enabled
	Enable(ctx context.Context, username string) error

	// Disable marks a user as disabled
	Disable(ctx context.Context, username string) error

	// Exists checks if a user exists
	Exists(ctx context.Context, username string) (bool, error)
}
