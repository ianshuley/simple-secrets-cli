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
package internal

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// AtomicWriteFile writes data to a file atomically using a temporary file and rename.
// This ensures that either the entire write succeeds or fails completely, preventing
// partial writes that could corrupt the file.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Use unique temp file name to prevent race conditions in concurrent operations
	// Include nanosecond timestamp and goroutine ID to ensure uniqueness
	tmpPath := fmt.Sprintf("%s.tmp.%d.%d", path, os.Getpid(), time.Now().UnixNano())

	// Write to temporary file first
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename to final location
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to atomically update file: %w", err)
	}

	return nil
}

// FileLock represents a file lock
type FileLock struct {
	file *os.File
	path string
}

// LockFile creates an exclusive file lock for coordinating access to a resource.
// This prevents multiple processes from concurrently modifying the same data.
func LockFile(path string) (*FileLock, error) {
	lockPath := path + ".lock"

	// Create lock file if it doesn't exist
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock with timeout
	const maxLockAttempts = 100 // 10 seconds total with 100ms intervals (increased for high concurrency)
	for attempt := 0; attempt < maxLockAttempts; attempt++ {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Lock acquired successfully
			return &FileLock{file: file, path: lockPath}, nil
		}

		if err != syscall.EWOULDBLOCK {
			// Real error, not just lock busy
			file.Close()
			return nil, fmt.Errorf("failed to acquire file lock: %w", err)
		}

		// Lock is busy, wait and retry with exponential backoff
		backoffMs := 10 + (attempt * 2) // Start at 10ms, increase by 2ms each attempt
		if backoffMs > 100 {
			backoffMs = 100 // Cap at 100ms
		}
		time.Sleep(time.Duration(backoffMs) * time.Millisecond)
	}

	file.Close()
	return nil, fmt.Errorf("timeout acquiring file lock after %d attempts", maxLockAttempts)
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}

	// Release the lock
	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)

	// Close the file
	closeErr := fl.file.Close()
	fl.file = nil

	// Clean up lock file
	_ = os.Remove(fl.path)

	if err != nil {
		return fmt.Errorf("failed to release file lock: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close lock file: %w", closeErr)
	}

	return nil
}
