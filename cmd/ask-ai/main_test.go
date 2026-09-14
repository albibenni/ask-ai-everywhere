package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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

func TestProviderWithoutArgumentsPromptsForAndPersistsSelection(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"provider":"chatgpt","pasteDelayMs":1500,"shortcutReleaseDelayMs":200}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runProvider(configPath, nil, strings.NewReader("2\n"), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("runProvider() exit = %d, stderr = %q", exitCode, stderr.String())
	}
	config, err := askai.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if config.Provider != "claude" {
		t.Fatalf("provider = %q, want claude", config.Provider)
	}
	for _, expected := range []string{"Current provider: chatgpt", "Choose provider", "Provider set to claude"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), expected)
		}
	}
}

func TestProviderPickerPromptsForCustomURL(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runProvider(
		configPath,
		nil,
		strings.NewReader("5\nhttps://example.com/chat\n"),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("runProvider() exit = %d, stderr = %q", exitCode, stderr.String())
	}
	config, err := askai.LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if config.Provider != "custom" || config.URL != "https://example.com/chat" {
		t.Fatalf("config = %+v", config)
	}
}

func TestProviderCurrentIsNonInteractive(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runProvider(configPath, []string{"current"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "chatgpt\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", exitCode, stdout.String(), stderr.String())
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
