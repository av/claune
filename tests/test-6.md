# Test 6: Strict Configuration Parsing & Fallback UX

## Prerequisites
- The `claune` binary must be compiled and available in the system PATH.
- Run `go build -o ~/.local/bin/claune ./cmd/claune`

## Test 1: Invalid Configuration Rejection
**Steps:**
1. Manually edit `~/.claune.json` to use a deprecated legacy format:
   ```json
   {
     "mute": false,
     "sounds": { "tool:success": "my-sound.mp3" }
   }
   ```
2. Run `claune test-sounds`.

**Expectations:**
1. The command must exit with a non-zero exit code.
2. The standard error output must contain "invalid config format detected in ~/.claune.json" and/or "Sounds must now be configured as objects with 'paths' array, not strings."

## Test 2: Graceful File Fallback
**Steps:**
1. Manually edit `~/.claune.json` to use a valid schema but pointing to a non-existent file:
   ```json
   {
     "mute": false,
     "sounds": {
       "tool:success": {
         "paths": ["/path/to/nowhere.mp3"]
       }
     }
   }
   ```
2. Run `claune play tool:success`.

**Expectations:**
1. The command must exit with a 0 exit code.
2. The standard error output must contain "Warning: invalid custom sound path" and "/path/to/nowhere.mp3".
3. The command should successfully execute, implying it played the fallback sound.
