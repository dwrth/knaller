package lock_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/dwrth/knaller/internal/lock"
)

func TestTryLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knaller.lock")
	l := lock.New(path)

	unlock, err := l.TryLock()
	if err != nil {
		t.Fatalf("TryLock() failed: %v", err)
	}

	_, err = l.TryLock()
	if !errors.Is(err, lock.ErrBusy) {
		t.Fatalf("TryLock() while held = %v, want ErrBusy", err)
	}

	if err := unlock(); err != nil {
		t.Fatalf("unlock() failed: %v", err)
	}

	unlock, err = l.TryLock()
	if err != nil {
		t.Fatalf("TryLock() after unlock failed: %v", err)
	}
	if err := unlock(); err != nil {
		t.Fatalf("second unlock() failed: %v", err)
	}
}
