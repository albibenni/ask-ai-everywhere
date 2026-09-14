package askai

import (
	"os"
	"path/filepath"
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
