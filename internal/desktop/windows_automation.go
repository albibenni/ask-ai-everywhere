package desktop

import "strings"

func windowsBrowserExecutable(name string) bool {
	switch strings.ToLower(name) {
	case "chrome.exe", "msedge.exe", "firefox.exe", "brave.exe", "vivaldi.exe",
		"opera.exe", "chromium.exe", "librewolf.exe", "waterfox.exe", "floorp.exe", "zen.exe":
		return true
	}
	return false
}
