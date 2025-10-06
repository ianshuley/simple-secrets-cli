package platform

import (
	"context"
	"testing"

	"simple-secrets/pkg/rotation"
)

func TestPlatformBasics(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{
		DataDir:   tmpDir,
		MasterKey: []byte("test-master-key-32-chars-long!!"),
	}

	ctx := context.Background()
	platform, err := New(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}

	if platform == nil {
		t.Fatal("Platform is nil")
	}

	if platform.Secrets == nil {
		t.Error("Secrets service is nil")
	}
	if platform.Users == nil {
		t.Error("Users service is nil")
	}
	if platform.Auth == nil {
		t.Error("Auth service is nil")
	}
	if platform.Rotation == nil {
		t.Error("Rotation service is nil")
	}
}

func TestNewPlatformWithValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(t *testing.T) Config
		wantErr     bool
		errContains string
	}{
		{
			name: "empty_data_dir",
			setupConfig: func(t *testing.T) Config {
				return Config{
					DataDir:   "", // Empty data directory
					MasterKey: []byte("test-master-key-32-chars-long!!"),
				}
			},
			wantErr:     true,
			errContains: "dataDir is required",
		},
		{
			name: "empty_master_key",
			setupConfig: func(t *testing.T) Config {
				return Config{
					DataDir:   t.TempDir(),
					MasterKey: nil, // Empty master key
				}
			},
			wantErr:     true,
			errContains: "masterKey is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupConfig(t)
			ctx := context.Background()

			platform, err := New(ctx, config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("New() error = %v, want error containing %v", err, tt.errContains)
				}
				if platform != nil {
					t.Errorf("New() should return nil platform on error")
				}
			} else {
				if err != nil {
					t.Errorf("New() unexpected error = %v", err)
					return
				}
				if platform == nil {
					t.Errorf("New() returned nil platform")
				} else {
					platform.Close()
				}
			}
		})
	}
}

func TestNewWithOptionsCustomization(t *testing.T) {
	tmpDir := t.TempDir()
	config := Config{
		DataDir:   tmpDir,
		MasterKey: []byte("test-master-key-32-chars-long!!"),
	}

	ctx := context.Background()

	// Test with custom rotation config
	rotationConfig := &rotation.RotationConfig{BackupRetentionCount: 5}
	platform, err := NewWithOptions(ctx, config, WithCustomRotationConfig(rotationConfig))
	if err != nil {
		t.Fatalf("NewWithOptions() failed: %v", err)
	}
	defer platform.Close()

	if platform == nil {
		t.Fatal("NewWithOptions() returned nil platform")
	}

	// Verify all services are initialized
	if platform.Secrets == nil {
		t.Error("Secrets service is nil")
	}
	if platform.Users == nil {
		t.Error("Users service is nil")
	}
	if platform.Auth == nil {
		t.Error("Auth service is nil")
	}
	if platform.Rotation == nil {
		t.Error("Rotation service is nil")
	}
}

func TestPlatformHealthCheck(t *testing.T) {
	tmpDir := t.TempDir()
	config := Config{
		DataDir:   tmpDir,
		MasterKey: []byte("test-master-key-32-chars-long!!"),
	}

	ctx := context.Background()
	platform, err := New(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	// Test health check
	err = platform.Health(ctx)
	if err != nil {
		t.Errorf("Health() unexpected error = %v", err)
	}
}

func TestPlatformContextOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config := Config{
		DataDir:   tmpDir,
		MasterKey: []byte("test-master-key-32-chars-long!!"),
	}

	ctx := context.Background()
	platform, err := New(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create platform: %v", err)
	}
	defer platform.Close()

	// Test WithPlatform and FromContext
	ctxWithPlatform := WithPlatform(ctx, platform)
	retrievedPlatform, err := FromContext(ctxWithPlatform)
	if err != nil {
		t.Errorf("FromContext() unexpected error = %v", err)
		return
	}

	if retrievedPlatform == nil {
		t.Errorf("FromContext() returned nil platform")
	}
	if retrievedPlatform != platform {
		t.Errorf("FromContext() returned different platform instance")
	}

	// Test MustFromContext with valid context
	mustPlatform := MustFromContext(ctxWithPlatform)
	if mustPlatform != platform {
		t.Errorf("MustFromContext() returned different platform instance")
	}
}

func TestPlatformContextTimeout(t *testing.T) {
	ctx := context.Background()

	// Test WithTimeout with valid duration string
	ctxWithTimeout, cancel, err := WithTimeout(ctx, "5s")
	if err != nil {
		t.Errorf("WithTimeout() unexpected error = %v", err)
		return
	}
	defer cancel()

	if ctxWithTimeout == nil {
		t.Errorf("WithTimeout() returned nil context")
	}

	// Test context deadline is set
	_, hasDeadline := ctxWithTimeout.Deadline()
	if !hasDeadline {
		t.Errorf("WithTimeout() did not set deadline")
	}
}

func TestPlatformBackgroundContext(t *testing.T) {
	// Test Background context creation
	tmpDir := t.TempDir()
	config := Config{
		DataDir:   tmpDir,
		MasterKey: []byte("test-master-key-32-chars-long!!"),
	}
	platformForCtx, err := New(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create platform for context test: %v", err)
	}
	defer platformForCtx.Close()

	ctx := Background(platformForCtx)
	if ctx == nil {
		t.Errorf("Background() returned nil context")
	}

	// Test TODO context creation
	todoCtx := TODO(platformForCtx)
	if todoCtx == nil {
		t.Errorf("TODO() returned nil context")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
