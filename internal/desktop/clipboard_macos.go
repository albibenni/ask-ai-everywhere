package desktop

import (
	"encoding/base64"
	"fmt"
	"strings"

	"ask-ai-everywhere/internal/askai"
)

// parseMacOSClipboard decodes the MIME type and Base64 data emitted by the
// AppKit clipboard bridge.
func parseMacOSClipboard(output []byte) (askai.ClipboardItem, error) {
	mimeType, encodedData, found := strings.Cut(string(output), "\n")
	if !found || mimeType == "" {
		return askai.ClipboardItem{}, fmt.Errorf("clipboard bridge returned an invalid item")
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedData))
	if err != nil {
		return askai.ClipboardItem{}, fmt.Errorf("decode clipboard data: %w", err)
	}
	return askai.ClipboardItem{MIMEType: mimeType, Data: data}, nil
}

func macOSPasteboardType(mimeType string) string {
	switch mimeType {
	case "image/png":
		return "public.png"
	case "image/tiff":
		return "public.tiff"
	case "image/jpeg":
		return "public.jpeg"
	default:
		return ""
	}
}
