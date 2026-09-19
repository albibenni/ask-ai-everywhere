package askai

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeDesktop struct {
	clipboard       string
	clipboardImage  []byte
	selectedText    string
	copyErr         error
	openErr         error
	pasteErr        error
	openedURL       string
	pasted          bool
	replaceProvider string
	composer        string
	notifications   []string
	sleeps          []time.Duration
}

func (f *fakeDesktop) ReadClipboard() (ClipboardItem, error) {
	if f.clipboardImage != nil {
		return ClipboardItem{MIMEType: "image/png", Data: append([]byte(nil), f.clipboardImage...)}, nil
	}
	return textClipboardItem(f.clipboard), nil
}
func (f *fakeDesktop) WriteClipboard(item ClipboardItem) error {
	if strings.HasPrefix(item.MIMEType, "image/") {
		f.clipboard = ""
		f.clipboardImage = append([]byte(nil), item.Data...)
		return nil
	}
	f.clipboard = string(item.Data)
	f.clipboardImage = nil
	return nil
}

func TestRunUsesCopiedImageWhenThereIsNoSelection(t *testing.T) {
	image := []byte("\x89PNG\r\n\x1a\nimage data")
	desktop := &fakeDesktop{clipboardImage: image}

	if err := Run(Config{URL: "https://chatgpt.com/"}, desktop); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if desktop.openedURL != "https://chatgpt.com/" {
		t.Fatalf("opened URL = %q", desktop.openedURL)
	}
	if !desktop.pasted {
		t.Fatal("image was not pasted")
	}
	if !bytes.Equal(desktop.clipboardImage, image) {
		t.Fatalf("clipboard image = %q, want %q", desktop.clipboardImage, image)
	}
	if len(desktop.notifications) != 1 ||
		!strings.Contains(desktop.notifications[0], "clipboard") {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}
func (f *fakeDesktop) CopySelection() error {
	if f.copyErr == nil {
		f.clipboard = f.selectedText
		f.clipboardImage = nil
	}
	return f.copyErr
}
func (f *fakeDesktop) OpenURL(url string) error {
	f.openedURL = url
	return f.openErr
}
func (f *fakeDesktop) ReplaceDraft(provider string) error {
	f.pasted = true
	f.replaceProvider = provider
	f.composer = f.clipboard
	return f.pasteErr
}
func (f *fakeDesktop) Notify(_ string, body string) error {
	f.notifications = append(f.notifications, body)
	return nil
}
func (f *fakeDesktop) Sleep(duration time.Duration) { f.sleeps = append(f.sleeps, duration) }

func TestRunCopiesSelectionOpensFreshTabAndPastesDraft(t *testing.T) {
	desktop := &fakeDesktop{clipboard: "previous", selectedText: "Selected text"}
	config := Config{
		Provider:             "chatgpt",
		URL:                  "https://chatgpt.com/",
		ShortcutReleaseDelay: 200 * time.Millisecond,
		PasteDelay:           1500 * time.Millisecond,
	}

	if err := Run(config, desktop); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if desktop.openedURL != config.URL {
		t.Fatalf("opened URL = %q", desktop.openedURL)
	}
	if !desktop.pasted {
		t.Fatal("selection was not pasted")
	}
	if desktop.replaceProvider != "chatgpt" {
		t.Fatalf("replace provider = %q", desktop.replaceProvider)
	}
	if desktop.clipboard != "Selected text" {
		t.Fatalf("clipboard = %q", desktop.clipboard)
	}
	if len(desktop.notifications) != 0 {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}

func TestRunUsesMostRecentClipboardItemWhenThereIsNoSelection(t *testing.T) {
	desktop := &fakeDesktop{clipboard: "previous"}

	if err := Run(Config{URL: "https://chatgpt.com/"}, desktop); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if desktop.openedURL != "https://chatgpt.com/" {
		t.Fatalf("opened URL = %q", desktop.openedURL)
	}
	if desktop.clipboard != "previous" {
		t.Fatalf("clipboard = %q, want previous", desktop.clipboard)
	}
	if desktop.composer != "previous" {
		t.Fatalf("composer = %q, want previous", desktop.composer)
	}
	if len(desktop.notifications) != 1 ||
		!strings.Contains(desktop.notifications[0], "first") ||
		!strings.Contains(desktop.notifications[0], "clipboard") {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}

func TestRunStopsWhenThereIsNoSelectionAndClipboardIsEmpty(t *testing.T) {
	desktop := &fakeDesktop{clipboard: "  \n"}

	err := Run(Config{URL: "https://chatgpt.com/"}, desktop)
	if !errors.Is(err, ErrNoSelection) {
		t.Fatalf("Run() error = %v, want ErrNoSelection", err)
	}
	if desktop.openedURL != "" {
		t.Fatalf("opened URL = %q", desktop.openedURL)
	}
	if len(desktop.notifications) != 1 {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}

func TestRunLeavesSelectionOnClipboardAndNotifiesWhenPasteFails(t *testing.T) {
	desktop := &fakeDesktop{
		clipboard:    "previous",
		selectedText: "Selected text",
		pasteErr:     errors.New("accessibility denied"),
	}

	err := Run(Config{URL: "https://claude.ai/new"}, desktop)
	if err == nil {
		t.Fatal("Run() unexpectedly succeeded")
	}
	if desktop.clipboard != "Selected text" {
		t.Fatalf("clipboard = %q", desktop.clipboard)
	}
	if len(desktop.notifications) != 1 {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}

func TestRunDoesNotPasteWhenBrowserCannotOpen(t *testing.T) {
	desktop := &fakeDesktop{
		selectedText: "Selected text",
		openErr:      errors.New("browser unavailable"),
	}

	err := Run(Config{URL: "https://gemini.google.com/app"}, desktop)
	if err == nil {
		t.Fatal("Run() unexpectedly succeeded")
	}
	if desktop.pasted {
		t.Fatal("ReplaceDraft() called after OpenURL() failed")
	}
	if desktop.clipboard != "Selected text" {
		t.Fatalf("clipboard = %q", desktop.clipboard)
	}
}

func TestRunReportsSelectionAutomationFailureWithoutOpeningBrowser(t *testing.T) {
	desktop := &fakeDesktop{
		clipboard: "previous",
		copyErr:   errors.New("accessibility denied"),
	}

	err := Run(Config{URL: "https://chatgpt.com/"}, desktop)
	if err == nil {
		t.Fatal("Run() unexpectedly succeeded")
	}
	if desktop.openedURL != "" {
		t.Fatalf("opened URL = %q", desktop.openedURL)
	}
	if len(desktop.notifications) != 1 ||
		!strings.Contains(desktop.notifications[0], "permissions") {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}

func TestRunFromClipboardOpensAndPastesExplicitlyCopiedText(t *testing.T) {
	desktop := &fakeDesktop{clipboard: "Yanked visual selection"}
	config := Config{URL: "https://chatgpt.com/", PasteDelay: 1500 * time.Millisecond}

	if err := RunFromClipboard(config, desktop); err != nil {
		t.Fatalf("RunFromClipboard() error = %v", err)
	}
	if desktop.openedURL != config.URL || !desktop.pasted {
		t.Fatalf("opened URL = %q, pasted = %v", desktop.openedURL, desktop.pasted)
	}
	if desktop.clipboard != "Yanked visual selection" {
		t.Fatalf("clipboard = %q", desktop.clipboard)
	}
}

func TestRunFromClipboardRejectsEmptyClipboardWithoutOpeningBrowser(t *testing.T) {
	desktop := &fakeDesktop{clipboard: "  \n"}

	err := RunFromClipboard(Config{URL: "https://chatgpt.com/"}, desktop)
	if !errors.Is(err, ErrNoSelection) {
		t.Fatalf("RunFromClipboard() error = %v, want ErrNoSelection", err)
	}
	if desktop.openedURL != "" {
		t.Fatalf("opened URL = %q", desktop.openedURL)
	}
	if len(desktop.notifications) != 1 {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}

func TestRunReplacesRestoredComposerDraftWithNewSelection(t *testing.T) {
	desktop := &fakeDesktop{
		clipboard:    "previous selection",
		selectedText: "write plan",
		composer:     `"provider": "chatgpt",`,
	}

	if err := Run(Config{URL: "https://chatgpt.com/"}, desktop); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if desktop.composer != "write plan" {
		t.Fatalf("composer = %q, want only the new selection", desktop.composer)
	}
}
