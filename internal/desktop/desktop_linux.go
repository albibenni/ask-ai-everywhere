//go:build linux

package desktop

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

type System struct{}

func New() *System { return &System{} }

func (s *System) ReadClipboard() (string, error) {
	output, err := exec.Command("wl-paste", "--no-newline", "--type", "text").Output()
	return string(output), err
}

func (s *System) WriteClipboard(text string) error {
	command := exec.Command("wl-copy", "--type", "text/plain;charset=utf-8")
	command.Stdin = bytes.NewBufferString(text)
	return command.Run()
}

func (s *System) CopySelection() error {
	window, _ := exec.Command("hyprctl", "activewindow", "-j").Output()
	return exec.Command("wtype", copyArguments(window)...).Run()
}

func (s *System) OpenURL(url string) error {
	return exec.Command("xdg-open", url).Run()
}

func (s *System) Paste() error {
	return exec.Command("wtype", "-M", "ctrl", "v", "-m", "ctrl").Run()
}

func (s *System) Notify(title, body string) error {
	return exec.Command("notify-send", "--app-name=Ask AI", title, body).Run()
}

func (s *System) Sleep(duration time.Duration) { time.Sleep(duration) }

func copyArguments(windowJSON []byte) []string {
	if activeWindowIsTerminal(windowJSON) {
		return []string{"-M", "ctrl", "-P", "Insert", "-p", "Insert", "-m", "ctrl"}
	}
	return []string{"-M", "ctrl", "c", "-m", "ctrl"}
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
