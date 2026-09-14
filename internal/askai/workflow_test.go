package askai

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeDesktop struct {
	clipboard     string
	selectedText  string
	copyErr       error
	openErr       error
	pasteErr      error
	openedURL     string
	pasted        bool
	notifications []string
	sleeps        []time.Duration
}

func (f *fakeDesktop) ReadClipboard() (string, error) { return f.clipboard, nil }
func (f *fakeDesktop) WriteClipboard(text string) error {
	f.clipboard = text
	return nil
}
func (f *fakeDesktop) CopySelection() error {
	if f.copyErr == nil {
		f.clipboard = f.selectedText
	}
	return f.copyErr
}
func (f *fakeDesktop) OpenURL(url string) error {
	f.openedURL = url
	return f.openErr
}
func (f *fakeDesktop) Paste() error {
	f.pasted = true
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
	if desktop.clipboard != "Selected text" {
		t.Fatalf("clipboard = %q", desktop.clipboard)
	}
	if len(desktop.notifications) != 0 {
		t.Fatalf("notifications = %v", desktop.notifications)
	}
}

func TestRunStopsAndRestoresClipboardWhenThereIsNoSelection(t *testing.T) {
	desktop := &fakeDesktop{clipboard: "previous"}

	err := Run(Config{URL: "https://chatgpt.com/"}, desktop)
	if !errors.Is(err, ErrNoSelection) {
		t.Fatalf("Run() error = %v, want ErrNoSelection", err)
	}
	if desktop.openedURL != "" {
		t.Fatalf("opened URL = %q", desktop.openedURL)
	}
	if desktop.clipboard != "previous" {
		t.Fatalf("clipboard = %q, want previous", desktop.clipboard)
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
		t.Fatal("Paste() called after OpenURL() failed")
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
