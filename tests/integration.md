# Integration Test Suite

This suite outlines how a user would interact with the CLI to verify all major features, edge cases, and the AI workflows.

### 1. Installation & Claude Hooks Workflow
**Objective:** Verify that `claune` correctly attaches and detaches its sound hooks into the Claude Code settings.
* **Steps:**
  1. Run `claune install` in the terminal.
  2. Open the Claude Code global configuration file (usually `~/.claude.json` or equivalent) and look for the hook entries.
  3. Run `claune uninstall` in the terminal.
  4. Open the Claude Code global configuration file again.
* **Verify:**
  - After step 1, the CLI outputs a success message, and the Claude config contains `PreToolUse`, `PostToolUse`, etc., pointing to the `claune play` commands.
  - After step 3, the CLI outputs a success message, and all `claune` specific hooks are completely removed from the config without breaking other native settings.

### 2. Basic Playback & Passthrough Workflow
**Objective:** Verify that the built-in sounds work and that `claune` successfully wraps a standard Claude session.
* **Steps:**
  1. Run `claune test-sounds`.
  2. Run `claune play build:success`.
  3. Run `claune` to start an interactive Claude session, and ask it to run a simple bash command (e.g., `ls`).
* **Verify:**
  - `test-sounds` iterates through all default events and plays their embedded `.mp3` files sequentially.
  - `play build:success` plays the specific coin/yippee sound immediately and exits.
  - Running `claune` successfully passes through to Claude Code, and sounds trigger automatically when tools (like bash) start and finish.

### 3. Natural Language Configuration Workflow
**Objective:** Verify that the AI correctly parses user intent to update `~/.claune.json` settings.
* **Steps:**
  1. Run `claune config "mute all sounds for the next 2 hours"`.
  2. Run `claune test-sounds`.
  3. Run `claune config "set volume to 50% and unmute"`.
  4. Run `claune test-sounds`.
* **Verify:**
  - After step 1, `~/.claune.json` is updated with a `mute_until` timestamp.
  - In step 2, no sounds play and the CLI returns silently.
  - After step 3, `~/.claune.json` reflects `"volume": 0.5` and the `mute_until` field is removed or set to the past.
  - In step 4, the sounds play at half volume.

### 4. Downloading and Mapping Meme Sounds (Circus Importer)
**Objective:** Verify that users can download arbitrary sounds from the internet and map them to events in a single command.
* **Steps:**
  1. Run `claune import-circus https://boardsounds.com/api/sound/play/123-bruh.mp3 bruh.mp3 tool:error`.
  2. Run `claune play tool:error`.
* **Verify:**
  - The CLI outputs a success message indicating the file was downloaded and mapped.
  - The config file is updated to include `bruh.mp3` under the `tool:error` event.
  - Step 2 plays the newly downloaded "bruh" sound instead of the default embedded error sound.

### 5. Playback Strategies (Round-Robin vs. Random)
**Objective:** Verify that providing multiple sounds for a single event properly cycles through them based on the configuration strategy.
* **Steps:**
  1. Manually edit `~/.claune.json` to assign an array of three distinct sound paths to `tool:success`.
  2. Set `"strategy": "round_robin"` inside the `tool:success` object.
  3. Run `claune play tool:success` four times in a row.
  4. Change the strategy in the config to `"random"`.
  5. Run `claune play tool:success` several times.
* **Verify:**
  - In step 3, the CLI plays Sound A, then Sound B, then Sound C, and wraps around back to Sound A.
  - In step 5, the CLI plays sounds non-sequentially, proving the random fallback works.

### 6. Strict Configuration Parsing & Fallback UX
**Objective:** Verify that invalid configurations are rejected loudly, and missing files are handled gracefully.
* **Steps:**
  1. Manually edit `~/.claune.json` to use the deprecated legacy format (e.g., `"sounds": { "tool:success": "my-sound.mp3" }`).
  2. Run `claune test-sounds`.
  3. Fix the configuration to the proper new object format, but provide a file path that *does not exist* (e.g., `["/path/to/nowhere.mp3"]`).
  4. Run `claune play tool:success`.
* **Verify:**
  - In step 2, the application refuses to run, printing a clear parsing/schema error to standard error.
  - In step 4, the application logs a visible warning to standard error (`Warning: invalid custom sound path...`) but does not crash, successfully playing the default embedded sound instead.

### 7. AI Auto-Mapping Local Sounds
**Objective:** Verify the builder's new AI auto-mapping feature dynamically assigns local files to appropriate events.
* **Steps:**
  1. Create a local directory `~/my-sounds/` and copy two MP3s into it named `triumphant-horn.mp3` and `bomb-explosion.mp3`.
  2. Run `claune automap ~/my-sounds/`.
  3. Inspect `~/.claune.json`.
* **Verify:**
  - The CLI triggers an Anthropic API call and outputs a success message.
  - The config file is updated using the strict schema: `triumphant-horn.mp3` is intelligently mapped to `tool:success` or `build:success`, while `bomb-explosion.mp3` is mapped to `tool:destructive` or `panic`.

### 8. AI Sentiment / Urgency Override
**Objective:** Verify that AI log analysis can dynamically override the normal sound strategy for critical events.
* **Steps:**
  1. Run `claune analyze-resp "Just checking the working directory, nothing special."`
  2. Run `claune analyze-resp "CRITICAL PANIC: The entire database was accidentally dropped and no backups are found!"`
* **Verify:**
  - Step 1 results in either a neutral sound or a silent exit (depending on threshold logic).
  - Step 2 successfully detects the high-urgency sentiment and immediately plays the `panic` sound, bypassing whatever `round_robin` or default settings were queued for standard tool usage.
