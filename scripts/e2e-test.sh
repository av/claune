#!/usr/bin/env bash
set -euo pipefail

log() {
    printf '%s\n' "$1"
}

fail() {
    printf 'E2E FAILED: %s\n' "$1" >&2
    exit 1
}

assert_file_exists() {
    local path="$1"
    [[ -e "$path" ]] || fail "expected path to exist: $path"
}

assert_file_contains() {
    local path="$1"
    local needle="$2"
    grep -Fq -- "$needle" "$path" || fail "expected $path to contain: $needle"
}

assert_text_contains() {
    local text="$1"
    local needle="$2"
    [[ "$text" == *"$needle"* ]] || fail "expected output to contain: $needle"
}

run_success() {
    local output
    if ! output=$("$CLAUNE_BIN" "$@" 2>&1); then
        printf '%s\n' "$output" >&2
        fail "command failed: $*"
    fi
    printf '%s' "$output"
}

run_failure() {
    local output
    if output=$("$CLAUNE_BIN" "$@" 2>&1); then
        printf '%s\n' "$output" >&2
        fail "command unexpectedly succeeded: $*"
    fi
    printf '%s' "$output"
}

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
REAL_HOME=${HOME:-}

if [[ -n "${CLAUNE_CMD:-}" ]]; then
    CLAUNE_BIN="$CLAUNE_CMD"
elif [[ -x "$REPO_ROOT/claune" ]]; then
    CLAUNE_BIN="$REPO_ROOT/claune"
elif [[ -x "$REPO_ROOT/claune.exe" ]]; then
    CLAUNE_BIN="$REPO_ROOT/claune.exe"
else
    fail "CLAUNE_CMD is not set and no freshly built repo-local binary was found at $REPO_ROOT/claune"
fi

[[ -x "$CLAUNE_BIN" ]] || fail "claune binary is not executable: $CLAUNE_BIN"

PYTHON_BIN=$(command -v python3 || command -v python || true)
[[ -n "$PYTHON_BIN" ]] || fail "python3 or python is required for localhost HTTP fixtures"

mkdir -p "$REPO_ROOT/.tmp"
TEST_ROOT=$(mktemp -d "$REPO_ROOT/.tmp/e2e.XXXXXX")
WORKDIR="$TEST_ROOT/workdir"
FIXTURE_DIR="$TEST_ROOT/fixtures"
PORT_FILE="$TEST_ROOT/http-port"
SERVER_SCRIPT="$TEST_ROOT/http-fixture.py"
GLOBAL_BEFORE="$TEST_ROOT/global-before.json"
GLOBAL_AFTER="$TEST_ROOT/global-after.json"

cleanup() {
    local status=$?
    if [[ -n "${SERVER_PID:-}" ]]; then
        kill "$SERVER_PID" >/dev/null 2>&1 || true
        wait "$SERVER_PID" >/dev/null 2>&1 || true
    fi
    rm -rf "$TEST_ROOT"
    exit "$status"
}
trap cleanup EXIT

mkdir -p "$WORKDIR" "$FIXTURE_DIR"

snapshot_global_paths() {
    local destination="$1"
    "$PYTHON_BIN" - "$REAL_HOME" "$destination" <<'PY'
import json
import os
import pathlib
import sys

home = pathlib.Path(sys.argv[1])
destination = pathlib.Path(sys.argv[2])
targets = [
    home / ".config" / "claune",
    home / ".claune.json",
    home / ".cache" / "claune",
    home / ".local" / "state" / "claune",
    home / ".local" / "bin" / "claune",
    home / ".claude.json",
    home / ".claude" / "settings.json",
]

snapshot = {}
for target in targets:
    key = str(target)
    if not target.exists():
        snapshot[key] = None
        continue
    if target.is_dir():
        entries = []
        for root, dirs, files in os.walk(target):
            dirs.sort()
            files.sort()
            root_path = pathlib.Path(root)
            for name in dirs + files:
                path = root_path / name
                stat = path.stat()
                entries.append({
                    "path": str(path.relative_to(target)),
                    "is_dir": path.is_dir(),
                    "size": stat.st_size,
                    "mtime_ns": stat.st_mtime_ns,
                })
        snapshot[key] = {
            "is_dir": True,
            "entries": entries,
        }
    else:
        stat = target.stat()
        snapshot[key] = {
            "is_dir": False,
            "size": stat.st_size,
            "mtime_ns": stat.st_mtime_ns,
        }

destination.write_text(json.dumps(snapshot, indent=2, sort_keys=True))
PY
}

snapshot_global_paths "$GLOBAL_BEFORE"

export HOME="$TEST_ROOT/home"
export XDG_CONFIG_HOME="$TEST_ROOT/xdg/config"
export XDG_CACHE_HOME="$TEST_ROOT/xdg/cache"
export XDG_STATE_HOME="$TEST_ROOT/xdg/state"
export XDG_DATA_HOME="$TEST_ROOT/xdg/data"
export PATH="$WORKDIR/bin:$PATH"

mkdir -p \
    "$HOME" \
    "$XDG_CONFIG_HOME" \
    "$XDG_CACHE_HOME" \
    "$XDG_STATE_HOME" \
    "$XDG_DATA_HOME" \
    "$WORKDIR/bin"

cat > "$FIXTURE_DIR/import.mp3" <<'EOF'
not-a-real-mp3-but-fine-for-import-tests
EOF

cat > "$FIXTURE_DIR/invalid-pack.json" <<'EOF'
{"name":"bad-pack","description":
EOF

cat > "$SERVER_SCRIPT" <<'PY'
import http.server
import pathlib
import socketserver
import sys

root = pathlib.Path(sys.argv[1])
port_file = pathlib.Path(sys.argv[2])

class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(root), **kwargs)

    def log_message(self, format, *args):
        pass

with socketserver.TCPServer(("127.0.0.1", 0), Handler) as httpd:
    port_file.write_text(str(httpd.server_address[1]))
    httpd.serve_forever()
PY

"$PYTHON_BIN" "$SERVER_SCRIPT" "$FIXTURE_DIR" "$PORT_FILE" &
SERVER_PID=$!

for _ in 1 2 3 4 5 6 7 8 9 10; do
    [[ -s "$PORT_FILE" ]] && break
    sleep 1
done

[[ -s "$PORT_FILE" ]] || fail "localhost fixture server did not start"
PORT=$(tr -d '[:space:]' < "$PORT_FILE")
BASE_URL="http://127.0.0.1:$PORT"

cd "$WORKDIR"

log "Running hermetic end-to-end tests for Claune CLI..."

log "Testing 'version' command..."
VERSION_OUTPUT=$(run_success version)
assert_text_contains "$VERSION_OUTPUT" "claune version"

log "Testing 'doctor' command..."
DOCTOR_OUTPUT=$(run_success doctor)
assert_text_contains "$DOCTOR_OUTPUT" "Config File:"
assert_text_contains "$DOCTOR_OUTPUT" "$XDG_CONFIG_HOME/claune/config.json"

log "Testing 'init' command..."
INIT_OUTPUT=$(run_success init)
assert_text_contains "$INIT_OUTPUT" "Default configuration file created at:"
CONFIG_PATH="$XDG_CONFIG_HOME/claune/config.json"
assert_file_exists "$CONFIG_PATH"

log "Testing config mutation commands..."
run_success volume 50 >/dev/null
run_success mute >/dev/null
run_success unmute >/dev/null
run_success notify on >/dev/null
run_success notify off >/dev/null
assert_file_contains "$CONFIG_PATH" '"volume": 0.5'
assert_file_contains "$CONFIG_PATH" '"mute": false'
assert_file_contains "$CONFIG_PATH" '"notifications": false'

log "Testing 'completion' command..."
run_success completion bash >/dev/null
run_success completion zsh >/dev/null
run_success completion powershell >/dev/null

log "Testing 'install' command..."
INSTALL_OUTPUT=$(run_success install)
assert_text_contains "$INSTALL_OUTPUT" "$HOME/.claude/settings.json"
CLAUDE_SETTINGS_PATH="$HOME/.claude/settings.json"
assert_file_exists "$CLAUDE_SETTINGS_PATH"
assert_file_contains "$CLAUDE_SETTINGS_PATH" '"hooks"'
assert_file_contains "$CLAUDE_SETTINGS_PATH" 'CLAUNE_ACTIVE'
assert_file_contains "$CLAUDE_SETTINGS_PATH" 'tool:success'

log "Testing 'status' command after install..."
STATUS_OUTPUT=$(run_success status)
assert_text_contains "$STATUS_OUTPUT" "Installed — claune hooks are active in Claude Code."

log "Testing 'logs' command before log creation..."
LOGS_OUTPUT=$(run_success logs)
assert_text_contains "$LOGS_OUTPUT" "No logs found at"

log "Testing 'play' failure path writes hermetic logs..."
PLAY_FAILURE_OUTPUT=$(run_failure play not-a-real-event)
assert_text_contains "$PLAY_FAILURE_OUTPUT" "Error playing sound: unknown event type: not-a-real-event"
LOG_PATH="$XDG_STATE_HOME/claune/claune.log"
assert_file_exists "$LOG_PATH"
assert_file_contains "$LOG_PATH" 'unknown event type: not-a-real-event'

log "Testing 'logs' and 'logs clear' commands..."
LOGS_WITH_ERROR=$(run_success logs)
assert_text_contains "$LOGS_WITH_ERROR" "$LOG_PATH"
assert_text_contains "$LOGS_WITH_ERROR" "unknown event type: not-a-real-event"
run_success logs clear >/dev/null
[[ ! -e "$LOG_PATH" ]] || fail "expected logs clear to remove $LOG_PATH"

log "Testing hermetic 'import-circus' success path over localhost HTTP..."
IMPORT_OUTPUT=$(run_success import-circus "$BASE_URL/import.mp3" imported-test.mp3 tool:error)
assert_text_contains "$IMPORT_OUTPUT" "Imported imported-test.mp3 and mapped it to event tool:error"
IMPORTED_SOUND_PATH="$XDG_CACHE_HOME/claune/imported-test.mp3"
assert_file_exists "$IMPORTED_SOUND_PATH"
assert_file_contains "$CONFIG_PATH" '"tool:error"'
assert_file_contains "$CONFIG_PATH" 'imported-test.mp3'

log "Testing 'import-circus' rejects file:// URLs..."
FILE_URL_FAILURE=$(run_failure import-circus "file://$FIXTURE_DIR/import.mp3" rejected-file-url.mp3 tool:error)
assert_text_contains "$FILE_URL_FAILURE" "Import failed:"

log "Testing 'pack' URL handling stays on supported HTTP and fails cleanly on invalid JSON..."
PACK_FAILURE=$(run_failure pack ignored "$BASE_URL/invalid-pack.json")
assert_text_contains "$PACK_FAILURE" "failed to parse custom pack JSON from URL"

log "Testing 'uninstall' command..."
UNINSTALL_OUTPUT=$(run_success uninstall)
assert_text_contains "$UNINSTALL_OUTPUT" "Hooks removed from $CLAUDE_SETTINGS_PATH"

snapshot_global_paths "$GLOBAL_AFTER"
cmp -s "$GLOBAL_BEFORE" "$GLOBAL_AFTER" || fail "user-global claune or Claude paths changed outside the hermetic test environment"

for expected_path in \
    "$CONFIG_PATH" \
    "$XDG_CACHE_HOME/claune" \
    "$XDG_STATE_HOME/claune" \
    "$CLAUDE_SETTINGS_PATH"; do
    assert_file_exists "$expected_path"
done

log "All hermetic end-to-end tests passed successfully!"
