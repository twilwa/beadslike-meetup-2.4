// ABOUTME: File lock manager for write serialization on Windows using LockFileEx.
// ABOUTME: Provides non-blocking exclusive lock acquisition with fail-fast semantics.

//go:build windows

package tl

import (
	"os"

	"golang.org/x/sys/windows"
)

func withLock(lockPath string, fn func() error) error {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	ol := new(windows.Overlapped)
	err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
	if err != nil {
		if err == windows.ERROR_LOCK_VIOLATION {
			return ErrLockBusy
		}
		return err
	}
	defer windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)

	return fn()
}
