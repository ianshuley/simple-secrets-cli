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
	"context"
	"os"
	"testing"

	"simple-secrets/internal/users"
	"simple-secrets/pkg/auth"
)

func TestServiceImpl_Authentication(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "auth_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create user store
	repo := users.NewFileRepository(tempDir)
	userStore := users.NewStore(repo)
	ctx := context.Background()

	// Create auth service
	authService := NewServiceWithDefaults(userStore)

	// Create test user
	_, token, err := userStore.Create(ctx, "testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test successful authentication
	userContext, err := authService.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	if userContext.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", userContext.Username)
	}

	if userContext.Role != auth.RoleAdmin {
		t.Errorf("Expected role 'admin', got '%s'", userContext.Role)
	}

	// Test invalid token
	_, err = authService.Authenticate(ctx, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	// Test empty token
	_, err = authService.Authenticate(ctx, "")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestServiceImpl_TokenValidation(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "auth_token_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create user store and auth service
	repo := users.NewFileRepository(tempDir)
	userStore := users.NewStore(repo)
	authService := NewServiceWithDefaults(userStore)
	ctx := context.Background()

	// Create test user
	_, token, err := userStore.Create(ctx, "testuser", "reader")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test ValidateToken
	userContext, err := authService.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if userContext.Role != auth.RoleReader {
		t.Errorf("Expected role 'reader', got '%s'", userContext.Role)
	}

	// Test ValidateTokenHash
	tokenHash := authService.HashToken(token)
	userContext2, err := authService.ValidateTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("Failed to validate token hash: %v", err)
	}

	if userContext2.Username != userContext.Username {
		t.Errorf("Token hash validation returned different user")
	}

	// Test disabled user
	err = userStore.Disable(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to disable user: %v", err)
	}

	_, err = authService.ValidateToken(ctx, token)
	if err == nil {
		t.Error("Expected error when validating token for disabled user")
	}
}

func TestServiceImpl_PermissionChecking(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "auth_perms_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create user store and auth service
	repo := users.NewFileRepository(tempDir)
	userStore := users.NewStore(repo)
	authService := NewServiceWithDefaults(userStore)
	ctx := context.Background()

	// Create admin user
	_, adminToken, err := userStore.Create(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	adminContext, err := authService.Authenticate(ctx, adminToken)
	if err != nil {
		t.Fatalf("Failed to authenticate admin: %v", err)
	}

	// Create reader user
	_, readerToken, err := userStore.Create(ctx, "reader", "reader")
	if err != nil {
		t.Fatalf("Failed to create reader user: %v", err)
	}

	readerContext, err := authService.Authenticate(ctx, readerToken)
	if err != nil {
		t.Fatalf("Failed to authenticate reader: %v", err)
	}

	// Test admin permissions
	if !authService.HasPermission(ctx, adminContext, auth.PermissionRead) {
		t.Error("Admin should have read permission")
	}

	if !authService.HasPermission(ctx, adminContext, auth.PermissionWrite) {
		t.Error("Admin should have write permission")
	}

	if !authService.HasPermission(ctx, adminContext, auth.PermissionManageUsers) {
		t.Error("Admin should have manage users permission")
	}

	// Test reader permissions
	if !authService.HasPermission(ctx, readerContext, auth.PermissionRead) {
		t.Error("Reader should have read permission")
	}

	if authService.HasPermission(ctx, readerContext, auth.PermissionWrite) {
		t.Error("Reader should not have write permission")
	}

	if authService.HasPermission(ctx, readerContext, auth.PermissionManageUsers) {
		t.Error("Reader should not have manage users permission")
	}

	// Test RequirePermission
	err = authService.RequirePermission(ctx, adminContext, auth.PermissionWrite)
	if err != nil {
		t.Errorf("Admin should be authorized for write: %v", err)
	}

	err = authService.RequirePermission(ctx, readerContext, auth.PermissionWrite)
	if err == nil {
		t.Error("Reader should not be authorized for write")
	}
}

func TestServiceImpl_RoleManagement(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "auth_roles_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create user store and auth service
	repo := users.NewFileRepository(tempDir)
	userStore := users.NewStore(repo)
	authService := NewServiceWithDefaults(userStore)
	ctx := context.Background()

	// Test ValidateRole
	err = authService.ValidateRole(ctx, auth.RoleAdmin)
	if err != nil {
		t.Errorf("Admin role should be valid: %v", err)
	}

	err = authService.ValidateRole(ctx, auth.RoleReader)
	if err != nil {
		t.Errorf("Reader role should be valid: %v", err)
	}

	err = authService.ValidateRole(ctx, auth.Role("invalid"))
	if err == nil {
		t.Error("Invalid role should not be valid")
	}

	// Test GetRolePermissions
	adminPerms, err := authService.GetRolePermissions(ctx, auth.RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to get admin permissions: %v", err)
	}

	if len(adminPerms) == 0 {
		t.Error("Admin should have permissions")
	}

	readerPerms, err := authService.GetRolePermissions(ctx, auth.RoleReader)
	if err != nil {
		t.Fatalf("Failed to get reader permissions: %v", err)
	}

	if len(readerPerms) == 0 {
		t.Error("Reader should have permissions")
	}

	if len(adminPerms) <= len(readerPerms) {
		t.Error("Admin should have more permissions than reader")
	}

	// Test ListRoles
	roles, err := authService.ListRoles(ctx)
	if err != nil {
		t.Fatalf("Failed to list roles: %v", err)
	}

	if len(roles) < 2 {
		t.Error("Should have at least 2 roles (admin, reader)")
	}
}

func TestUserContext_Methods(t *testing.T) {
	// Test admin user context
	adminPerms := []auth.Permission{
		auth.PermissionRead,
		auth.PermissionWrite,
		auth.PermissionManageUsers,
	}

	adminContext := &auth.UserContext{
		Username:    "admin",
		Role:        auth.RoleAdmin,
		Permissions: adminPerms,
		TokenHash:   "hash123",
	}

	if !adminContext.IsAdmin() {
		t.Error("Admin context should return true for IsAdmin")
	}

	if adminContext.IsReader() {
		t.Error("Admin context should return false for IsReader")
	}

	if !adminContext.CanRead() {
		t.Error("Admin should be able to read")
	}

	if !adminContext.CanWrite() {
		t.Error("Admin should be able to write")
	}

	if !adminContext.CanManageUsers() {
		t.Error("Admin should be able to manage users")
	}

	// Test reader user context
	readerPerms := []auth.Permission{auth.PermissionRead}

	readerContext := &auth.UserContext{
		Username:    "reader",
		Role:        auth.RoleReader,
		Permissions: readerPerms,
		TokenHash:   "hash456",
	}

	if readerContext.IsAdmin() {
		t.Error("Reader context should return false for IsAdmin")
	}

	if !readerContext.IsReader() {
		t.Error("Reader context should return true for IsReader")
	}

	if !readerContext.CanRead() {
		t.Error("Reader should be able to read")
	}

	if readerContext.CanWrite() {
		t.Error("Reader should not be able to write")
	}

	if readerContext.CanManageUsers() {
		t.Error("Reader should not be able to manage users")
	}
}