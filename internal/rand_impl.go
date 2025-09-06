package internal

import "crypto/rand"

func randReadImpl(b []byte) (int, error) { return rand.Read(b) }
