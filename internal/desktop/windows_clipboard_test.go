package desktop

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestWindowsClipboardPrefersScreenshotOverText(t *testing.T) {
	for _, format := range []uint32{50000, 50001, 17, 8} {
		t.Run(fmt.Sprintf("format_%d", format), func(t *testing.T) {
			image := []byte{1, 2, 3, 0, 255}
			item, err := readWindowsClipboard(50000, 50001,
				func(id uint32) bool { return id == format || id == 13 },
				func(id uint32) ([]byte, error) {
					if id == 13 {
						t.Fatal("screenshot was read as text")
					}
					return image, nil
				})
			if err != nil || !bytes.Equal(item.Data, image) || !strings.HasPrefix(item.MIMEType, "image/") {
				t.Fatalf("clipboard = %+v, error = %v", item, err)
			}
		})
	}
}

func TestWindowsClipboardEmptyDoesNotReadAnUnavailableFormat(t *testing.T) {
	item, err := readWindowsClipboard(0, 0,
		func(uint32) bool { return false },
		func(uint32) ([]byte, error) {
			t.Fatal("read unavailable clipboard format")
			return nil, nil
		})
	if err != nil || len(item.Data) != 0 {
		t.Fatalf("clipboard = %+v, error = %v", item, err)
	}
}

func TestWindowsClipboardDecodesUnicodeAndStopsAtTerminator(t *testing.T) {
	text := "Caffè ☕ — 你好 👋\r\nsecond line"
	data, err := windowsClipboardText([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, 0x78, 0)
	item, err := readWindowsClipboard(0, 0,
		func(id uint32) bool { return id == 13 },
		func(uint32) ([]byte, error) { return data, nil })
	if err != nil || string(item.Data) != text || item.MIMEType != "text/plain;charset=utf-8" {
		t.Fatalf("clipboard = %+v, error = %v", item, err)
	}
	if binary.LittleEndian.Uint16(data[len(data)-4:]) != 0 {
		t.Fatal("text has no UTF-16 terminator")
	}
}

func TestWindowsClipboardDoesNotFallBackToTextWhenScreenshotIsBusy(t *testing.T) {
	want := errors.New("clipboard image not ready")
	_, err := readWindowsClipboard(50000, 0,
		func(id uint32) bool { return id == 50000 || id == 13 },
		func(uint32) ([]byte, error) { return nil, want })
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestWindowsClipboardRejectsMalformedText(t *testing.T) {
	_, err := readWindowsClipboard(0, 0,
		func(id uint32) bool { return id == 13 },
		func(uint32) ([]byte, error) { return []byte{1}, nil })
	if err == nil {
		t.Fatal("accepted odd-length UTF-16 clipboard data")
	}
	if _, err := windowsClipboardText([]byte("before\x00after")); err == nil {
		t.Fatal("accepted text that would be silently truncated")
	}
}
