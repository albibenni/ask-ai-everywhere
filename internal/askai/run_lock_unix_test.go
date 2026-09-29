//go:build darwin || linux

package askai

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestRunLockRejectsConcurrentLaunch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ask-ai.lock")
	release, err := acquireRunLock(path)
	if err != nil {
		t.Fatalf("acquireRunLock() error = %v", err)
	}
	defer release()

	_, err = acquireRunLock(path)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second acquireRunLock() error = %v, want ErrAlreadyRunning", err)
	}
}
