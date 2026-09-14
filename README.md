# Ask AI Everywhere

Open selected text as an unsent draft in a fresh consumer-AI browser tab. The
same command works from an Omarchy/Hyprland or macOS/AeroSpace keybinding.

Ask AI Everywhere does not use an API key, put selected text in a URL, or submit
the prompt. It copies the selection locally, opens the configured provider in
the default browser, and makes a best-effort simulated paste. If automation
fails, the selection remains on the clipboard for a manual paste.

## Behavior

1. Select text in the focused application.
2. Press your configured window-manager shortcut.
3. `ask-ai` simulates Copy and checks that text was captured.
4. It opens a fresh tab for the provider saved in
   `~/.config/ask-ai/config.json`.
5. After a configurable page-load delay, it simulates Paste without pressing
   Enter.

If no text is selected, the previous text clipboard is restored where possible,
a desktop notification appears, and no browser tab is opened. If the browser or
paste command reports a failure, the selected text stays on the clipboard and a
notification explains that it can be pasted manually.

Browser pages do not expose a portable way for this utility to inspect their
prompt field. A successful paste command therefore means "injection attempted,"
not that the page DOM was verified. Increase `pasteDelayMs` if a provider loads
too slowly.

## Supported systems

### Linux under Wayland

Runtime commands:

- `wl-copy` and `wl-paste` from `wl-clipboard`
- `wtype`
- `xdg-open`
- `notify-send`
- `hyprctl` (used to recognize terminal windows)

These are already present on the Omarchy machine used to develop the project.
Other Wayland desktops can invoke the CLI too, provided those commands work in
the session. Under Omarchy, terminal-tagged windows use Ctrl-Insert for Copy so
selected terminal text is captured without sending Ctrl-C to the shell.

### macOS

The CLI uses the built-in `pbcopy`, `pbpaste`, `open`, and `osascript` commands.
Grant Accessibility access to the application that launches `ask-ai` so
AppleScript can simulate Command-C and Command-V:

1. Open **System Settings → Privacy & Security → Accessibility**.
2. Enable AeroSpace. Depending on how the binary is launched, macOS may instead
   ask you to enable the executable or its parent application.
3. Invoke the shortcut once and approve any macOS automation prompt.

Without Accessibility permission, selection capture or paste will fail safely
and produce a notification. Selection capture must succeed before a browser tab
is opened.

## Build and install

Go 1.24 or newer is required to build. The resulting executable is standalone;
Go is not needed at runtime.

```sh
go test ./...
mkdir -p "$HOME/.local/bin"
go build -o "$HOME/.local/bin/ask-ai" ./cmd/ask-ai
```

Confirm the installation:

```sh
$HOME/.local/bin/ask-ai -version
```

The development build reports `dev`. Release builds can inject a version with:

```sh
go build -ldflags "-X main.version=1.0.0" -o bin/ask-ai ./cmd/ask-ai
```

Cross-compile from Linux for common macOS targets:

```sh
GOOS=darwin GOARCH=arm64 go build -o bin/ask-ai-darwin-arm64 ./cmd/ask-ai
GOOS=darwin GOARCH=amd64 go build -o bin/ask-ai-darwin-amd64 ./cmd/ask-ai
```

## Configuration

Copy [`config.example.json`](config.example.json) or symlink a file from your
dotfiles repository:

```sh
mkdir -p "$HOME/.config/ask-ai"
ln -s "$HOME/path/to/dotfiles/ask-ai/config.json" \
  "$HOME/.config/ask-ai/config.json"
```

Example:

```json
{
  "provider": "chatgpt",
  "customUrl": "",
  "shortcutReleaseDelayMs": 200,
  "pasteDelayMs": 1500
}
```

Fields:

| Field | Meaning |
| --- | --- |
| `provider` | `chatgpt`, `claude`, `gemini`, `kimi`, or `custom` |
| `customUrl` | HTTPS browser URL required when `provider` is `custom` |
| `shortcutReleaseDelayMs` | Delay before Copy, allowing shortcut keys to be released |
| `pasteDelayMs` | Delay between opening the tab and attempting Paste |

The built-in destinations are:

| Provider | URL |
| --- | --- |
| ChatGPT | `https://chatgpt.com/` |
| Claude | `https://claude.ai/new` |
| Gemini | `https://gemini.google.com/app` |
| Kimi | `https://www.kimi.com/` |

The Kimi destination follows its [official consumer site](https://www.kimi.com/).
Custom destinations must be valid HTTPS URLs. Unknown fields and invalid values
are rejected rather than silently ignored. If the file does not exist, ChatGPT
and the default delays are used. Set `ASK_AI_CONFIG` to use another config path
for testing or an alternative dotfiles layout.

## Hyprland/Omarchy keybinding

Add a binding to `~/.config/hypr/bindings.lua`. This example intentionally
replaces Omarchy's stock `SUPER + SHIFT + A` ChatGPT binding, so it unbinds that
shortcut first:

```lua
local home = os.getenv("HOME")

-- Previously: stock ChatGPT web-app shortcut.
hl.unbind("SUPER + SHIFT + A")
o.bind(
  "SUPER + SHIFT + A",
  "Ask AI with selection",
  home .. "/.local/bin/ask-ai",
  { release = true }
)
```

Using a release binding prevents the Super and Shift keys from contaminating
the simulated Copy shortcut. If you choose an unused key combination, omit the
`hl.unbind` line. Inspect existing bindings first:

```sh
omarchy menu keybindings --print
```

Hyprland reloads Lua configuration automatically. Validate it after editing:

```sh
hyprctl reload
hyprctl configerrors
```

## AeroSpace keybinding

Add a command under `[mode.main.binding]` in `~/.aerospace.toml` or
`~/.config/aerospace/aerospace.toml`:

```toml
[mode.main.binding]
alt-shift-a = 'exec-and-forget $HOME/.local/bin/ask-ai'
```

Choose any unused binding. AeroSpace runs `exec-and-forget` through Bash; see
the official [AeroSpace command reference](https://nikitabobko.github.io/AeroSpace/commands#exec-and-forget).

## Troubleshooting

- **No selected text notification:** the focused application did not respond to
  simulated Copy, or only whitespace was selected. On macOS, verify
  Accessibility permission.
- **The text remains only on the clipboard:** increase `pasteDelayMs`; verify
  that the AI page focuses its prompt when it finishes loading.
- **Paste failure notification on macOS:** enable Accessibility access for the
  launcher and retry.
- **Nothing happens from a keybinding:** use an absolute binary path, then run
  that path from a terminal to expose any missing-command or config error.
- **Invalid configuration notification:** run the command from a terminal for
  the precise validation error. Custom URLs must use HTTPS.

The tool intentionally has no provider-specific browser extension or DOM
integration. Provider UI changes may affect autofocus, but the clipboard
fallback remains available.
