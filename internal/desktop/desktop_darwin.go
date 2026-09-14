//go:build darwin

package desktop

import (
	"bytes"
	"os/exec"
	"time"
)

type System struct{}

func New() *System { return &System{} }

func (s *System) ReadClipboard() (string, error) {
	output, err := exec.Command("pbpaste").Output()
	return string(output), err
}

func (s *System) WriteClipboard(text string) error {
	command := exec.Command("pbcopy")
	command.Stdin = bytes.NewBufferString(text)
	return command.Run()
}

func (s *System) CopySelection() error {
	return runAppleScript(`tell application "System Events" to keystroke "c" using command down`)
}

func (s *System) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}

func (s *System) Paste() error {
	return runAppleScript(`tell application "System Events" to keystroke "v" using command down`)
}

func (s *System) Notify(title, body string) error {
	script := `on run argv
display notification (item 2 of argv) with title (item 1 of argv)
end run`
	return exec.Command("osascript", "-e", script, title, body).Run()
}

func (s *System) Sleep(duration time.Duration) { time.Sleep(duration) }

func runAppleScript(script string) error {
	return exec.Command("osascript", "-e", script).Run()
}
