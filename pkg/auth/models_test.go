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
	"testing"
)

func TestParseRole(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Role
		wantErr  bool
	}{
		{
			name:     "valid admin role",
			input:    "admin",
			expected: RoleAdmin,
			wantErr:  false,
		},
		{
			name:     "valid reader role",
			input:    "reader",
			expected: RoleReader,
			wantErr:  false,
		},
		{
			name:     "invalid role returns error",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty string returns error",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "whitespace only returns error",
			input:    "   ",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "role with leading whitespace",
			input:    "  admin",
			expected: RoleAdmin,
			wantErr:  false,
		},
		{
			name:     "role with trailing whitespace",
			input:    "reader  ",
			expected: RoleReader,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRole(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseRole() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParsePermission(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Permission
		wantErr  bool
	}{
		{
			name:     "valid read permission",
			input:    "read",
			expected: PermissionRead,
			wantErr:  false,
		},
		{
			name:     "valid write permission",
			input:    "write",
			expected: PermissionWrite,
			wantErr:  false,
		},
		{
			name:     "valid rotate-own-token permission",
			input:    "rotate-own-token",
			expected: PermissionRotateOwn,
			wantErr:  false,
		},
		{
			name:     "valid rotate-tokens permission",
			input:    "rotate-tokens",
			expected: PermissionRotateTokens,
			wantErr:  false,
		},
		{
			name:     "valid manage-users permission",
			input:    "manage-users",
			expected: PermissionManageUsers,
			wantErr:  false,
		},
		{
			name:     "invalid permission returns error",
			input:    "invalid",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty string returns error",
			input:    "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePermission(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePermission() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ParsePermission() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRolePermissions_Has(t *testing.T) {
	rolePerms := NewDefaultRolePermissions()

	// Test admin permissions
	if !rolePerms.Has(RoleAdmin, PermissionRead) {
		t.Error("Admin should have read permission")
	}

	if !rolePerms.Has(RoleAdmin, PermissionWrite) {
		t.Error("Admin should have write permission")
	}

	if !rolePerms.Has(RoleAdmin, PermissionManageUsers) {
		t.Error("Admin should have manage users permission")
	}

	// Test reader permissions
	if !rolePerms.Has(RoleReader, PermissionRead) {
		t.Error("Reader should have read permission")
	}

	if rolePerms.Has(RoleReader, PermissionWrite) {
		t.Error("Reader should not have write permission")
	}

	if rolePerms.Has(RoleReader, PermissionManageUsers) {
		t.Error("Reader should not have manage users permission")
	}
}

func TestRolePermissions_ValidateRole(t *testing.T) {
	rolePerms := NewDefaultRolePermissions()

	// Test valid roles
	if err := rolePerms.ValidateRole(RoleAdmin); err != nil {
		t.Errorf("Admin role should be valid: %v", err)
	}

	if err := rolePerms.ValidateRole(RoleReader); err != nil {
		t.Errorf("Reader role should be valid: %v", err)
	}

	// Test invalid role
	if err := rolePerms.ValidateRole(Role("invalid")); err == nil {
		t.Error("Invalid role should not be valid")
	}
}

func TestRolePermissions_GetAllRoles(t *testing.T) {
	rolePerms := NewDefaultRolePermissions()
	roles := rolePerms.GetAllRoles()

	if len(roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(roles))
	}

	// Check that both expected roles are present
	foundAdmin := false
	foundReader := false

	for _, role := range roles {
		if role == RoleAdmin {
			foundAdmin = true
		}
		if role == RoleReader {
			foundReader = true
		}
	}

	if !foundAdmin {
		t.Error("Admin role not found in GetAllRoles")
	}

	if !foundReader {
		t.Error("Reader role not found in GetAllRoles")
	}
}

func TestNewUserContext(t *testing.T) {
	rolePerms := NewDefaultRolePermissions()

	userContext := NewUserContext("testuser", RoleAdmin, "hash123", rolePerms)

	if userContext.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", userContext.Username)
	}

	if userContext.Role != RoleAdmin {
		t.Errorf("Expected role 'admin', got '%s'", userContext.Role)
	}

	if userContext.TokenHash != "hash123" {
		t.Errorf("Expected token hash 'hash123', got '%s'", userContext.TokenHash)
	}

	// Check that permissions were populated
	if len(userContext.Permissions) == 0 {
		t.Error("Expected permissions to be populated")
	}

	// Verify some key permissions are present
	hasRead := false
	hasWrite := false
	for _, perm := range userContext.Permissions {
		if perm == PermissionRead {
			hasRead = true
		}
		if perm == PermissionWrite {
			hasWrite = true
		}
	}

	if !hasRead {
		t.Error("Admin should have read permission")
	}

	if !hasWrite {
		t.Error("Admin should have write permission")
	}
}

func TestAuthError(t *testing.T) {
	// Test basic error
	err := NewAuthError(AuthErrorInvalidToken, "Token is invalid")
	if err.Type != AuthErrorInvalidToken {
		t.Errorf("Expected error type %s, got %s", AuthErrorInvalidToken, err.Type)
	}

	expectedMsg := "INVALID_TOKEN: Token is invalid"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Test error with details
	err = NewAuthError(AuthErrorPermissionDenied, "Access denied", "user lacks write permission")
	expectedMsgWithDetails := "PERMISSION_DENIED: Access denied (user lacks write permission)"
	if err.Error() != expectedMsgWithDetails {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsgWithDetails, err.Error())
	}

	// Test convenience constructors
	invalidTokenErr := NewInvalidTokenError("token expired")
	if invalidTokenErr.Type != AuthErrorInvalidToken {
		t.Error("NewInvalidTokenError should create INVALID_TOKEN error")
	}

	permDeniedErr := NewPermissionDeniedError(PermissionWrite, "user is reader")
	if permDeniedErr.Type != AuthErrorPermissionDenied {
		t.Error("NewPermissionDeniedError should create PERMISSION_DENIED error")
	}

	invalidRoleErr := NewInvalidRoleError("superuser")
	if invalidRoleErr.Type != AuthErrorInvalidRole {
		t.Error("NewInvalidRoleError should create INVALID_ROLE error")
	}
}
