package platform

import (
	"context"
	"testing"
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
