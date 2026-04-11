#!/bin/sh
set -e

# Claune Installer

REPO="av/claune"

echo "Installing Claune..."

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS" && exit 1 ;;
esac

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

echo "Detected OS: $OS, Architecture: $ARCH"

# Find latest release
if command -v curl >/dev/null 2>&1; then
    LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
else
    echo "Error: curl is required to download Claune."
    exit 1
fi

if [ -z "$LATEST_TAG" ]; then
    echo "Error: Could not determine latest release version."
    exit 1
fi

echo "Latest version: $LATEST_TAG"

# Construct download URL
FILENAME="claune-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${FILENAME}"

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

echo "Downloading $FILENAME to $INSTALL_DIR/claune..."

if command -v curl >/dev/null 2>&1; then
    curl -sL "$DOWNLOAD_URL" -o "$INSTALL_DIR/claune"
else
    echo "Error: curl is required to download Claune."
    exit 1
fi

chmod +x "$INSTALL_DIR/claune"

echo ""
echo "Claune installed successfully to $INSTALL_DIR/claune"

echo ""
echo "Setting up shell completions..."
if [ -d "$HOME/.local/share/bash-completion/completions" ]; then
    if "$INSTALL_DIR/claune" completion bash > "$HOME/.local/share/bash-completion/completions/claune" 2>/dev/null; then
        echo "Installed bash completion."
    fi
elif [ -d "$HOME/.bash_completion.d" ]; then
    if "$INSTALL_DIR/claune" completion bash > "$HOME/.bash_completion.d/claune" 2>/dev/null; then
        echo "Installed bash completion."
    fi
fi

if [ -d "$HOME/.local/share/zsh/site-functions" ]; then
    if "$INSTALL_DIR/claune" completion zsh > "$HOME/.local/share/zsh/site-functions/_claune" 2>/dev/null; then
        echo "Installed zsh completion."
    fi
fi


echo ""

if [ -d "$HOME/.local/share/zsh/site-functions" ]; then
    "$INSTALL_DIR/claune" completion zsh > "$HOME/.local/share/zsh/site-functions/_claune" 2>/dev/null || true
    echo "Installed zsh completion."
fi


if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "Warning: $INSTALL_DIR is not in your PATH."
    echo "You may need to add 'export PATH=\"\$HOME/.local/bin:\$PATH\"' to your shell configuration (e.g. ~/.bashrc or ~/.zshrc)."
fi

echo ""
echo "To get started, run:"
echo "  claune install"
echo ""
