package askai

import (
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

// Desktop contains the operating-system actions needed by the workflow.
type Desktop interface {
	ReadClipboard() (string, error)
	WriteClipboard(string) error
	CopySelection() error
	OpenURL(string) error
	ReplaceDraft() error
	Notify(title, body string) error
	Sleep(time.Duration)
}

// Run captures selected text, opens a fresh provider tab, and pastes a draft.
func Run(config Config, desktop Desktop) error {
	previousClipboard, _ := desktop.ReadClipboard()
	desktop.Sleep(config.ShortcutReleaseDelay)

	selectedText, err := captureSelection(desktop, previousClipboard)
	if err != nil {
		if errors.Is(err, ErrNoSelection) {
			if strings.TrimSpace(previousClipboard) != "" {
				_ = desktop.Notify("Ask AI", "No text selected. Using the first (most recent) item in the clipboard.")
				return openAndPaste(config, desktop, previousClipboard)
			}
			_ = desktop.Notify("Ask AI", "No selected text was found. Nothing was opened.")
		} else {
			_ = desktop.Notify("Ask AI", "Could not capture selected text. Check desktop automation permissions. Nothing was opened.")
		}
		return err
	}

	return openAndPaste(config, desktop, selectedText)
}

// RunFromClipboard opens text explicitly placed on the clipboard by an
// application integration, such as a Neovim Visual-mode mapping.
func RunFromClipboard(config Config, desktop Desktop) error {
	selectedText, err := desktop.ReadClipboard()
	if err != nil || strings.TrimSpace(selectedText) == "" {
		_ = desktop.Notify("Ask AI", "No copied text was found. Nothing was opened.")
		return ErrNoSelection
	}
	return openAndPaste(config, desktop, selectedText)
}

func openAndPaste(config Config, desktop Desktop, selectedText string) error {
	// Keep the exact selection available even if browser automation fails.
	if err := desktop.WriteClipboard(selectedText); err != nil {
		_ = desktop.Notify("Ask AI", "Could not copy the selected text to the clipboard.")
		return fmt.Errorf("copy selection to clipboard: %w", err)
	}

	if err := desktop.OpenURL(config.URL); err != nil {
		_ = desktop.Notify("Ask AI", "Could not open the AI chat. The selected text is on your clipboard.")
		return fmt.Errorf("open AI chat: %w", err)
	}

	desktop.Sleep(config.PasteDelay)
	if err := desktop.ReplaceDraft(); err != nil {
		_ = desktop.Notify("Ask AI", "Text injection failed. Paste the selected text from your clipboard.")
		return fmt.Errorf("paste selection: %w", err)
	}
	return nil
}

func captureSelection(desktop Desktop, previousClipboard string) (string, error) {
	sentinel := fmt.Sprintf("ask-ai-selection-%d", time.Now().UnixNano())
	if err := desktop.WriteClipboard(sentinel); err != nil {
		return "", fmt.Errorf("prepare clipboard: %w", err)
	}
	if err := desktop.CopySelection(); err != nil {
		_ = desktop.WriteClipboard(previousClipboard)
		return "", fmt.Errorf("copy selection: %w", err)
	}

	for range capturePollAttempts {
		text, err := desktop.ReadClipboard()
		if err == nil && text != sentinel {
			if strings.TrimSpace(text) == "" {
				_ = desktop.WriteClipboard(previousClipboard)
				return "", ErrNoSelection
			}
			return text, nil
		}
		desktop.Sleep(capturePollInterval)
	}

	_ = desktop.WriteClipboard(previousClipboard)
	return "", ErrNoSelection
}
