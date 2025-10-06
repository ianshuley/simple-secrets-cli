package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"simple-secrets/internal/platform"
)

func main() {
	// Set up test environment
	testDir := "/tmp/test-race-controlled"
	os.RemoveAll(testDir)
	os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", testDir)

	ctx := context.Background()

	// Create platform with proper config (32 bytes for AES-256)
	config := platform.Config{
		DataDir:   testDir,
		MasterKey: []byte("test-master-key-1234567890123456"),
	}

	p, err := platform.New(ctx, config)
	if err != nil {
		fmt.Printf("Failed to create platform: %v\n", err)
		return
	}

	// Test concurrent operations - testing the raw repository level
	numOperations := 50
	var wg sync.WaitGroup
	successCount := 0
	var successMutex sync.Mutex

	fmt.Printf("Testing %d concurrent operations...\n", numOperations)

	start := time.Now()

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			key := fmt.Sprintf("concurrent-test-%03d", index)
			value := fmt.Sprintf("value-%03d", index)

			err := p.Secrets.Put(ctx, key, value)
			if err != nil {
				fmt.Printf("Failed to store %s: %v\n", key, err)
			} else {
				successMutex.Lock()
				successCount++
				successMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("Concurrent operations completed in %v\n", elapsed)
	fmt.Printf("Success count: %d/%d\n", successCount, numOperations)

	// Verify all secrets were actually stored
	secrets, err := p.Secrets.List(ctx)
	if err != nil {
		fmt.Printf("Failed to list secrets: %v\n", err)
		return
	}

	actualCount := 0
	for _, secret := range secrets {
		if len(secret.Key) >= 16 && secret.Key[:16] == "concurrent-test-" {
			actualCount++
		}
	}

	fmt.Printf("Actually stored: %d/%d secrets\n", actualCount, numOperations)

	if actualCount == numOperations {
		fmt.Println("✅ Race condition test PASSED - all operations succeeded")
	} else {
		fmt.Printf("❌ Race condition test FAILED - lost %d operations\n", numOperations-actualCount)

		// Show which ones are missing
		missing := []string{}
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("concurrent-test-%03d", i)
			found := false
			for _, secret := range secrets {
				if secret.Key == key {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, key)
			}
		}
		fmt.Printf("Missing secrets: %v\n", missing)
	}
}
