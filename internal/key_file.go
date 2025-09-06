package internal

import (
	"crypto/rand"
	"encoding/base64"
	"os"
)

// loadOrCreateKey sets s.masterKey; creates the file if missing.
func (s *SecretsStore) loadOrCreateKey() error {
	if _, err := os.Stat(s.KeyPath); os.IsNotExist(err) {
		key := make([]byte, 32) // AES-256
		if _, err := rand.Read(key); err != nil {
			return err
		}
		if err := s.writeMasterKey(key); err != nil {
			return err
		}
		s.masterKey = key
		return nil
	}

	data, err := os.ReadFile(s.KeyPath)
	if err != nil {
		return err
	}

	key, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}

	s.masterKey = key
	return nil
}

// writeMasterKey overwrites the key file (0600).
func (s *SecretsStore) writeMasterKey(newKey []byte) error {
	enc := base64.StdEncoding.EncodeToString(newKey)
	return os.WriteFile(s.KeyPath, []byte(enc), 0600)
}
