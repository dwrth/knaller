// Package lock provides a node-local file lock for Knaller mutating operations.
package lock

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// ErrBusy is returned when TryLock cannot acquire the lock without blocking.
var ErrBusy = errors.New("lock: busy")

// UnlockFunc releases a lock acquired by TryLock.
type UnlockFunc func() error

// Lock is a non-blocking exclusive flock on a single path.
type Lock struct {
	path string
}

// New returns a Lock for path. Callers typically use a fixed node-local path
// (for example under the Knaller data directory).
func New(path string) *Lock {
	return &Lock{path: path}
}

// TryLock acquires an exclusive flock without blocking.
// On success, the returned UnlockFunc must be called to release the lock.
// If another holder exists, it returns ErrBusy.
func (l *Lock) TryLock() (UnlockFunc, error) {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		_ = f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, ErrBusy
		}
		return nil, err
	}

	unlock := func() error {
		defer f.Close()
		return unix.Flock(int(f.Fd()), unix.LOCK_UN)
	}
	return unlock, nil
}
