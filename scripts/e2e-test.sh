#!/usr/bin/env bash
set -e

echo "Running end-to-end tests for Claune CLI..."

# Ensure we're using the freshly built binary
COMMAND="${CLAUNE_CMD:-claune}"

# Test 1: Version command works
echo "Testing 'version' command..."
$COMMAND version | grep -q "."
echo "-> OK"

# Test 2: Doctor command doesn't crash
echo "Testing 'doctor' command..."
$COMMAND doctor > /dev/null
echo "-> OK"

# Test 3: Initialize config
echo "Testing 'init' command..."
$COMMAND init > /dev/null || true
echo "-> OK"

# Test 4: Configure audio volume
echo "Testing 'volume' command..."
$COMMAND volume 50 > /dev/null
echo "-> OK"

# Test 5: Mute audio
echo "Testing 'mute' command..."
$COMMAND mute > /dev/null
echo "-> OK"

# Test 6: Unmute audio
echo "Testing 'unmute' command..."
$COMMAND unmute > /dev/null
echo "-> OK"

# Test Notify
echo "Testing 'notify on' command..."
$COMMAND notify on > /dev/null
echo "-> OK"

echo "Testing 'notify off' command..."
$COMMAND notify off > /dev/null
echo "-> OK"


# Test 7: Generate shell completions
echo "Testing 'completion' command..."
for shell in bash zsh powershell; do
    $COMMAND completion $shell > /dev/null
done
echo "-> OK"

# Test 8: Read logs
echo "Testing 'logs' command..."
$COMMAND logs > /dev/null
echo "-> OK"

# Test 9: Clear logs
echo "Testing 'logs clear' command..."
$COMMAND logs clear > /dev/null
echo "-> OK"

# Test 10: Play sounds
echo "Testing 'test-sounds' command..."
$COMMAND test-sounds > /dev/null
echo "-> OK"

echo "All End-to-End tests passed successfully!"
