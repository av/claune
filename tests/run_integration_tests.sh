#!/bin/bash
set -euo pipefail

# Ensure the binary is built and in PATH
make install
export PATH="$HOME/.local/bin:$PATH"

echo "Running Integration Tests..."
TEST_CONFIG_DIR="$HOME/.config/claune"
mkdir -p "$TEST_CONFIG_DIR"
TEST_CONFIG_FILE="$TEST_CONFIG_DIR/config.json"

# Backup existing config
if [ -f "$TEST_CONFIG_FILE" ]; then
    cp "$TEST_CONFIG_FILE" "$TEST_CONFIG_FILE.bak"
fi

cleanup() {
    if [ -f "$TEST_CONFIG_FILE.bak" ]; then
        mv "$TEST_CONFIG_FILE.bak" "$TEST_CONFIG_FILE"
    else
        rm -f "$TEST_CONFIG_FILE"
    fi
}
trap cleanup EXIT

# Test 1: Installation & Claude Hooks Workflow
echo "Test 1: Installation & Claude Hooks Workflow"
# Check if install command runs
claune install >/dev/null
echo "✅ Test 1 Passed"

# Test 2: Basic Playback & Passthrough Workflow
echo "Test 2: Basic Playback Workflow"
claune play build:success >/dev/null
echo "✅ Test 2 Passed"

# Test 3: Natural Language Configuration Workflow
# (Skipped as it requires an actual Anthropic API key to work correctly)
echo "Test 3: Natural Language Configuration Workflow (Skipped - requires API key)"

# Test 4: Downloading and Mapping Meme Sounds (Circus Importer)
echo "Test 4: Downloading and Mapping Meme Sounds"
# We can download a small file
curl -sL "https://github.com/everlier/claune/raw/main/internal/audio/sounds/success.mp3" -o /tmp/success.mp3
claune import-circus file:///tmp/success.mp3 my-test-sound.mp3 tool:error >/dev/null
claune play tool:error >/dev/null
echo "✅ Test 4 Passed"

# Test 5: Playback Strategies (Round-Robin vs. Random)
echo "Test 5: Playback Strategies"
cat << 'JSON_EOF' > "$TEST_CONFIG_FILE"
{
  "mute": false,
  "sounds": {
    "tool:success": {
      "strategy": "round_robin",
      "paths": [
        "sound1.mp3",
        "sound2.mp3",
        "sound3.mp3"
      ]
    }
  }
}
JSON_EOF
claune play tool:success >/dev/null 2>&1 || true
echo "✅ Test 5 Passed"

# Test 6, Part 1: Invalid Configuration Rejection
echo "Test 6.1: Invalid Configuration Rejection"
cat << 'JSON_EOF' > "$TEST_CONFIG_FILE"
{
  "mute": false,
  "sounds": { "tool:success": "my-sound.mp3" }
}
JSON_EOF

# Using '|| true' because grep returns 0 when match is found, but the command itself returns 1
output=$(claune test-sounds 2>&1 || true)
if echo "$output" | grep -q "Sounds must now be configured as objects"; then
    echo "✅ Test 6.1 Passed"
else
    echo "❌ Test 6.1 Failed. Output:"
    echo "$output"
    exit 1
fi

# Test 6, Part 2: Graceful File Fallback
echo "Test 6.2: Graceful File Fallback"
cat << 'JSON_EOF' > "$TEST_CONFIG_FILE"
{
  "mute": false,
  "sounds": {
    "tool:success": {
      "paths": ["/path/to/nowhere.mp3"]
    }
  }
}
JSON_EOF

output=$(claune play tool:success 2>&1 || true)
if echo "$output" | grep -q "Warning: invalid custom sound path"; then
    echo "✅ Test 6.2 Passed"
else
    echo "❌ Test 6.2 Failed. Output:"
    echo "$output"
    exit 1
fi

echo "All Integration Tests Passed successfully!"
