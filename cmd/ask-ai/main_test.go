package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"ask-ai-everywhere/internal/askai"
)

func TestNormalizeArgsTurnsHelperInvocationIntoProviderCommand(t *testing.T) {
	want := []string{"provider", "claude"}
	if got := normalizeArgs("ask-ai-provider", []string{"claude"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeArgs() = %v, want %v", got, want)
	}
}

func TestRunErrorHelpsWhenNoSelectionWasCaptured(t *testing.T) {
	message := formatRunError(askai.ErrNoSelection)
	for _, expected := range []string{
		"no text selection was captured",
		"configured keybinding",
		"ask-ai provider",
		"ask-ai -h",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("formatRunError() = %q, missing %q", message, expected)
		}
	}
}

func TestRunErrorKeepsOtherErrorsConcise(t *testing.T) {
	if got := formatRunError(errors.New("browser unavailable")); got != "ask-ai: browser unavailable\n" {
		t.Fatalf("formatRunError() = %q", got)
	}
}

func TestNormalizeArgsLeavesMainCommandArgumentsUnchanged(t *testing.T) {
	want := []string{"provider", "gemini"}
	if got := normalizeArgs("ask-ai", want); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeArgs() = %v, want %v", got, want)
	}
}
