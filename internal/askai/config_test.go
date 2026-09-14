package askai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigUsesChatGPTDefaultsWhenFileDoesNotExist(t *testing.T) {
	config, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if config.Provider != "chatgpt" {
		t.Fatalf("Provider = %q, want chatgpt", config.Provider)
	}
	if config.URL != "https://chatgpt.com/" {
		t.Fatalf("URL = %q", config.URL)
	}
	if config.ShortcutReleaseDelay != 200*time.Millisecond {
		t.Fatalf("ShortcutReleaseDelay = %v", config.ShortcutReleaseDelay)
	}
	if config.PasteDelay != 1500*time.Millisecond {
		t.Fatalf("PasteDelay = %v", config.PasteDelay)
	}
}

func TestSetProviderUpdatesSymlinkTargetAndPreservesOtherSettings(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "dotfiles-config.json")
	link := filepath.Join(directory, "config.json")
	body := `{"provider":"chatgpt","customUrl":"https://example.com/chat","shortcutReleaseDelayMs":321,"pasteDelayMs":2345}`
	if err := os.WriteFile(target, []byte(body), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := SetProvider(link, "claude", ""); err != nil {
		t.Fatalf("SetProvider() error = %v", err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("config symlink was replaced")
	}
	config, err := LoadConfig(link)
	if err != nil {
		t.Fatal(err)
	}
	if config.Provider != "claude" || config.ShortcutReleaseDelay != 321*time.Millisecond || config.PasteDelay != 2345*time.Millisecond {
		t.Fatalf("config = %+v", config)
	}
	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), `"customUrl": "https://example.com/chat"`) {
		t.Fatalf("custom URL was not preserved: %s", written)
	}
}

func TestSetProviderRequiresHTTPSForCustomProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SetProvider(path, "custom", "http://example.com"); err == nil {
		t.Fatal("SetProvider() unexpectedly accepted an HTTP URL")
	}
}

func TestBashCompletionIncludesProvidersAndHelperCommand(t *testing.T) {
	completion := BashCompletion()
	for _, expected := range []string{"chatgpt", "claude", "gemini", "kimi", "custom", "ask-ai-provider"} {
		if !strings.Contains(completion, expected) {
			t.Fatalf("completion does not include %q", expected)
		}
	}
}

func TestLoadConfigResolvesBuiltInAndCustomProviders(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"claude", `{"provider":"claude"}`, "https://claude.ai/new"},
		{"gemini", `{"provider":"gemini"}`, "https://gemini.google.com/app"},
		{"kimi", `{"provider":"kimi"}`, "https://www.kimi.com/"},
		{"custom", `{"provider":"custom","customUrl":"https://example.com/chat"}`, "https://example.com/chat"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(test.body), 0o600); err != nil {
				t.Fatal(err)
			}

			config, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			if config.URL != test.want {
				t.Fatalf("URL = %q, want %q", config.URL, test.want)
			}
		})
	}
}

func TestLoadConfigRejectsUnsafeOrUnknownDestinations(t *testing.T) {
	tests := []string{
		`{"provider":"unknown"}`,
		`{"provider":"custom"}`,
		`{"provider":"custom","customUrl":"http://example.com"}`,
		`{"provider":"custom","customUrl":"javascript:alert(1)"}`,
	}

	for _, body := range tests {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); err == nil {
			t.Fatalf("LoadConfig(%s) unexpectedly succeeded", body)
		}
	}
}
