// ABOUTME: File lock manager for write serialization using flock(2).
// ABOUTME: Provides non-blocking exclusive lock acquisition with fail-fast semantics.

package tl

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// withLock acquires an exclusive non-blocking file lock on lockPath,
// executes fn under the lock, and releases the lock before returning.
// If the lock cannot be acquired immediately, returns ErrLockBusy.
func withLock(lockPath string, fn func() error) error {
	// Open or create the lock file
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Attempt non-blocking exclusive lock
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return ErrLockBusy
		}
		return err
	}

	// Ensure lock is released before returning
	defer func() {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
	}()

	// Execute the function under lock
	return fn()
}
