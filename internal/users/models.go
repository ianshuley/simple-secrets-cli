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

package users

import (
	"strings"
	"time"
)

// Permission represents specific actions users can perform
type Permission string

const (
	PermissionRead   Permission = "read"
	PermissionWrite  Permission = "write"
	PermissionDelete Permission = "delete"
)

// Role represents user roles in the system
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleReader Role = "reader"
)

// RolePermissions maps roles to their permissions
type RolePermissions map[Permission]bool

// DefaultRolePermissions defines standard permissions for each role
var DefaultRolePermissions = map[Role]RolePermissions{
	RoleAdmin: {
		PermissionRead:   true,
		PermissionWrite:  true,
		PermissionDelete: true,
	},
	RoleReader: {
		PermissionRead:   true,
		PermissionWrite:  false,
		PermissionDelete: false,
	},
}

// Has checks if the role has a specific permission
func (rp RolePermissions) Has(permission Permission) bool {
	return rp[permission]
}

// User represents a user in the secrets management system
type User struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	HashedToken    string     `json:"hashed_token"`
	Role           Role       `json:"role"`
	CreatedAt      time.Time  `json:"created_at"`
	TokenRotatedAt *time.Time `json:"token_rotated_at,omitempty"`
	Disabled       bool       `json:"disabled,omitempty"`
}

// Can checks if the user can perform a specific permission
func (u *User) Can(permission Permission) bool {
	rolePerms, exists := DefaultRolePermissions[u.Role]
	if !exists {
		return false
	}
	return rolePerms.Has(permission)
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsReader returns true if the user has reader role
func (u *User) IsReader() bool {
	return u.Role == RoleReader
}

// GetTokenRotationDisplay returns a formatted string for token rotation status
func (u *User) GetTokenRotationDisplay() string {
	if u.TokenRotatedAt == nil {
		return "Legacy user (no rotation info)"
	}
	return u.TokenRotatedAt.Format("2006-01-02 15:04:05")
}

// ParseRole converts a string to a Role type with validation
func ParseRole(roleStr string) (Role, error) {
	// Trim whitespace
	roleStr = strings.TrimSpace(roleStr)
	
	// Check for empty string
	if roleStr == "" {
		return "", &ValidationError{Field: "role", Message: "role cannot be empty"}
	}
	
	// Validate against known roles
	role := Role(roleStr)
	switch role {
	case RoleAdmin, RoleReader:
		return role, nil
	default:
		return "", &ValidationError{Field: "role", Message: "invalid role: must be 'admin' or 'reader'"}
	}
}

// ValidationError represents a validation error for user-related operations
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}