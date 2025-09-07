package main

import (
	"fmt"
	"strings"
)

func validateKey(key string) error {
	// Check for null bytes and other problematic characters
	if strings.Contains(key, "\x00") {
		return fmt.Errorf("key name cannot contain null bytes")
	}

	// Check for control characters (0x00-0x1F except \t, \n, \r)
	for _, r := range key {
		if r < 0x20 && r != 0x09 && r != 0x0A && r != 0x0D {
			return fmt.Errorf("key name cannot contain control characters")
		}
	}

	// Check for path traversal attempts
	if strings.Contains(key, "..") || strings.Contains(key, "/") || strings.Contains(key, "\\") {
		return fmt.Errorf("key name cannot contain path separators or path traversal sequences")
	}

	return nil
}

func main() {
	// Test cases
	testCases := []string{
		"validkey",
		"test\x00key",
		"test\x01key",
		"test\x1fkey",
		"test\x09key", // tab - should be allowed
		"test/key",
		"test..key",
		"test\\key",
	}

	for _, key := range testCases {
		err := validateKey(key)
		if err != nil {
			fmt.Printf("REJECTED: %q - %v\n", key, err)
		} else {
			fmt.Printf("ACCEPTED: %q\n", key)
		}
	}
}
