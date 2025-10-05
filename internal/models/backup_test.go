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
	"strings"
	"testing"
	"time"
)

func TestBackupInfo_String(t *testing.T) {
	timestamp := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)

	testCases := []struct {
		name     string
		backup   BackupInfo
		expected string
	}{
		{
			name: "valid backup",
			backup: BackupInfo{
				Name:      "test_backup_20231225_153045",
				Path:      "/path/to/backup",
				Timestamp: timestamp,
				IsValid:   true,
			},
			expected: "✓ test_backup_20231225_153045 (2023-12-25 15:30:45)",
		},
		{
			name: "invalid backup",
			backup: BackupInfo{
				Name:      "corrupted_backup_20231225_153045",
				Path:      "/path/to/backup",
				Timestamp: timestamp,
				IsValid:   false,
			},
			expected: "✗ corrupted_backup_20231225_153045 (2023-12-25 15:30:45)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.backup.String()
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestBackupInfo_Age(t *testing.T) {
	pastTime := time.Now().Add(-2 * time.Hour)
	backup := BackupInfo{
		Name:      "test_backup",
		Timestamp: pastTime,
	}

	age := backup.Age()
	if age < 2*time.Hour || age > 3*time.Hour {
		t.Errorf("Expected age around 2 hours, got %v", age)
	}
}

func TestBackupInfo_IsRecent(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		backup   BackupInfo
		within   time.Duration
		expected bool
	}{
		{
			name: "recent backup within 1 hour",
			backup: BackupInfo{
				Timestamp: now.Add(-30 * time.Minute),
			},
			within:   time.Hour,
			expected: true,
		},
		{
			name: "old backup beyond 1 hour",
			backup: BackupInfo{
				Timestamp: now.Add(-2 * time.Hour),
			},
			within:   time.Hour,
			expected: false,
		},
		{
			name: "exact boundary",
			backup: BackupInfo{
				Timestamp: now.Add(-time.Hour),
			},
			within:   time.Hour,
			expected: false, // Age equals duration, so not within
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.backup.IsRecent(tc.within)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestBackupInfo_BaseName(t *testing.T) {
	testCases := []struct {
		name     string
		backup   BackupInfo
		expected string
	}{
		{
			name: "standard timestamped backup",
			backup: BackupInfo{
				Name: "myapp_backup_20231225_153045",
			},
			expected: "myapp_backup",
		},
		{
			name: "complex name with underscores",
			backup: BackupInfo{
				Name: "my_complex_app_name_20231225_153045",
			},
			expected: "my_complex_app_name",
		},
		{
			name: "simple name without timestamp",
			backup: BackupInfo{
				Name: "backup",
			},
			expected: "backup",
		},
		{
			name: "name with only date part",
			backup: BackupInfo{
				Name: "backup_20231225",
			},
			expected: "backup_20231225",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.backup.BaseName()
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestBackupInfo_ValidationChecks(t *testing.T) {
	backup := BackupInfo{
		Name:      "test_backup_20231225_153045",
		Path:      "/var/backups/test_backup_20231225_153045.enc",
		Timestamp: time.Now(),
		IsValid:   true,
	}

	// Test that all fields are properly set
	if backup.Name == "" {
		t.Error("Name should not be empty")
	}

	if backup.Path == "" {
		t.Error("Path should not be empty")
	}

	if backup.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if !backup.IsValid {
		t.Error("IsValid should be true")
	}

	// Test string representation contains expected elements
	str := backup.String()
	if !strings.Contains(str, "✓") {
		t.Error("String representation should contain validity indicator")
	}

	if !strings.Contains(str, backup.Name) {
		t.Error("String representation should contain backup name")
	}
}
