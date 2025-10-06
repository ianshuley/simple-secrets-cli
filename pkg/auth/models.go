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

package auth

import (
	"fmt"
	"strings"
)

// RolePermissions maps roles to their allowed permissions
type RolePermissions map[Role][]Permission

// Has checks if a role has a specific permission
func (rp RolePermissions) Has(role Role, permission Permission) bool {
	perms := rp[role]
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a role
func (rp RolePermissions) GetPermissions(role Role) []Permission {
	return rp[role]
}

// ValidateRole checks if a role exists in the permissions map
func (rp RolePermissions) ValidateRole(role Role) error {
	if _, exists := rp[role]; !exists {
		return fmt.Errorf("invalid role: %s", role)
	}
	return nil
}

// GetAllRoles returns all roles defined in the permissions map
func (rp RolePermissions) GetAllRoles() []Role {
	roles := make([]Role, 0, len(rp))
	for role := range rp {
		roles = append(roles, role)
	}
	return roles
}

// ParseRole parses a string into a Role type with validation
func ParseRole(roleStr string) (Role, error) {
	roleStr = strings.TrimSpace(roleStr)
	if roleStr == "" {
		return "", fmt.Errorf("role cannot be empty")
	}

	role := Role(roleStr)

	// Validate against known roles
	switch role {
	case RoleAdmin, RoleReader:
		return role, nil
	default:
		return "", fmt.Errorf("invalid role: %s (must be 'admin' or 'reader')", roleStr)
	}
}

// ParsePermission parses a string into a Permission type with validation
func ParsePermission(permStr string) (Permission, error) {
	permStr = strings.TrimSpace(permStr)
	if permStr == "" {
		return "", fmt.Errorf("permission cannot be empty")
	}

	permission := Permission(permStr)

	// Validate against known permissions
	switch permission {
	case PermissionRead, PermissionWrite, PermissionRotateOwn,
		PermissionRotateTokens, PermissionManageUsers:
		return permission, nil
	default:
		return "", fmt.Errorf("invalid permission: %s", permStr)
	}
}

// NewDefaultRolePermissions creates the default role-to-permission mappings
func NewDefaultRolePermissions() RolePermissions {
	return RolePermissions{
		RoleAdmin: {
			PermissionRead,
			PermissionWrite,
			PermissionRotateOwn,
			PermissionRotateTokens,
			PermissionManageUsers,
		},
		RoleReader: {
			PermissionRead,
			PermissionRotateOwn,
		},
	}
}

// NewUserContext creates a new user context with permissions populated from role
func NewUserContext(username string, role Role, tokenHash string, rolePermissions RolePermissions) *UserContext {
	permissions := rolePermissions.GetPermissions(role)

	return &UserContext{
		Username:    username,
		Role:        role,
		Permissions: permissions,
		TokenHash:   tokenHash,
	}
}

// AuthError represents domain-specific authentication/authorization errors
type AuthError struct {
	Type    AuthErrorType `json:"type"`
	Message string        `json:"message"`
	Details string        `json:"details,omitempty"`
}

// AuthErrorType categorizes different authentication/authorization error types
type AuthErrorType string

const (
	AuthErrorInvalidToken     AuthErrorType = "INVALID_TOKEN"
	AuthErrorTokenExpired     AuthErrorType = "TOKEN_EXPIRED"
	AuthErrorPermissionDenied AuthErrorType = "PERMISSION_DENIED"
	AuthErrorInvalidRole      AuthErrorType = "INVALID_ROLE"
	AuthErrorInvalidUser      AuthErrorType = "INVALID_USER"
)

// Error implements the error interface
func (e *AuthError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewAuthError creates a new authentication/authorization error
func NewAuthError(errorType AuthErrorType, message string, details ...string) *AuthError {
	err := &AuthError{
		Type:    errorType,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewInvalidTokenError creates an invalid token error
func NewInvalidTokenError(details string) *AuthError {
	return NewAuthError(AuthErrorInvalidToken, "Invalid or expired token", details)
}

// NewPermissionDeniedError creates a permission denied error
func NewPermissionDeniedError(permission Permission, details string) *AuthError {
	message := fmt.Sprintf("Permission denied: %s required", permission)
	return NewAuthError(AuthErrorPermissionDenied, message, details)
}

// NewInvalidRoleError creates an invalid role error
func NewInvalidRoleError(role string) *AuthError {
	return NewAuthError(AuthErrorInvalidRole, fmt.Sprintf("Invalid role: %s", role))
}
