package desktop

import (
	"bytes"
	"testing"
)

func TestParseMacOSClipboardImagePrefersPNG(t *testing.T) {
	image := []byte("\x89PNG\r\n\x1a\nimage data")
	output := append([]byte("image/png\n"), []byte("iVBORw0KGgppbWFnZSBkYXRh\n")...)

	item, err := parseMacOSClipboard(output)
	if err != nil {
		t.Fatalf("parseMacOSClipboard() error = %v", err)
	}
	if item.MIMEType != "image/png" {
		t.Fatalf("MIMEType = %q, want image/png", item.MIMEType)
	}
	if !bytes.Equal(item.Data, image) {
		t.Fatalf("Data = %q, want %q", item.Data, image)
	}
}

func TestParseMacOSClipboardText(t *testing.T) {
	item, err := parseMacOSClipboard([]byte("text/plain;charset=utf-8\naGVsbG8=\n"))
	if err != nil {
		t.Fatalf("parseMacOSClipboard() error = %v", err)
	}
	if item.MIMEType != "text/plain;charset=utf-8" || string(item.Data) != "hello" {
		t.Fatalf("item = %#v, want UTF-8 text hello", item)
	}
}

func TestMacOSPasteboardTypeMapsSupportedImageMIMETypes(t *testing.T) {
	tests := map[string]string{
		"image/png":  "public.png",
		"image/tiff": "public.tiff",
		"image/jpeg": "public.jpeg",
	}
	for mimeType, want := range tests {
		if got := macOSPasteboardType(mimeType); got != want {
			t.Errorf("macOSPasteboardType(%q) = %q, want %q", mimeType, got, want)
		}
	}
}
