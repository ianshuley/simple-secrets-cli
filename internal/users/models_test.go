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

package users

import (
	"testing"
	"time"
)

func TestUser_Can(t *testing.T) {
	adminUser := &User{Username: "admin", Role: RoleAdmin}
	readerUser := &User{Username: "reader", Role: RoleReader}

	testCases := []struct {
		name     string
		user     *User
		perm     Permission
		expected bool
	}{
		{"admin can read", adminUser, PermissionRead, true},
		{"admin can write", adminUser, PermissionWrite, true},
		{"admin can delete", adminUser, PermissionDelete, true},
		{"reader can read", readerUser, PermissionRead, true},
		{"reader cannot write", readerUser, PermissionWrite, false},
		{"reader cannot delete", readerUser, PermissionDelete, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.user.Can(tc.perm)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	adminUser := &User{Role: RoleAdmin}
	readerUser := &User{Role: RoleReader}

	if !adminUser.IsAdmin() {
		t.Error("Admin user should return true for IsAdmin()")
	}

	if readerUser.IsAdmin() {
		t.Error("Reader user should return false for IsAdmin()")
	}
}

func TestUser_IsReader(t *testing.T) {
	adminUser := &User{Role: RoleAdmin}
	readerUser := &User{Role: RoleReader}

	if adminUser.IsReader() {
		t.Error("Admin user should return false for IsReader()")
	}

	if !readerUser.IsReader() {
		t.Error("Reader user should return true for IsReader()")
	}
}

func TestRolePermissions_Has(t *testing.T) {
	testCases := []struct {
		name     string
		role     Role
		perm     Permission
		expected bool
	}{
		{"admin has read", RoleAdmin, PermissionRead, true},
		{"admin has write", RoleAdmin, PermissionWrite, true},
		{"admin has delete", RoleAdmin, PermissionDelete, true},
		{"reader has read", RoleReader, PermissionRead, true},
		{"reader lacks write", RoleReader, PermissionWrite, false},
		{"reader lacks delete", RoleReader, PermissionDelete, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rolePerms := DefaultRolePermissions[tc.role]
			result := rolePerms.Has(tc.perm)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestUser_TokenRotation(t *testing.T) {
	rotatedAt := time.Now()
	user := &User{
		Username:       "testuser",
		HashedToken:    "hash123",
		Role:           RoleReader,
		TokenRotatedAt: &rotatedAt,
	}

	if user.TokenRotatedAt == nil {
		t.Error("TokenRotatedAt should not be nil")
	}

	if !user.TokenRotatedAt.Equal(rotatedAt) {
		t.Error("TokenRotatedAt should match the set time")
	}
}

func TestRole_Constants(t *testing.T) {
	if RoleAdmin != "admin" {
		t.Errorf("RoleAdmin should be 'admin', got %s", RoleAdmin)
	}

	if RoleReader != "reader" {
		t.Errorf("RoleReader should be 'reader', got %s", RoleReader)
	}
}

func TestParseRole(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    Role
		expectError bool
	}{
		{"valid admin role", "admin", RoleAdmin, false},
		{"valid reader role", "reader", RoleReader, false},
		{"invalid role returns error", "invalid", "", true},
		{"empty string returns error", "", "", true},
		{"case sensitive admin uppercase", "ADMIN", "", true},
		{"case sensitive reader uppercase", "READER", "", true},
		{"case sensitive mixed case", "Admin", "", true},
		{"whitespace only returns error", "   ", "", true},
		{"role with leading whitespace", "  admin", RoleAdmin, false},
		{"role with trailing whitespace", "reader  ", RoleReader, false},
		{"numeric role returns error", "123", "", true},
		{"special characters return error", "admin!", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseRole(tc.input)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tc.input, err)
				}
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			}
		})
	}
}

func TestGetTokenRotationDisplay(t *testing.T) {
	testCases := []struct {
		name     string
		user     *User
		expected string
	}{
		{
			name: "nil timestamp shows legacy user",
			user: &User{TokenRotatedAt: nil},
			expected: "Legacy user (no rotation info)",
		},
		{
			name: "valid timestamp formatted correctly",
			user: &User{TokenRotatedAt: &time.Time{}},
			expected: "0001-01-01 00:00:00",
		},
		{
			name: "zero time formatted",
			user: &User{TokenRotatedAt: &time.Time{}},
			expected: "0001-01-01 00:00:00",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.user.GetTokenRotationDisplay()
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}