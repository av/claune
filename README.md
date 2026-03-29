# claune

Sound effects for Claude Code tool use events.

## Install

### Pre-built binary

Download the binary for your platform from [Releases](https://github.com/av/claune/releases).

```
chmod +x claune-linux-amd64
sudo mv claune-linux-amd64 /usr/local/bin/claune
```

### From source

```
git clone https://github.com/av/claune.git
cd claune
make install
```

Installs to `~/.local/bin/claune` by default. Override with `PREFIX=/usr/local make install`.

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
| `test-sounds` | Play all sounds to verify audio |
| `play <event>` | Play a sound for the given event type |
| `help` | Show help message |

Event types: `cli:start`, `tool:start`, `tool:success`, `tool:error`, `cli:done`.

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

## Audio backends

claune uses the first available backend in this order:

1. `paplay` (PulseAudio / PipeWire)
2. `pw-play` (PipeWire native)
3. `aplay` (ALSA)
4. `afplay` (macOS)
