//go:build linux

package desktop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"ask-ai-everywhere/internal/askai"
)

type System struct{}

func New() *System { return &System{} }

func (s *System) ReadClipboard() (askai.ClipboardItem, error) {
	typesOutput, err := exec.Command("wl-paste", "--list-types").Output()
	if err != nil {
		return askai.ClipboardItem{}, err
	}
	mimeType := chooseClipboardType(strings.Fields(string(typesOutput)))
	if mimeType == "" {
		return askai.ClipboardItem{}, fmt.Errorf("clipboard has no available MIME type")
	}
	data, err := exec.Command("wl-paste", "--no-newline", "--type", mimeType).Output()
	return askai.ClipboardItem{MIMEType: mimeType, Data: data}, err
}

func (s *System) WriteClipboard(item askai.ClipboardItem) error {
	mimeType := item.MIMEType
	if mimeType == "" {
		mimeType = "text/plain;charset=utf-8"
	}
	command := exec.Command("wl-copy", "--type", mimeType)
	command.Stdin = bytes.NewReader(item.Data)
	return command.Run()
}

func chooseClipboardType(types []string) string {
	for _, mimeType := range types {
		if mimeType == "image/png" {
			return mimeType
		}
	}
	for _, mimeType := range types {
		if strings.HasPrefix(mimeType, "image/") {
			return mimeType
		}
	}
	for _, preferred := range []string{"text/plain;charset=utf-8", "text/plain"} {
		for _, mimeType := range types {
			if mimeType == preferred {
				return mimeType
			}
		}
	}
	for _, mimeType := range types {
		if strings.HasPrefix(mimeType, "text/") {
			return mimeType
		}
	}
	if len(types) > 0 {
		return types[0]
	}
	return ""
}

func (s *System) CopySelection() error {
	window, _ := exec.Command("hyprctl", "activewindow", "-j").Output()
	key := copyKey(window)
	if err := exec.Command("hyprctl", "dispatch", shortcutState(key, "down")).Run(); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return exec.Command("hyprctl", "dispatch", shortcutState(key, "up")).Run()
}

func (s *System) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Run()
}

func (s *System) ReplaceDraft(provider string) error {
	return exec.Command("wtype", replaceDraftArguments(provider)...).Run()
}

func (s *System) Notify(title, body string) error {
	return exec.Command("notify-send", "--app-name=Ask AI", title, body).Run()
}

func (s *System) Sleep(duration time.Duration) { time.Sleep(duration) }

func copyKey(windowJSON []byte) string {
	if activeWindowIsTerminal(windowJSON) {
		return "Insert"
	}
	return "C"
}

func shortcutState(key, state string) string {
	return fmt.Sprintf(`hl.dsp.send_key_state({ mods = "CTRL", key = %q, state = %q })`, key, state)
}

func replaceDraftArguments(provider string) []string {
	arguments := make([]string, 0, 16)
	if provider == "kimi" {
		arguments = append(arguments, "-M", "ctrl", "-k", "k", "-m", "ctrl", "-s", "100")
	}
	return append(arguments, "-M", "ctrl", "-k", "a", "-s", "50", "-k", "v", "-m", "ctrl")
}

func activeWindowIsTerminal(windowJSON []byte) bool {
	var window struct {
		Class string   `json:"class"`
		Tags  []string `json:"tags"`
	}
	if err := json.Unmarshal(windowJSON, &window); err != nil {
		return false
	}
	for _, tag := range window.Tags {
		if strings.TrimSuffix(tag, "*") == "terminal" {
			return true
		}
	}

	class := window.Class
	return class == "Alacritty" || class == "kitty" ||
		class == "com.mitchellh.ghostty" || class == "foot" ||
		class == "org.codeberg.dnkl.foot" || class == "wezterm" ||
		strings.HasPrefix(class, "org.omarchy.") || strings.HasPrefix(class, "TUI.")
}
