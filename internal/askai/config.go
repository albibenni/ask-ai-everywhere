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
	"kimi":    "https://www.kimi.ai/",
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
	raw, _, err := readConfigFile(path)
	if err != nil {
		return Config{}, err
	}
	return runtimeConfig(raw)
}

// SetProvider atomically updates the provider while preserving other settings
// and a directly symlinked config file.
func SetProvider(path, provider, customURL string) error {
	raw, mode, err := readConfigFile(path)
	if err != nil {
		return err
	}
	if provider != "custom" && customURL != "" {
		return errors.New("a custom URL can only be used with the custom provider")
	}
	raw.Provider = provider
	if customURL != "" {
		raw.CustomURL = customURL
	}
	if _, err := runtimeConfig(raw); err != nil {
		return err
	}
	return writeConfigFile(path, raw, mode)
}

func defaultConfigFile() configFile {
	return configFile{
		Provider:               "chatgpt",
		ShortcutReleaseDelayMS: defaultShortcutReleaseDelay,
		PasteDelayMS:           defaultPasteDelay,
	}
}

func readConfigFile(path string) (configFile, os.FileMode, error) {
	raw := defaultConfigFile()
	mode := os.FileMode(0o644)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return raw, mode, nil
		}
		return configFile{}, 0, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()
	if info, statErr := file.Stat(); statErr == nil {
		mode = info.Mode().Perm()
	}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return configFile{}, 0, fmt.Errorf("decode config: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return configFile{}, 0, err
	}
	return raw, mode, nil
}

func runtimeConfig(raw configFile) (Config, error) {

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

func writeConfigFile(path string, raw configFile, mode os.FileMode) error {
	writePath := path
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve config symlink: %w", err)
		}
		writePath = resolved
	}

	directory := filepath.Dir(writePath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".config.json.*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(raw); err != nil {
		temporary.Close()
		return fmt.Errorf("encode config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(temporaryPath, writePath); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
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
