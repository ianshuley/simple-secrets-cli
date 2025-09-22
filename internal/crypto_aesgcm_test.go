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
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, AES256KeySize)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}

	payloads := [][]byte{
		nil,
		[]byte(""),
		[]byte("x"),
		bytes.Repeat([]byte{0}, 32),
		bytes.Repeat([]byte("ab"), 1024),
	}

	for i, p := range payloads {
		ct, err := encrypt(key, p)
		if err != nil {
			t.Fatalf("case %d: encrypt: %v", i, err)
		}

		pt, err := decrypt(key, ct)
		if err != nil {
			t.Fatalf("case %d: decrypt: %v", i, err)
		}

		if !bytes.Equal(p, pt) {
			t.Fatalf("case %d: round-trip mismatch", i)
		}
	}
}

func TestDecrypt_TamperFails(t *testing.T) {
	key := make([]byte, AES256KeySize)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}

	ct, err := encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	raw, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		t.Fatal(err)
	}

	// flip a byte in the ciphertext body (not the nonce prefix)
	if len(raw) < 17 {
		t.Skip("ciphertext too short to tamper safely")
	}
	raw[len(raw)-1] ^= 0xFF

	tampered := base64.StdEncoding.EncodeToString(raw)
	_, err = decrypt(key, tampered)
	if err == nil {
		t.Fatal("expected auth failure on tampered ciphertext")
	}
}

func FuzzEncryptDecrypt(f *testing.F) {
	key := make([]byte, AES256KeySize)
	_, _ = rand.Read(key)

	seeds := [][]byte{nil, []byte("a"), bytes.Repeat([]byte("abc"), 100)}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, p []byte) {
		ct, err := encrypt(key, p)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		pt, err := decrypt(key, ct)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if !bytes.Equal(p, pt) {
			t.Fatalf("mismatch")
		}
	})
}
