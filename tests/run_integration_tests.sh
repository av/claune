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

# Test 6, Part 1: Invalid Configuration Rejection
echo "Test 6.1: Invalid Configuration Rejection"
cat << 'EOF' > "$TEST_CONFIG_FILE"
{
  "mute": false,
  "sounds": { "tool:success": "my-sound.mp3" }
}
EOF

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
cat << 'EOF' > "$TEST_CONFIG_FILE"
{
  "mute": false,
  "sounds": {
    "tool:success": {
      "paths": ["/path/to/nowhere.mp3"]
    }
  }
}
EOF

output=$(claune play tool:success 2>&1 || true)
if echo "$output" | grep -q "Warning: invalid custom sound path"; then
    echo "✅ Test 6.2 Passed"
else
    echo "❌ Test 6.2 Failed. Output:"
    echo "$output"
    exit 1
fi

echo "All Integration Tests Passed successfully!"
