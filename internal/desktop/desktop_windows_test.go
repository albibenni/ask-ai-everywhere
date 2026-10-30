//go:build windows

package desktop

import (
	"errors"
	"strings"
	"testing"
	"unsafe"

	"ask-ai-everywhere/internal/askai"
)

func TestWindowsInputMatchesNativeABI(t *testing.T) {
	var input windowsInput
	wantSize, wantOffset := uintptr(40), uintptr(8)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		wantSize, wantOffset = 28, 4
	}
	if unsafe.Sizeof(input) != wantSize || unsafe.Offsetof(input.Data) != wantOffset {
		t.Fatalf("INPUT size = %d, data offset = %d; want %d, %d", unsafe.Sizeof(input), unsafe.Offsetof(input.Data), wantSize, wantOffset)
	}
	if unsafe.Alignof(input) != unsafe.Sizeof(uintptr(0)) {
		t.Fatalf("INPUT alignment = %d, want pointer alignment", unsafe.Alignof(input))
	}
}

func TestWindowsDoesNotCopyOverAnUnreadableClipboard(t *testing.T) {
	want := errors.New("screenshot clipboard busy")
	system := &System{clipboardReadErr: want}
	if err := system.CopySelection(); !errors.Is(err, want) {
		t.Fatalf("CopySelection() error = %v, want %v", err, want)
	}
}

func TestWindowsSelectionAttemptPreservesScreenshotForManualPaste(t *testing.T) {
	system := &System{clipboardImage: true}
	if err := system.CopySelection(); err != nil {
		t.Fatalf("image clipboard should bypass simulated Copy: %v", err)
	}
}

func TestWindowsRejectsUnrepresentableTextBeforeChangingClipboard(t *testing.T) {
	// Text encoding fails before any native allocation or clipboard change.
	system := New()
	if err := system.WriteClipboard(askai.ClipboardItem{MIMEType: "text/plain", Data: []byte("before\x00after")}); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("WriteClipboard() error = %v, want NUL rejection", err)
	}
}
