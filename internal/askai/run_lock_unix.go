//go:build darwin || linux

package askai

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

var ErrAlreadyRunning = errors.New("Ask AI is already opening a chat")

// AcquireRunLock prevents one key press repeated by the window manager from
// opening multiple chats and pasting the same content more than once.
func AcquireRunLock() (func(), error) {
	return acquireRunLock(filepath.Join(os.TempDir(), "ask-ai-everywhere.lock"))
}

func acquireRunLock(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
