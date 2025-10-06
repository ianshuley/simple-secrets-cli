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
package rotation

import (
	"strings"
	"testing"
	"time"
)

func TestBackupInfo_String(t *testing.T) {
	backup := &BackupInfo{
		Name:      "test-backup.bak",
		Path:      "/test/backup/file.bak",
		Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Size:      1024,
	}

	result := backup.String()

	// Should contain key information
	if !strings.Contains(result, "test-backup.bak") {
		t.Errorf("BackupInfo.String() should contain name, got: %s", result)
	}
}

func TestBackupInfo_StringInvalidCases(t *testing.T) {
	tests := []struct {
		name   string
		backup *BackupInfo
		want   string
	}{
		{
			name: "empty_name",
			backup: &BackupInfo{
				Name:      "",
				Path:      "/test/backup",
				Timestamp: time.Now(),
			},
			want: "Invalid backup info",
		},
		{
			name: "zero_timestamp",
			backup: &BackupInfo{
				Name:      "test.bak",
				Path:      "/test/backup",
				Timestamp: time.Time{},
			},
			want: "Invalid backup info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.backup.String()
			if result != tt.want {
				t.Errorf("BackupInfo.String() = %q, want %q", result, tt.want)
			}
		})
	}
}
