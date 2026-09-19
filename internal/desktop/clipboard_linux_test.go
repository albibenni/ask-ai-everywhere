//go:build linux

package desktop

import "testing"

func TestChooseClipboardTypePreservesPNGWhenClipboardAlsoOffersText(t *testing.T) {
	types := []string{"text/html", "text/plain", "image/png"}

	if got := chooseClipboardType(types); got != "image/png" {
		t.Fatalf("chooseClipboardType() = %q, want image/png", got)
	}
}

func TestChooseClipboardTypePreservesJPEGWhenClipboardAlsoOffersText(t *testing.T) {
	types := []string{"text/plain", "image/jpeg"}

	if got := chooseClipboardType(types); got != "image/jpeg" {
		t.Fatalf("chooseClipboardType() = %q, want image/jpeg", got)
	}
}

func TestChooseClipboardTypeUsesUTF8PlainTextForTextClipboard(t *testing.T) {
	types := []string{"text/html", "text/plain", "text/plain;charset=utf-8"}

	if got := chooseClipboardType(types); got != "text/plain;charset=utf-8" {
		t.Fatalf("chooseClipboardType() = %q, want text/plain;charset=utf-8", got)
	}
}
