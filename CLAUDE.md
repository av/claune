# Claune

## Build & Test

After any code changes, always rebuild and reinstall the binary so it can be tested locally:

```
PATH="/home/everlier/go/bin:$PATH" go test ./... && PATH="/home/everlier/go/bin:$PATH" make install
```

The Go binary is at `/home/everlier/go/bin/go`. The install target puts the binary in `~/.local/bin/claune`.

## Downloading Sounds

If you need to fetch meme sounds for development from boardsounds.com, there is an automated script included in the repository that skips the need for checking network logs:

```bash
./scripts/download-boardsound.sh <sound-slug> [output-file.mp3]
```

Example:
```bash
./scripts/download-boardsound.sh metal-gear-solid-alert internal/audio/sounds/alert.mp3
```
