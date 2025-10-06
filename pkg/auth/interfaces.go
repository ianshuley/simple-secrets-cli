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

// Package auth provides the public interfaces for the authentication and authorization domain.
// These interfaces are designed for platform extension and are considered stable contracts.
package auth

import "context"

// Permission represents an action that can be performed in the system
type Permission string

const (
	// Core permissions
	PermissionRead      Permission = "read"             // Read secrets
	PermissionWrite     Permission = "write"            // Create/update/delete secrets
	PermissionRotateOwn Permission = "rotate-own-token" // Rotate own token

	// Administrative permissions
	PermissionRotateTokens Permission = "rotate-tokens" // Rotate any user's token
	PermissionManageUsers  Permission = "manage-users"  // Create/delete/modify users
)

// Role represents a user role with associated permissions
type Role string

const (
	RoleAdmin  Role = "admin"  // Full access to all operations
	RoleReader Role = "reader" // Read-only access
)

// TokenValidator validates authentication tokens and returns user information
type TokenValidator interface {
	// ValidateToken validates a token and returns the associated user context
	ValidateToken(ctx context.Context, token string) (*UserContext, error)

	// ValidateTokenHash validates a token hash and returns the associated user context
	ValidateTokenHash(ctx context.Context, tokenHash string) (*UserContext, error)
}

// PermissionChecker checks if users have specific permissions
type PermissionChecker interface {
	// HasPermission checks if a user has a specific permission
	HasPermission(ctx context.Context, user *UserContext, permission Permission) bool

	// RequirePermission checks permission and returns error if not granted
	RequirePermission(ctx context.Context, user *UserContext, permission Permission) error

	// GetPermissions returns all permissions for a role
	GetPermissions(ctx context.Context, role Role) []Permission
}

// RoleManager manages role definitions and role-to-permission mappings
type RoleManager interface {
	// ValidateRole checks if a role is valid
	ValidateRole(ctx context.Context, role Role) error

	// GetRolePermissions returns the permissions for a specific role
	GetRolePermissions(ctx context.Context, role Role) ([]Permission, error)

	// ListRoles returns all available roles
	ListRoles(ctx context.Context) ([]Role, error)
}

// AuthService is the main interface for authentication and authorization operations
// This interface is designed to be extended by the platform for audit logging, etc.
type AuthService interface {
	TokenValidator
	PermissionChecker
	RoleManager

	// Authenticate validates credentials and returns user context
	Authenticate(ctx context.Context, token string) (*UserContext, error)

	// Authorize checks if user can perform specific action
	Authorize(ctx context.Context, user *UserContext, permission Permission) error

	// HashToken creates a secure hash of a token for storage (utility function)
	HashToken(token string) string
}

// UserContext represents an authenticated user with their role and permissions
type UserContext struct {
	Username    string       `json:"username"`
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
	TokenHash   string       `json:"token_hash"`
}

// HasPermission checks if this user context has a specific permission
func (uc *UserContext) HasPermission(permission Permission) bool {
	for _, p := range uc.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// IsAdmin returns true if the user has admin role
func (uc *UserContext) IsAdmin() bool {
	return uc.Role == RoleAdmin
}

// IsReader returns true if the user has reader role
func (uc *UserContext) IsReader() bool {
	return uc.Role == RoleReader
}

// CanRead returns true if user can read secrets
func (uc *UserContext) CanRead() bool {
	return uc.HasPermission(PermissionRead)
}

// CanWrite returns true if user can write secrets
func (uc *UserContext) CanWrite() bool {
	return uc.HasPermission(PermissionWrite)
}

// CanManageUsers returns true if user can manage other users
func (uc *UserContext) CanManageUsers() bool {
	return uc.HasPermission(PermissionManageUsers)
}
