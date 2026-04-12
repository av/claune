# Contributing to Claune

Thank you for your interest in contributing to Claune! We welcome all contributions, from bug reports to feature additions.

## Getting Started

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally: `git clone git@github.com:your-username/claune.git`
3. **Install Go** (1.24.0 or higher).
4. **Make changes** and run tests.

## Building and Testing

After any code changes, rebuild and reinstall the binary to test locally:

```bash
PATH="$HOME/go/bin:$PATH" go test ./... && PATH="$HOME/go/bin:$PATH" make install
```

The Go binary is installed at `~/.local/bin/claune`.

## Downloading Sounds

If you need to fetch meme sounds for development from boardsounds.com, there is an automated script included:

```bash
./scripts/download-boardsound.sh <sound-slug> [output-file.mp3]
```

## Pull Request Process

1. Create a descriptive branch name.
2. Commit your changes with clear messages.
3. Push to your fork and submit a Pull Request to the `main` branch.
4. Ensure all tests pass and that your code adheres to standard Go formatting (`go fmt`).

## Code of Conduct

Please be respectful and constructive in your communications. Let's keep the meme energy positive!
