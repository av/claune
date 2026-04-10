package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/everlier/claune/internal/ai"
	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/config"
)

type HookEntry struct {
	Matcher string `json:"matcher"`
	Hooks   []Hook `json:"hooks"`
}

type Hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

var settingsPathFunc = func() string {
	cwd, err := os.Getwd()
	if err == nil {
		localPath := filepath.Join(cwd, ".claude.json")
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
	}
	home, _ := os.UserHomeDir()
	path1 := filepath.Join(home, ".claude.json")
	if _, err := os.Stat(path1); err == nil {
		return path1
	}
	path2 := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(path2); err == nil {
		return path2
	}
	return filepath.Join(home, ".claude", "settings.json")
}

func settingsPath() string {
	return settingsPathFunc()
}

func readSettings() (map[string]interface{}, error) {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", settingsPath(), err)
	}
	return settings, nil
}

func writeSettings(settings map[string]interface{}) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(settingsPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return os.WriteFile(settingsPath(), append(data, '\n'), 0644)
}

func parseHookEntries(hooksMap map[string]interface{}, key string) []HookEntry {
	raw, ok := hooksMap[key]
	if !ok || raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var entries []HookEntry
	json.Unmarshal(data, &entries)
	return entries
}

func isClauneHook(cmd string) bool {
	return strings.Contains(cmd, "claune play") || strings.Contains(cmd, ".cache/claune/")
}

func mergeHooks(existing []HookEntry, newHooks []HookEntry) []HookEntry {
	kept, _ := removeClauneHooks(existing)
	return append(kept, newHooks...)
}

func removeClauneHooks(entries []HookEntry) ([]HookEntry, bool) {
	var kept []HookEntry
	changed := false
	for _, entry := range entries {
		var keptHooks []Hook
		for _, hook := range entry.Hooks {
			if isClauneHook(hook.Command) {
				changed = true
			} else {
				keptHooks = append(keptHooks, hook)
			}
		}
		if len(keptHooks) > 0 {
			entry.Hooks = keptHooks
			kept = append(kept, entry)
		}
	}
	return kept, changed
}

func directHookCmd(wavPath string, event string) string {
	// Fallback to slow claune play
	bin := "claune"
	if path, err := exec.LookPath("claune"); err == nil {
		bin = path
	}
	return `bash -c '[ "$CLAUNE_ACTIVE" = "1" ] && ` + bin + ` play ` + event + ` >/dev/null 2>&1 &'`
}

func clauneHookEntries() map[string][]HookEntry {
	cacheDir := audio.SoundCacheDir()
	return map[string][]HookEntry{
		"SessionStart": {{
			Matcher: "",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["cli:start"][0]), "cli:start"),
				Timeout: 5,
			}},
		}},
		"PreToolUse": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["tool:start"][0]), "tool:start"),
				Timeout: 5,
			}},
		}},
		"PostToolUse": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["tool:success"][0]), "tool:success"),
				Timeout: 5,
			}},
		}},
		"PostToolUseFailure": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["tool:error"][0]), "tool:error"),
				Timeout: 5,
			}},
		}},
		"SessionEnd": {{
			Matcher: "",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["cli:done"][0]), "cli:done"),
				Timeout: 5,
			}},
		}},
	}
}

func installHooks() error {
	settings, err := readSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading settings: %v\n", err)
		c, _ := config.Load()
		fmt.Fprintf(os.Stderr, "AI Diagnostics: %s\n", ai.DiagnoseInstallFailure(err, c))
		return err
	}
	var hooksMap map[string]interface{}
	if raw, ok := settings["hooks"]; ok {
		if hm, ok := raw.(map[string]interface{}); ok {
			hooksMap = hm
		}
	}
	if hooksMap == nil {
		hooksMap = make(map[string]interface{})
	}

	for key, newEntries := range clauneHookEntries() {
		existing := parseHookEntries(hooksMap, key)
		merged := mergeHooks(existing, newEntries)
		hooksMap[key] = merged
	}

	settings["hooks"] = hooksMap
	if err := writeSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing settings: %v\n", err)
		c, _ := config.Load()
		fmt.Fprintf(os.Stderr, "AI Diagnostics: %s\n", ai.DiagnoseInstallFailure(err, c))
		return err
	}

	if err := audio.EnsureSoundCache(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not cache sounds: %v\n", err)
	}

	fmt.Println("Hooks installed into " + settingsPath())
	return nil
}

func uninstallHooks() error {
	settings, err := readSettings()
	if err != nil {
		return err
	}
	hooksMap, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		fmt.Println("No hooks found — nothing to remove.")
		return nil
	}
	changed := false
	for _, key := range []string{"SessionStart", "PreToolUse", "PostToolUse", "PostToolUseFailure", "SessionEnd"} {
		entries := parseHookEntries(hooksMap, key)
		filtered, keyChanged := removeClauneHooks(entries)
		if keyChanged {
			changed = true
		}
		if len(filtered) == 0 {
			delete(hooksMap, key)
		} else {
			hooksMap[key] = filtered
		}
	}
	if !changed {
		fmt.Println("No claune hooks found — nothing to remove.")
		return nil
	}
	if len(hooksMap) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooksMap
	}
	if err := writeSettings(settings); err != nil {
		return err
	}
	fmt.Println("Hooks removed from " + settingsPath())
	return nil
}

func hooksInstalled() bool {
	settings, err := readSettings()
	if err != nil {
		return false
	}
	hooksMap, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, key := range []string{"SessionStart", "PreToolUse", "PostToolUse", "PostToolUseFailure", "SessionEnd"} {
		entries := parseHookEntries(hooksMap, key)
		for _, entry := range entries {
			for _, hook := range entry.Hooks {
				if isClauneHook(hook.Command) {
					return true
				}
			}
		}
	}
	return false
}

func runPassthrough(args []string) {
	audio.EnsureSoundCache()
	c, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claune: error loading config: %v\n", err)
		os.Exit(1)
	}
	audio.PlaySound("cli:start", false, c)

	if !hooksInstalled() {
		if err := installHooks(); err != nil {
			fmt.Fprintf(os.Stderr, "claune: warning: could not install hooks: %v\n", err)
		}
	}

	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintf(os.Stderr, "claune: claude not found in PATH\n")
		os.Exit(1)
	}

	myExe, err := os.Executable()
	if err == nil {
		myExeEval, err1 := filepath.EvalSymlinks(myExe)
		claudeExeEval, err2 := filepath.EvalSymlinks(claudeBin)
		if err1 == nil && err2 == nil && myExeEval == claudeExeEval {
			fmt.Fprintf(os.Stderr, "claune: fork bomb detected! 'claude' resolves to this executable.\n")
			os.Exit(1)
		}
	}

	if os.Getenv("CLAUNE_ACTIVE") == "1" {
		fmt.Fprintf(os.Stderr, "claune: nested execution detected! CLAUNE_ACTIVE is already set.\n")
		os.Exit(1)
	}

	cmd := exec.Command(claudeBin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "CLAUNE_ACTIVE=1")

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "claune: %v\n", err)
		os.Exit(1)
	}
}
