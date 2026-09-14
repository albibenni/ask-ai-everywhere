//go:build linux

package desktop

import (
	"strings"
	"testing"
)

func TestReplaceDraftArgumentsSelectAllBeforePasting(t *testing.T) {
	want := "-M ctrl -k a -s 50 -k v -m ctrl"
	if got := strings.Join(replaceDraftArguments(), " "); got != want {
		t.Fatalf("replaceDraftArguments() = %q, want %q", got, want)
	}
}

func TestCopyKeyUsesTerminalSafeShortcutForOmarchyTerminalTag(t *testing.T) {
	window := []byte(`{"class":"com.mitchellh.ghostty","tags":["terminal*"]}`)
	if got := copyKey(window); got != "Insert" {
		t.Fatalf("copyKey() = %q, want Insert", got)
	}
}

func TestCopyKeyUsesStandardShortcutForOtherWindows(t *testing.T) {
	window := []byte(`{"class":"Brave-browser","tags":[]}`)
	if got := copyKey(window); got != "C" {
		t.Fatalf("copyKey() = %q, want C", got)
	}
}

func TestCopyKeyFallsBackToKnownTerminalClassesWithoutTags(t *testing.T) {
	window := []byte(`{"class":"Alacritty"}`)
	if got := copyKey(window); got != "Insert" {
		t.Fatalf("copyKey() = %q, want Insert", got)
	}
}

func TestShortcutStateUsesHyprlandExactKeyDispatch(t *testing.T) {
	want := `hl.dsp.send_key_state({ mods = "CTRL", key = "C", state = "down" })`
	if got := shortcutState("C", "down"); got != want {
		t.Fatalf("shortcutState() = %q, want %q", got, want)
	}
}
