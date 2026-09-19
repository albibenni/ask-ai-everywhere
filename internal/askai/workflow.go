package askai

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNoSelection = errors.New("no text selection was captured")

const (
	capturePollInterval = 25 * time.Millisecond
	capturePollAttempts = 20
)

// ClipboardItem is one clipboard representation that can be restored without
// converting images or other binary content to text.
type ClipboardItem struct {
	MIMEType string
	Data     []byte
}

func textClipboardItem(text string) ClipboardItem {
	return ClipboardItem{MIMEType: "text/plain;charset=utf-8", Data: []byte(text)}
}

func (item ClipboardItem) empty() bool {
	if strings.HasPrefix(item.MIMEType, "text/") || item.MIMEType == "" {
		return strings.TrimSpace(string(item.Data)) == ""
	}
	return len(item.Data) == 0
}

func (item ClipboardItem) equal(other ClipboardItem) bool {
	return item.MIMEType == other.MIMEType && bytes.Equal(item.Data, other.Data)
}

// Desktop contains the operating-system actions needed by the workflow.
type Desktop interface {
	ReadClipboard() (ClipboardItem, error)
	WriteClipboard(ClipboardItem) error
	CopySelection() error
	OpenURL(string) error
	ReplaceDraft(provider string) error
	Notify(title, body string) error
	Sleep(time.Duration)
}

// Run captures selected text, opens a fresh provider tab, and pastes a draft.
func Run(config Config, desktop Desktop) error {
	previousClipboard, _ := desktop.ReadClipboard()
	desktop.Sleep(config.ShortcutReleaseDelay)

	_, err := captureSelection(desktop, previousClipboard)
	if err != nil {
		if errors.Is(err, ErrNoSelection) {
			if !previousClipboard.empty() {
				_ = desktop.Notify("Ask AI", "No text selected. Using the first (most recent) item in the clipboard.")
				return openAndPaste(config, desktop)
			}
			_ = desktop.Notify("Ask AI", "No selected text was found. Nothing was opened.")
		} else {
			_ = desktop.Notify("Ask AI", "Could not capture selected text. Check desktop automation permissions. Nothing was opened.")
		}
		return err
	}

	return openAndPaste(config, desktop)
}

// RunFromClipboard opens content explicitly placed on the clipboard by an
// application integration, such as a Neovim Visual-mode mapping.
func RunFromClipboard(config Config, desktop Desktop) error {
	selectedItem, err := desktop.ReadClipboard()
	if err != nil || selectedItem.empty() {
		_ = desktop.Notify("Ask AI", "No copied content was found. Nothing was opened.")
		return ErrNoSelection
	}
	return openAndPaste(config, desktop)
}

func openAndPaste(config Config, desktop Desktop) error {
	if err := desktop.OpenURL(config.URL); err != nil {
		_ = desktop.Notify("Ask AI", "Could not open the AI chat. The selected content is on your clipboard.")
		return fmt.Errorf("open AI chat: %w", err)
	}

	desktop.Sleep(config.PasteDelay)
	if err := desktop.ReplaceDraft(config.Provider); err != nil {
		_ = desktop.Notify("Ask AI", "Content injection failed. Paste the selected content from your clipboard.")
		return fmt.Errorf("paste selection: %w", err)
	}
	return nil
}

func captureSelection(desktop Desktop, previousClipboard ClipboardItem) (ClipboardItem, error) {
	if err := desktop.CopySelection(); err != nil {
		return ClipboardItem{}, fmt.Errorf("copy selection: %w", err)
	}

	for range capturePollAttempts {
		item, err := desktop.ReadClipboard()
		if err == nil && !item.equal(previousClipboard) {
			if item.empty() {
				_ = desktop.WriteClipboard(previousClipboard)
				return ClipboardItem{}, ErrNoSelection
			}
			return item, nil
		}
		desktop.Sleep(capturePollInterval)
	}

	return ClipboardItem{}, ErrNoSelection
}
