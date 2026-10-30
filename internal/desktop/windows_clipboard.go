package desktop

// Clipboard codecs stay independent of Win32 so image priority and Unicode
// handling can be checked on every development platform.
import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"

	"ask-ai-everywhere/internal/askai"
)

const (
	windowsCFDIB         = 8
	windowsCFUnicodeText = 13
	windowsCFDIBV5       = 17
)

func readWindowsClipboard(png, mimePNG uint32, available func(uint32) bool, read func(uint32) ([]byte, error)) (askai.ClipboardItem, error) {
	formats := []struct {
		id   uint32
		mime string
	}{
		{png, "image/png"},
		{mimePNG, "image/png"},
		{windowsCFDIBV5, "image/x-windows-dibv5"},
		{windowsCFDIB, "image/x-windows-dib"},
		{windowsCFUnicodeText, "text/plain;charset=utf-8"},
	}
	for _, format := range formats {
		if format.id == 0 || !available(format.id) {
			continue
		}
		data, err := read(format.id)
		if err != nil {
			return askai.ClipboardItem{}, err
		}
		if format.id == windowsCFUnicodeText {
			if len(data)%2 != 0 {
				return askai.ClipboardItem{}, fmt.Errorf("clipboard text has an odd UTF-16 byte length")
			}
			units := make([]uint16, 0, len(data)/2)
			for offset := 0; offset < len(data); offset += 2 {
				unit := binary.LittleEndian.Uint16(data[offset:])
				if unit == 0 {
					break
				}
				units = append(units, unit)
			}
			data = []byte(string(utf16.Decode(units)))
		}
		return askai.ClipboardItem{MIMEType: format.mime, Data: data}, nil
	}
	return askai.ClipboardItem{}, nil
}

func windowsClipboardText(text []byte) ([]byte, error) {
	if bytes.IndexByte(text, 0) >= 0 {
		return nil, fmt.Errorf("clipboard text contains a NUL character")
	}
	units := utf16.Encode([]rune(string(text)))
	data := make([]byte, (len(units)+1)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(data[index*2:], unit)
	}
	return data, nil
}
