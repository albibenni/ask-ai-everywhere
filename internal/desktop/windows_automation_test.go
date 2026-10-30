package desktop

import "testing"

func TestWindowsTextPasteRecognizesBrowsersAndRejectsEditors(t *testing.T) {
	for _, name := range []string{"chrome.exe", "msedge.exe", "firefox.exe", "brave.exe", "vivaldi.exe", "opera.exe", "CHROME.EXE"} {
		if !windowsBrowserExecutable(name) {
			t.Errorf("browser %q should support text paste", name)
		}
	}
	// Electron apps can use the same window class as Chromium browsers. Window
	// class alone must never authorize Select All/Paste into an editor.
	for _, name := range []string{"Code.exe", "notepad.exe", "slack.exe", "WindowsTerminal.exe", "", "unknown.exe"} {
		if windowsBrowserExecutable(name) {
			t.Errorf("application %q must use manual paste", name)
		}
	}
}
