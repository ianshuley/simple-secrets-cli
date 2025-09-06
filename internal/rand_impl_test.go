package internal

import (
	"errors"
	"testing"
)

func TestRotateMasterKey_RandFailure(t *testing.T) {
	s := newTempStore(t)
	err := s.Put("a", "b")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	orig := randRead
	randRead = func(b []byte) (int, error) { return 0, errors.New("rng fail") }
	defer func() { randRead = orig }()

	err = s.RotateMasterKey("")
	if err == nil {
		t.Fatal("expected rotate to fail on RNG error")
	}
}
