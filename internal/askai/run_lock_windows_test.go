//go:build windows

package askai

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestRunLockRejectsConcurrentLaunchAndAllowsRetry(t *testing.T) {
	name := fmt.Sprintf(`Local\AskAITest-%d-%s`, os.Getpid(), t.Name())
	release, err := acquireRunLock(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { release() })
	if _, err := acquireRunLock(name); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second lock error = %v, want ErrAlreadyRunning", err)
	}
	release()
	// Avoid closing a recycled handle again during test cleanup.
	release = func() {}
	retryRelease, err := acquireRunLock(name)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	retryRelease()
}
