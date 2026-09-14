package askai

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultShortcutReleaseDelay = 200
	defaultPasteDelay           = 1500
)

var providerURLs = map[string]string{
	"chatgpt": "https://chatgpt.com/",
	"claude":  "https://claude.ai/new",
	"gemini":  "https://gemini.google.com/app",
	"kimi":    "https://www.kimi.com/",
}

type configFile struct {
	Provider               string `json:"provider"`
	CustomURL              string `json:"customUrl"`
	ShortcutReleaseDelayMS int    `json:"shortcutReleaseDelayMs"`
	PasteDelayMS           int    `json:"pasteDelayMs"`
}

// Config is the validated runtime configuration used by the workflow.
type Config struct {
	Provider             string
	URL                  string
	ShortcutReleaseDelay time.Duration
	PasteDelay           time.Duration
}

// DefaultConfigPath returns the cross-platform, dotfile-friendly config path.
func DefaultConfigPath() (string, error) {
	if path := os.Getenv("ASK_AI_CONFIG"); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ask-ai", "config.json"), nil
}

// LoadConfig reads and validates config. A missing file uses safe defaults.
func LoadConfig(path string) (Config, error) {
	raw := configFile{
		Provider:               "chatgpt",
		ShortcutReleaseDelayMS: defaultShortcutReleaseDelay,
		PasteDelayMS:           defaultPasteDelay,
	}

	file, err := os.Open(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("open config: %w", err)
		}
	} else {
		defer file.Close()
		decoder := json.NewDecoder(file)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&raw); err != nil {
			return Config{}, fmt.Errorf("decode config: %w", err)
		}
		if err := ensureJSONEnd(decoder); err != nil {
			return Config{}, err
		}
	}

	if raw.ShortcutReleaseDelayMS < 0 || raw.ShortcutReleaseDelayMS > 60_000 {
		return Config{}, errors.New("shortcutReleaseDelayMs must be between 0 and 60000")
	}
	if raw.PasteDelayMS < 0 || raw.PasteDelayMS > 60_000 {
		return Config{}, errors.New("pasteDelayMs must be between 0 and 60000")
	}

	destination, ok := providerURLs[raw.Provider]
	if raw.Provider == "custom" {
		if !validHTTPSURL(raw.CustomURL) {
			return Config{}, errors.New("customUrl must be a valid HTTPS URL")
		}
		destination = raw.CustomURL
	} else if !ok {
		return Config{}, fmt.Errorf("unknown provider %q", raw.Provider)
	}

	return Config{
		Provider:             raw.Provider,
		URL:                  destination,
		ShortcutReleaseDelay: time.Duration(raw.ShortcutReleaseDelayMS) * time.Millisecond,
		PasteDelay:           time.Duration(raw.PasteDelayMS) * time.Millisecond,
	}, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode config: multiple JSON values")
		}
		return fmt.Errorf("decode config: %w", err)
	}
	return nil
}

func validHTTPSURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}
