# claune

Sound effects for Claude Code tool use events.

## Install

### Prerequisites

- **Go is required for source installs.** This repository declares `go 1.24.0` and `toolchain go1.24.1` in `go.mod`, so install a compatible Go toolchain and verify the exact executable that `make install` will use before continuing:

  ```bash
  if command -v go >/dev/null 2>&1; then
    command -v go
    go version
  else
    test -x "$HOME/go/bin/go"
    "$HOME/go/bin/go" version
  fi
  ```

  `make install` first uses `go` from `PATH`, and if that is missing it falls back to `$HOME/go/bin/go` when present.

- **`~/.local/bin` should be on your `PATH`** if you use the default install target, because `make install` installs `claune` to `~/.local/bin/claune`. After installation, verify both of these:

  ```bash
  command -v claune
  printf '%s\n' "$PATH"
  ```

  `command -v claune` should resolve to `~/.local/bin/claune` (or your overridden `PREFIX` location). If it does not, add `~/.local/bin` to your shell `PATH` before treating that as a project problem.

- **Claude Code's `claude` CLI is only required for passthrough usage.** Local management commands such as `claune help`, `claune status`, and `claune test-sounds` work without `claude`, but running Claude Code through `claune` requires `claude` to be available on `PATH`. Verify readiness with:

  ```bash
  command -v claude
  claude --version
  ```

  If the CLI is installed but passthrough still does not work, you may also need to authenticate/login with Claude Code separately; installation alone may not be sufficient.

### Pre-built binary

Download the binary for your platform from [Releases](https://github.com/av/claune/releases).

```
chmod +x claune-linux-amd64
sudo mv claune-linux-amd64 /usr/local/bin/claune
```

### From source

```
git clone git@github.com:av/claune.git
cd claune
PATH="$HOME/go/bin:$PATH" make install
```

`make install` builds and installs the binary to `~/.local/bin/claune` by default. Override with `PREFIX=/usr/local make install`.

If Go is installed but not on your `PATH`, use this exact recovery command:

```bash
PATH="$HOME/go/bin:$PATH" make install
```

That matches the Makefile's fallback behavior for the common `~/go/bin/go` install location.

After `make install`, confirm the installed binary is reachable from your shell:

```bash
command -v claune
```

If that command does not print `~/.local/bin/claune` (or your overridden `PREFIX` location), update your `PATH` before continuing.

## Usage

### Passthrough mode (recommended)

Use `claune` as a drop-in replacement for `claude`:

```
claune                     # Start Claude Code interactively with sounds
claune -p "explain this"   # Pass any arguments through to Claude Code
claune --model sonnet      # All claude flags work transparently
```

Hooks are auto-installed on first run. Sound effects only play in sessions started via `claune` — running `claude` directly is unaffected.

### Manual hook management

```
claune install    # Add hooks to ~/.claude/settings.json
claune uninstall  # Remove hooks from ~/.claude/settings.json
```

## Commands

| Command | Description |
|---|---|
| `claune [args...]` | Run Claude Code with sound effects (passthrough) |
| `install` | Add sound hooks to Claude Code settings |
| `uninstall` | Remove sound hooks from Claude Code settings |
| `status` | Show hook and audio status |
| `test-sounds` | Attempt to play all sounds to verify audio |
| `play <event>` | Play a sound for the given event type |
| `play <event> <tool-name> <tool-input>` | Play a sound using semantic tool context |
| `config <msg>` | Update configuration from a natural-language prompt |
| `automap <dir>` | Use AI to map sound files in a directory to events |
| `import-circus <url> <name> [event]` | Import a meme sound (name must be a short alias without slashes) |
| `analyze-log` | Analyze stdin log text and play a sound |
| `analyze-resp` | Analyze stdin AI response text and optionally override the sound strategy |
| `help` | Show help message |

`claune test-sounds` currently exercises these built-in events: `cli:start`, `tool:start`, `tool:success`, `tool:error`, `cli:done`, `build:success`, `test:fail`, `panic`, `warn`.

`test-sounds` does not override mute state. If claune is muted — including the default smart-mute window described below — it exits silently before any playback attempt or status lines are printed.

`claune play` accepts exactly these forms:

```bash
claune play <event>
claune play <event> <tool-name> <tool-input>
```

The first form plays the requested event directly. The second preserves the existing semantic-analysis path by using `<tool-name>` and `<tool-input>` as AI context before playback.

`claune play <event>` accepts the full built-in event set: `cli:start`, `tool:start`, `tool:success`, `tool:error`, `cli:done`, `tool:destructive`, `tool:readonly`, `build:success`, `build:fail`, `test:fail`, `panic`, `warn`.

## Configuration

Config file: `~/.claune.json`

```json
{
  "mute": false,
  "volume": 0.7,
  "sounds": {
    "tool:success": "/path/to/custom.wav"
  }
}
```

| Field | Type | Default | Description |
|---|---|---|---|
| `mute` | bool | unset | Force mute or unmute |
| `volume` | float | 1.0 | Playback volume (0.0 to 1.0) |
| `sounds` | object | {} | Override default sounds per event type |

### Smart mute

When `mute` is not set in the config, claune auto-mutes between 23:00 and 07:00 local time.
Set `"mute": false` to disable this behavior.

For a zero-improvisation cold-start audio check, create `~/.claune.json` with mute explicitly disabled, then run the sound commands:

```bash
mkdir -p "$HOME"
printf '{"mute":false}\n' > "$HOME/.claune.json"
claune test-sounds
claune play tool:success
```

Use `claune status` if you want to confirm the effective mute state first: a muted fresh config reports `Sound: muted`, while an explicit `{"mute":false}` config reports a volume line instead (for example, `Volume: 100%`).

## Audio backends

claune uses the first available backend in this order:

1. `paplay` (PulseAudio / PipeWire)
2. `pw-play` (PipeWire native)
3. `aplay` (ALSA)
4. `afplay` (macOS)
