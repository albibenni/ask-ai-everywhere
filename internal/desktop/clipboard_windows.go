//go:build windows

package desktop

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"ask-ai-everywhere/internal/askai"
	"golang.org/x/sys/windows"
)

var (
	windowsUser32              = windows.NewLazySystemDLL("user32.dll")
	windowsKernel32            = windows.NewLazySystemDLL("kernel32.dll")
	openClipboard              = windowsUser32.NewProc("OpenClipboard")
	closeClipboard             = windowsUser32.NewProc("CloseClipboard")
	getClipboardData           = windowsUser32.NewProc("GetClipboardData")
	setClipboardData           = windowsUser32.NewProc("SetClipboardData")
	emptyClipboard             = windowsUser32.NewProc("EmptyClipboard")
	isClipboardFormatAvailable = windowsUser32.NewProc("IsClipboardFormatAvailable")
	registerClipboardFormat    = windowsUser32.NewProc("RegisterClipboardFormatW")
	createWindowEx             = windowsUser32.NewProc("CreateWindowExW")
	destroyWindow              = windowsUser32.NewProc("DestroyWindow")
	globalAlloc                = windowsKernel32.NewProc("GlobalAlloc")
	globalFree                 = windowsKernel32.NewProc("GlobalFree")
	globalLock                 = windowsKernel32.NewProc("GlobalLock")
	globalUnlock               = windowsKernel32.NewProc("GlobalUnlock")
	globalSize                 = windowsKernel32.NewProc("GlobalSize")
	moveMemory                 = windows.NewLazySystemDLL("ntdll.dll").NewProc("RtlMoveMemory")
)

func (s *System) ReadClipboard() (item askai.ClipboardItem, err error) {
	err = withWindowsClipboard(false, func() error {
		item, err = readOpenWindowsClipboard()
		return err
	})
	s.clipboardReadErr = err
	s.clipboardImage = err == nil && strings.HasPrefix(item.MIMEType, "image/") && len(item.Data) != 0
	return item, err
}

func readOpenWindowsClipboard() (askai.ClipboardItem, error) {
	format := func(name string) uint32 {
		pointer, _ := windows.UTF16PtrFromString(name)
		id, _, _ := registerClipboardFormat.Call(uintptr(unsafe.Pointer(pointer)))
		return uint32(id)
	}
	return readWindowsClipboard(format("PNG"), format("image/png"), func(id uint32) bool {
		available, _, _ := isClipboardFormatAvailable.Call(uintptr(id))
		return available != 0
	}, func(id uint32) ([]byte, error) {
		handle, _, err := getClipboardData.Call(uintptr(id))
		if handle == 0 {
			return nil, fmt.Errorf("GetClipboardData: %w", err)
		}
		size, _, err := globalSize.Call(handle)
		if size == 0 || size > 1<<30 {
			return nil, fmt.Errorf("invalid clipboard size %d: %w", size, err)
		}
		pointer, _, err := globalLock.Call(handle)
		if pointer == 0 {
			return nil, fmt.Errorf("GlobalLock: %w", err)
		}
		defer globalUnlock.Call(handle)
		data := make([]byte, int(size))
		moveMemory.Call(uintptr(unsafe.Pointer(&data[0])), pointer, size)
		runtime.KeepAlive(data)
		return data, nil
	})
}

func (s *System) WriteClipboard(item askai.ClipboardItem) error {
	if strings.HasPrefix(item.MIMEType, "image/") {
		// The shared workflow refreshes images before injection. Windows uses
		// manual image paste: retain ALL original native formats instead of
		// emptying the clipboard and publishing just one representation.
		return withWindowsClipboard(false, func() error {
			current, err := readOpenWindowsClipboard()
			if err != nil {
				return err
			}
			if current.MIMEType != item.MIMEType || !bytes.Equal(current.Data, item.Data) {
				return fmt.Errorf("clipboard image changed; copy the screenshot again and paste manually")
			}
			return nil
		})
	}
	if item.MIMEType != "" && !strings.HasPrefix(item.MIMEType, "text/") {
		return fmt.Errorf("unsupported clipboard type %q", item.MIMEType)
	}
	data, err := windowsClipboardText(item.Data)
	if err != nil {
		return err
	}
	// Allocate before clearing the clipboard, so allocation failures preserve it.
	handle, _, err := globalAlloc.Call(0x0002, uintptr(len(data))) // GMEM_MOVEABLE
	if handle == 0 {
		return fmt.Errorf("GlobalAlloc: %w", err)
	}
	owned := true
	defer func() {
		if owned {
			globalFree.Call(handle)
		}
	}()
	pointer, _, err := globalLock.Call(handle)
	if pointer == 0 {
		return fmt.Errorf("GlobalLock: %w", err)
	}
	moveMemory.Call(pointer, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
	runtime.KeepAlive(data)
	globalUnlock.Call(handle)
	return withWindowsClipboard(true, func() error {
		if ok, _, err := emptyClipboard.Call(); ok == 0 {
			return fmt.Errorf("EmptyClipboard: %w", err)
		}
		if result, _, err := setClipboardData.Call(windowsCFUnicodeText, handle); result == 0 {
			return fmt.Errorf("SetClipboardData: %w", err)
		}
		owned = false // Windows owns the allocation after SetClipboardData.
		return nil
	})
}

func withWindowsClipboard(write bool, action func() error) error {
	// Clipboard ownership and the invisible owner window belong to this thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var owner uintptr
	if write {
		class, _ := windows.UTF16PtrFromString("STATIC")
		var err error
		owner, _, err = createWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
		if owner == 0 {
			return fmt.Errorf("create clipboard owner: %w", err)
		}
		defer destroyWindow.Call(owner)
	}
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		if ok, _, err := openClipboard.Call(owner); ok != 0 {
			defer closeClipboard.Call()
			return action()
		} else {
			lastErr = err
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("clipboard is busy: %w", lastErr)
}
