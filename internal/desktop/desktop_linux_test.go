//go:build linux

package desktop

import (
	"reflect"
	"testing"
)

func TestCopyArgumentsUseTerminalSafeShortcutForOmarchyTerminalTag(t *testing.T) {
	window := []byte(`{"class":"com.mitchellh.ghostty","tags":["terminal*"]}`)
	want := []string{"-M", "ctrl", "-P", "Insert", "-p", "Insert", "-m", "ctrl"}

	if got := copyArguments(window); !reflect.DeepEqual(got, want) {
		t.Fatalf("copyArguments() = %v, want %v", got, want)
	}
}

func TestCopyArgumentsUseStandardShortcutForOtherWindows(t *testing.T) {
	window := []byte(`{"class":"Brave-browser","tags":[]}`)
	want := []string{"-M", "ctrl", "c", "-m", "ctrl"}

	if got := copyArguments(window); !reflect.DeepEqual(got, want) {
		t.Fatalf("copyArguments() = %v, want %v", got, want)
	}
}

func TestCopyArgumentsFallBackToKnownTerminalClassesWithoutTags(t *testing.T) {
	window := []byte(`{"class":"Alacritty"}`)
	want := []string{"-M", "ctrl", "-P", "Insert", "-p", "Insert", "-m", "ctrl"}

	if got := copyArguments(window); !reflect.DeepEqual(got, want) {
		t.Fatalf("copyArguments() = %v, want %v", got, want)
	}
}
