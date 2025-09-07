package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestBuildInfo(t *testing.T) {
	info := BuildInfo()

	// Should contain the application name
	if !strings.Contains(info, "simple-secrets") {
		t.Errorf("BuildInfo should contain 'simple-secrets', got: %s", info)
	}

	// Should contain version
	if !strings.Contains(info, Version) {
		t.Errorf("BuildInfo should contain version '%s', got: %s", Version, info)
	}

	// Should contain platform info
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !strings.Contains(info, expectedPlatform) {
		t.Errorf("BuildInfo should contain platform '%s', got: %s", expectedPlatform, info)
	}
}

func TestShort(t *testing.T) {
	// Test with release version
	originalVersion := Version
	Version = "v1.0.0"

	result := Short()
	if result != "v1.0.0" {
		t.Errorf("Short() with release version should return 'v1.0.0', got: %s", result)
	}

	// Test with dev version
	Version = "dev"
	GitCommit = "abcd1234567890"

	result = Short()
	expected := "dev-abcd123"
	if result != expected {
		t.Errorf("Short() with dev version should return '%s', got: %s", expected, result)
	}

	// Test with short commit
	GitCommit = "abc"
	result = Short()
	expected = "dev-abc"
	if result != expected {
		t.Errorf("Short() with short commit should return '%s', got: %s", expected, result)
	}

	// Restore original version
	Version = originalVersion
}
