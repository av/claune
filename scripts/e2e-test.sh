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

# Test 11: Pack custom URL
echo "Testing 'pack <url>' command..."
cat << 'EOF' > /tmp/test-pack.json
{
  "name": "e2e-test-pack",
  "description": "A dummy pack for E2E tests",
  "sounds": {
    "cli:start": "anime-wow"
  }
}
EOF
python3 -m http.server 12345 --directory /tmp >/dev/null 2>&1 &
SERVER_PID=$!
sleep 1
$COMMAND pack http://localhost:12345/test-pack.json > /dev/null
kill $SERVER_PID || true
echo "-> OK"

# Test 12: Pack local file
echo "Testing 'pack <local_file>' command..."
cat << 'EOF' > /tmp/test-local-pack.json
{
  "name": "e2e-local-pack",
  "description": "A dummy local pack for E2E tests",
  "sounds": {
    "cli:success": "anime-wow"
  }
}
EOF
$COMMAND pack /tmp/test-local-pack.json > /dev/null
echo "-> OK"

# Test 13: Pack invalid JSON file
echo "Testing 'pack <invalid_file>' command..."
cat << 'EOF' > /tmp/test-invalid-pack.json
{
  "name": "e2e-invalid-pack"
  "description": "Missing comma"
}
EOF
if $COMMAND pack /tmp/test-invalid-pack.json > /dev/null 2>&1; then
    echo "Expected pack to fail with invalid JSON"
    exit 1
fi
echo "-> OK"

echo "All End-to-End tests passed successfully!"
