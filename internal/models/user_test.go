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

package models

import (
	"testing"
	"time"
)

func TestUser_Can(t *testing.T) {
	perms := RolePermissions{
		RoleAdmin:  []string{"read", "write", "delete"},
		RoleReader: []string{"read"},
	}

	adminUser := &User{Username: "admin", Role: RoleAdmin}
	readerUser := &User{Username: "reader", Role: RoleReader}

	testCases := []struct {
		name     string
		user     *User
		perm     string
		expected bool
	}{
		{"admin can read", adminUser, "read", true},
		{"admin can write", adminUser, "write", true},
		{"admin can delete", adminUser, "delete", true},
		{"admin cannot unknown", adminUser, "unknown", false},
		{"reader can read", readerUser, "read", true},
		{"reader cannot write", readerUser, "write", false},
		{"reader cannot delete", readerUser, "delete", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.user.Can(tc.perm, perms)
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
	perms := RolePermissions{
		RoleAdmin:  []string{"read", "write", "delete"},
		RoleReader: []string{"read"},
	}

	testCases := []struct {
		name     string
		role     Role
		perm     string
		expected bool
	}{
		{"admin has read", RoleAdmin, "read", true},
		{"admin has write", RoleAdmin, "write", true},
		{"admin has delete", RoleAdmin, "delete", true},
		{"admin lacks unknown", RoleAdmin, "unknown", false},
		{"reader has read", RoleReader, "read", true},
		{"reader lacks write", RoleReader, "write", false},
		{"reader lacks delete", RoleReader, "delete", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := perms.Has(tc.role, tc.perm)
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
		TokenHash:      "hash123",
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
