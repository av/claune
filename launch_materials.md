# 🚀 Launching Claune v1.0.0: The Claude Desktop Audio Companion

**Ever wished your Claude Desktop had more personality? Claune is an open-source background audio companion that automatically reacts to your Claude Desktop activities with customizable sound effects and AI-generated text-to-speech.**

### What is Claune?
Claune is a fast, lightweight, and cross-platform CLI tool written in Go that runs silently in the background. It listens to Claude Desktop events and plays highly-customizable sound effects. Whether you want a satisfying "ding" when an AI message completes, a meme sound when an error occurs, or even AI-generated TTS narrations of Claude's actions, Claune has you covered!

### 🌟 Key Features
- **Seamless Claude Desktop Integration**: Automatically detects when Claude starts, stops, errors, or completes tasks.
- **Customizable Event Sounds**: Bind any local audio file (.mp3, .wav) to specific AI states.
- **AI Text-to-Speech (TTS)**: Don't just want sounds? Use Anthropic's API to dynamically generate context-aware TTS responses to what Claude is doing.
- **Cross-Platform**: Works beautifully on macOS, Windows, and Linux.
- **Silent Background Hook**: Designed to run cleanly as an unobtrusive background process alongside your workflow.
- **Instant Setup**: Our new `claune setup` wizard walks you through everything in seconds!

### 🛠️ What's new in v1.0.0 (The Release-Ready Update)
We've spent the last few iterations deeply polishing the end-user experience so that anyone can use Claune without being a power user.
- **One-Line Installers**: Just run our simple curl or PowerShell scripts to get up and running instantly.
- **Setup Wizard**: `claune setup` interactively walks you through configuring sounds, volumes, and API keys.
- **Robust Error Handling**: We now gracefully catch audio hardware and permission errors, and provide a `claune doctor` command to instantly diagnose system issues.
- **Package Manager Support**: Now available via Homebrew, `.deb`, and `.rpm` packages!
- **Auto-Update & Auto-Completions**: Stay up-to-date effortlessly with `claune update` and enjoy seamless tab-completions in bash, zsh, and powershell.
- **Simple Volume Controls**: Easily tweak your audio levels directly from the CLI via `claune volume <0-100>`, `claune mute`, and `claune unmute`.

### 📦 How to Install

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/everlier/claune/main/install.sh | bash
```

**Windows:**
```powershell
irm https://raw.githubusercontent.com/everlier/claune/main/install.ps1 | iex
```

**Homebrew (macOS/Linux):**
```bash
brew install everlier/tap/claune
```

### 🧑‍💻 Get Started
Once installed, just run:
```bash
claune setup
```
This will guide you through adding your first sounds. After that, just run `claune` and it will securely attach itself to Claude Desktop's event stream.

---

### ❤️ Open Source & Community
Claune is 100% open source and community-driven. If you want to contribute, share your custom sound packs, or report issues, visit our GitHub repository:
[GitHub - everlier/claune](https://github.com/everlier/claune)

Try it out and let us know what you think! We can't wait to see (and hear) how you customize your AI workspace.

*Note: Claune is an independent open-source project and is not affiliated with Anthropic.*
