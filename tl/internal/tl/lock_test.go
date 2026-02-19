// ABOUTME: Tests for file lock manager.
// ABOUTME: Validates lock acquisition, release, contention, and file creation.

package tl

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestLockAcquireRelease verifies that withLock acquires and releases the lock.
func TestLockAcquireRelease(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	executed := false
	err := withLock(lockPath, func() error {
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed)

	// Verify lock file was created
	_, err = os.Stat(lockPath)
	assert.NoError(t, err)

	// Verify we can acquire the lock again (it was released)
	err = withLock(lockPath, func() error {
		return nil
	})
	assert.NoError(t, err)
}

// TestLockContention verifies that concurrent lock attempts fail with ErrLockBusy.
func TestLockContention(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "contention.lock")

	var wg sync.WaitGroup
	var goroutine2Err error
	var goroutine2Executed bool

	// Goroutine 1: Hold the lock for 100ms
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := withLock(lockPath, func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		assert.NoError(t, err)
	}()

	// Give goroutine 1 time to acquire the lock
	time.Sleep(10 * time.Millisecond)

	// Goroutine 2: Try to acquire the lock immediately (should fail)
	wg.Add(1)
	go func() {
		defer wg.Done()
		goroutine2Err = withLock(lockPath, func() error {
			goroutine2Executed = true
			return nil
		})
	}()

	// Give goroutine 2 time to attempt the lock
	time.Sleep(10 * time.Millisecond)

	// Goroutine 2 should have failed with ErrLockBusy
	assert.True(t, errors.Is(goroutine2Err, ErrLockBusy))
	assert.False(t, goroutine2Executed)

	// Wait for goroutine 1 to finish
	wg.Wait()

	// Now goroutine 2 should succeed if we try again
	err := withLock(lockPath, func() error {
		return nil
	})
	assert.NoError(t, err)
}

// TestLockCreatesFile verifies that withLock creates the lock file if it doesn't exist.
func TestLockCreatesFile(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "nonexistent", "test.lock")

	// Ensure parent directory exists
	err := os.MkdirAll(filepath.Dir(lockPath), 0755)
	assert.NoError(t, err)

	// Lock file should not exist yet
	_, err = os.Stat(lockPath)
	assert.True(t, os.IsNotExist(err))

	// Call withLock
	err = withLock(lockPath, func() error {
		return nil
	})
	assert.NoError(t, err)

	// Lock file should now exist
	_, err = os.Stat(lockPath)
	assert.NoError(t, err)
}

// TestLockFunctionError verifies that errors from fn are propagated.
func TestLockFunctionError(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "error.lock")

	testErr := errors.New("test error")
	err := withLock(lockPath, func() error {
		return testErr
	})

	assert.Equal(t, testErr, err)
}
