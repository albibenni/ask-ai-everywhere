//go:build windows

package askai

import (
	"errors"

	"golang.org/x/sys/windows"
)

var ErrAlreadyRunning = errors.New("Ask AI is already opening a chat")

func AcquireRunLock() (func(), error) {
	return acquireRunLock(`Local\AskAIEverywhereRun`)
}

func acquireRunLock(name string) (func(), error) {
	pointer, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateMutex(nil, false, pointer)
	if err != nil {
		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}
	return func() { _ = windows.CloseHandle(handle) }, nil
}
