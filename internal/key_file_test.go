package internal

import (
	"encoding/base64"
	"os"
	"testing"
)

func TestLoadOrCreateKey_CreatesKeyIfMissing(t *testing.T) {
	s := &SecretsStore{
		KeyPath: t.TempDir() + "/master.key",
	}
	// Should not exist yet
	if _, err := os.Stat(s.KeyPath); !os.IsNotExist(err) {
		t.Fatalf("expected key file to not exist")
	}
	if err := s.loadOrCreateKey(); err != nil {
		t.Fatalf("loadOrCreateKey: %v", err)
	}
	// Should now exist
	if _, err := os.Stat(s.KeyPath); err != nil {
		t.Fatalf("expected key file to exist: %v", err)
	}
	if len(s.masterKey) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(s.masterKey))
	}
}

func TestLoadOrCreateKey_LoadsExistingKey(t *testing.T) {
	s := &SecretsStore{
		KeyPath: t.TempDir() + "/master.key",
	}
	// Create a key file
	if err := s.loadOrCreateKey(); err != nil {
		t.Fatalf("initial create: %v", err)
	}
	origKey := make([]byte, len(s.masterKey))
	copy(origKey, s.masterKey)
	// Now reload
	s2 := &SecretsStore{KeyPath: s.KeyPath}
	if err := s2.loadOrCreateKey(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(s2.masterKey) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(s2.masterKey))
	}
	if string(s2.masterKey) != string(origKey) {
		t.Fatalf("expected loaded key to match original")
	}
}

func TestLoadOrCreateKey_InvalidBase64Fails(t *testing.T) {
	dir := t.TempDir()
	keyPath := dir + "/master.key"
	if err := os.WriteFile(keyPath, []byte("not-base64!"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	s := &SecretsStore{KeyPath: keyPath}
	if err := s.loadOrCreateKey(); err == nil {
		t.Fatal("expected error on invalid base64")
	}
}

func TestWriteMasterKey_WritesBase64(t *testing.T) {
	dir := t.TempDir()
	keyPath := dir + "/master.key"
	s := &SecretsStore{KeyPath: keyPath}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := s.writeMasterKey(key); err != nil {
		t.Fatalf("writeMasterKey: %v", err)
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Should be valid base64 and decodable
	dec, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(dec) != string(key) {
		t.Fatalf("decoded key does not match original")
	}
}
