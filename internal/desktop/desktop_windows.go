//go:build windows

package desktop

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"ask-ai-everywhere/internal/askai"
	"golang.org/x/sys/windows"
)

type System struct {
	clipboardReadErr error
}

var _ askai.Desktop = (*System)(nil)

func New() *System { return &System{} }

var (
	sendInput        = windowsUser32.NewProc("SendInput")
	getAsyncKeyState = windowsUser32.NewProc("GetAsyncKeyState")
)

// INPUT's union must be as large as MOUSEINPUT even for keyboard events.
// Its alignment differs between 32-bit and 64-bit Windows.
type windowsInput struct {
	Type uint32
	Data windowsKeyboardInput
}

type windowsKeyboardInput struct {
	VirtualKey uint16
	ScanCode   uint16
	Flags      uint32
	Time       uint32
	ExtraInfo  uintptr
	_          [8]byte // Pad KEYBDINPUT to the size of the MOUSEINPUT union member.
}

func (s *System) CopySelection() error {
	// The shared workflow tolerates a failed initial clipboard read. On Windows
	// do not risk overwriting an unreadable screenshot with a simulated Copy.
	if s.clipboardReadErr != nil {
		return fmt.Errorf("could not inspect clipboard safely: %w", s.clipboardReadErr)
	}
	source := windows.GetForegroundWindow()
	if err := waitForWindowsModifiers(); err != nil {
		return err
	}
	if source == 0 || source != windows.GetForegroundWindow() {
		return fmt.Errorf("source application lost focus before Copy")
	}
	// Ctrl+Insert copies selected text in Windows Terminal and classic consoles
	// without risking Ctrl+C interrupting a running command.
	class := windowsForegroundClass()
	if class == "ConsoleWindowClass" || class == "CASCADIA_HOSTING_WINDOW_CLASS" || class == "mintty" {
		return windowsControlKey(0x2D) // VK_INSERT
	}
	return windowsControlKey('C')
}

func (s *System) OpenURL(url string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	target, err := windows.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	// ShellExecute may delegate to COM shell extensions. Keep initialization
	// and the call on the same Windows thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	switch err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err {
	case nil, syscall.Errno(1): // S_OK and S_FALSE both require CoUninitialize.
		defer windows.CoUninitialize()
	case syscall.Errno(windows.RPC_E_CHANGED_MODE):
		// COM is already initialized on this thread in another apartment model.
	default:
		return fmt.Errorf("initialize Windows shell: %w", err)
	}
	return windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL)
}

func (s *System) ReplaceDraft(provider, mimeType string) error {
	if strings.HasPrefix(mimeType, "image/") {
		// Manual paste avoids changing screenshot formats or depending on browser
		// attachment focus. A notification failure must not turn this into a
		// failed draft: the chat is open and the image is still on the clipboard.
		_ = s.Notify("Ask AI", "Chat opened. Press Ctrl+V in the chat to paste your screenshot.")
		return nil
	}
	if err := waitForWindowsModifiers(); err != nil {
		return err
	}
	target := windowsFocusedBrowser()
	if target == 0 {
		return fmt.Errorf("browser is not focused; paste the clipboard into the chat manually")
	}
	if provider == "kimi" {
		if err := windowsControlKey('K'); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	if target != windows.GetForegroundWindow() {
		return fmt.Errorf("browser lost focus before Select All; paste manually")
	}
	if err := windowsControlKey('A'); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	// Check again rather than pasting into another application if focus changed.
	if target != windows.GetForegroundWindow() {
		return fmt.Errorf("browser lost focus; paste the clipboard into the chat manually")
	}
	return windowsControlKey('V')
}

func (s *System) Sleep(duration time.Duration) { time.Sleep(duration) }

func windowsForegroundClass() string {
	var class [256]uint16
	count, err := windows.GetClassName(windows.GetForegroundWindow(), &class[0], int32(len(class)))
	if err != nil || count == 0 {
		return ""
	}
	return windows.UTF16ToString(class[:count])
}

func windowsFocusedBrowser() windows.HWND {
	target := windows.GetForegroundWindow()
	if target == 0 {
		return 0
	}
	var processID uint32
	if _, err := windows.GetWindowThreadProcessId(target, &processID); err != nil {
		return 0
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, processID)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(process)
	var path [32768]uint16
	size := uint32(len(path))
	if err := windows.QueryFullProcessImageName(process, 0, &path[0], &size); err != nil {
		return 0
	}
	if !windowsBrowserExecutable(filepath.Base(windows.UTF16ToString(path[:size]))) {
		return 0
	}
	switch windowsForegroundClass() {
	case "Chrome_WidgetWin_1", "MozillaWindowClass", "OperaWindowClass":
		if target == windows.GetForegroundWindow() {
			return target
		}
	}
	return 0
}

func waitForWindowsModifiers() error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		held := false
		for _, key := range []uintptr{0x10, 0x11, 0x12, 0x5B, 0x5C} { // Shift, Ctrl, Alt, Win
			state, _, _ := getAsyncKeyState.Call(key)
			held = held || state&0x8000 != 0
		}
		if !held {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("shortcut modifiers are still held; release them and try again")
}

func windowsControlKey(key uint16) error {
	input := make([]windowsInput, 4)
	for index, code := range []uint16{0x11, key, key, 0x11} {
		input[index].Type = 1 // INPUT_KEYBOARD
		input[index].Data.VirtualKey = code
		var flags uint32
		if code == 0x2D {
			flags |= 1 // KEYEVENTF_EXTENDEDKEY for Insert
		}
		if index >= 2 {
			flags |= 2 // KEYEVENTF_KEYUP
		}
		input[index].Data.Flags = flags
	}
	count, _, err := sendInput.Call(uintptr(len(input)), uintptr(unsafe.Pointer(&input[0])), unsafe.Sizeof(input[0]))
	if count != uintptr(len(input)) {
		// Best-effort cleanup if Windows accepted only part of the chord.
		sendInput.Call(2, uintptr(unsafe.Pointer(&input[2])), unsafe.Sizeof(input[0]))
		return fmt.Errorf("SendInput accepted %d of %d events (check application elevation): %w", count, len(input), err)
	}
	return nil
}
