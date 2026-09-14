# Ask AI Everywhere

Open selected text as an unsent draft in a fresh consumer-AI browser tab. The
same command works from an Omarchy/Hyprland or macOS/AeroSpace keybinding.

Ask AI Everywhere does not use an API key, put selected text in a URL, or submit
the prompt. It copies the selection locally, opens the configured provider in
the default browser, selects any restored composer draft, and makes a
best-effort simulated paste that replaces it. If automation fails, the
selection remains on the clipboard for a manual paste.

## Behavior

1. Select text in the focused application.
2. Press your configured window-manager shortcut.
3. `ask-ai` copies the selection and checks that text was captured. Hyprland
   uses its native exact-key dispatcher; macOS uses Accessibility automation.
4. It opens a fresh tab for the provider saved in
   `~/.config/ask-ai/config.json`.
5. After a configurable page-load delay, it simulates Select All followed by
   Paste, replacing any restored unsent draft without pressing Enter.

If no text is selected and the clipboard contains text, a desktop notification
warns that the first (most recent) clipboard item will be used, then that text is
opened as the draft. The utility reads the current clipboard value; it does not
inspect a clipboard manager's full history. If neither source contains text, it
notifies and stops without opening a browser tab. If the browser or paste command
reports a failure, the selected or clipboard text stays available for a manual
paste.

Running `ask-ai` directly in a terminal without an available selection uses the
current clipboard text after showing the warning. If the clipboard is empty too,
it prints provider commands and points to `ask-ai -h`. The same executable
remains the keybinding target; no separate selection flag is required.

Browser pages do not expose a portable way for this utility to inspect their
prompt field. A successful paste command therefore means "injection attempted,"
not that the page DOM was verified. Increase `pasteDelayMs` if a provider loads
too slowly. Kimi receives its advertised Ctrl-K New Chat shortcut before
Select All and Paste so its composer has focus.

## Supported systems

### Linux under Wayland

Runtime commands:

- `wl-copy` and `wl-paste` from `wl-clipboard`
- `wtype` (used for browser paste)
- `xdg-open`
- `notify-send`
- `hyprctl` (used to recognize terminal windows)

These are already present on the Omarchy machine used to develop the project.
Other Wayland desktops can invoke the CLI too, provided those commands work in
the session. Under Omarchy, native Hyprland dispatch avoids merging physically
held shortcut modifiers into Ctrl-C. Terminal-tagged windows use Ctrl-Insert so
selected terminal text is captured without interrupting the shell.

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
make check
make install
```

Confirm the installation:

```sh
$HOME/.local/bin/ask-ai -version
```

Build versions are derived from Git. A tagged commit reports its release tag;
other commits report their abbreviated commit hash, with `-dirty` appended when
the working tree has changes. Inspect the value before building with:

```sh
make version
```

`make install` uses that value automatically. `VERSION` remains available as an
explicit override when needed:

```sh
make build VERSION=v1.0.0
```

Build Linux and macOS binaries for Intel and ARM:

```sh
make cross-build VERSION=v1.0.0
```

Create a release from a clean `main` branch:

```sh
make release
```

The release command suggests the next patch version, prompts for a semantic
version and confirmation, runs the test/vet suite and all cross-builds, creates
an annotated Git tag, then atomically pushes `main` and the tag to `origin`.
Afterward, a plain `make install` embeds that tag in the installed binary.

Run `make help` for formatting, testing, vetting, building, installation, and
cleanup commands. Override `INSTALL_PREFIX` when `~/.local` is not the desired
installation prefix.

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

### Change provider from the command line

Open the interactive provider picker:

```sh
ask-ai provider
```

The Bubble Tea picker shows the current provider and lets you choose ChatGPT,
Claude, Gemini, Kimi, or a custom HTTPS URL. Use the arrow keys or `j`/`k` to
move and Enter to select. Press Esc, `q`, or Ctrl-C to keep the current provider.
Choosing Custom URL opens its HTTPS URL editor inside the same TUI; Enter saves
it and Esc returns to the provider list.

Print the current provider non-interactively:

```sh
ask-ai provider current
```

Change it without opening the JSON file:

```sh
ask-ai provider chatgpt
ask-ai provider claude
ask-ai provider gemini
ask-ai provider kimi
ask-ai provider custom https://example.com/chat
```

The installed helper provides shorter equivalents:

```sh
ask-ai-provider
ask-ai-provider claude
ask-ai-provider custom https://example.com/chat
```

Running `ask-ai-provider` without arguments opens the same interactive picker.

Both commands update the configured file atomically. If that file is a symlink,
its target is updated without replacing the symlink. In this setup, changes are
written directly to the tracked file under `~/dotfiles/ask-ai`.

### Bash completion

`make install` installs completion for both `ask-ai` and `ask-ai-provider` under
`~/.local/share/bash-completion/completions`. Start a new Bash session, or load
it immediately with:

```sh
source <(ask-ai completion bash)
```

Provider completion includes `current`, `chatgpt`, `claude`, `gemini`, `kimi`,
and `custom`.

The built-in destinations are:

| Provider | URL |
| --- | --- |
| ChatGPT | `https://chatgpt.com/` |
| Claude | `https://claude.ai/new` |
| Gemini | `https://gemini.google.com/app` |
| Kimi | `https://www.kimi.ai/` |

The Kimi destination follows its [official consumer site](https://www.kimi.ai/).
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

- **Clipboard fallback notification:** the focused application had no selection,
  so the first (most recent) clipboard item is being used. Cancel or close the
  fresh tab if that is not the text you intended.
- **No selected text notification with no browser:** neither the focused
  selection nor the current clipboard contains text. On macOS, verify
  Accessibility permission if copying a visible selection fails.
- **The text remains only on the clipboard:** increase `pasteDelayMs`; verify
  that the AI page focuses its prompt when it finishes loading.
- **Old and new drafts appear together:** update the installed binary. Current
  versions select the restored composer draft before pasting the new selection.
- **Paste failure notification on macOS:** enable Accessibility access for the
  launcher and retry.
- **Nothing happens from a keybinding:** use an absolute binary path, then run
  that path from a terminal to expose any missing-command or config error.
- **Invalid configuration notification:** run the command from a terminal for
  the precise validation error. Custom URLs must use HTTPS.

The tool intentionally has no provider-specific browser extension or DOM
integration. Provider UI changes may affect autofocus, but the clipboard
fallback remains available.

## Neovim Visual selections

A Neovim Visual selection belongs to Neovim rather than the terminal surface,
so a compositor-level Copy shortcut cannot read it. Add an editor mapping that
yanks the selection to the system clipboard and invokes the explicit clipboard
mode:

```lua
vim.keymap.set("x", "<leader>ai", function()
  vim.cmd([[normal! "+y]])
  vim.fn.jobstart(
    { (vim.env.HOME or "") .. "/.local/bin/ask-ai", "--from-clipboard" },
    { detach = true }
  )
end, { desc = "Open selection in AI chat" })
```

In Visual mode, press `<leader>ai`. The `--from-clipboard` option is intended
for integrations that explicitly copy fresh text first; the normal global
shortcut retains stale-clipboard detection.
