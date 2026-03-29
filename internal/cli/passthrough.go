package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

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
	home, _ := os.UserHomeDir()
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
	if len(existing) == 0 {
		return newHooks
	}
	existingCmds := make(map[string]bool)
	for _, entry := range existing {
		for _, hook := range entry.Hooks {
			if isClauneHook(hook.Command) {
				existingCmds[hook.Command] = true
			}
		}
	}
	for _, entry := range newHooks {
		alreadyExists := true
		for _, hook := range entry.Hooks {
			if !existingCmds[hook.Command] {
				alreadyExists = false
				break
			}
		}
		if !alreadyExists {
			existing = append(existing, entry)
		}
	}
	return existing
}

func removeClauneHooks(entries []HookEntry) []HookEntry {
	var kept []HookEntry
	for _, entry := range entries {
		isClaune := false
		for _, hook := range entry.Hooks {
			if isClauneHook(hook.Command) {
				isClaune = true
				break
			}
		}
		if !isClaune {
			kept = append(kept, entry)
		}
	}
	return kept
}

func directHookCmd(wavPath string, event string) string {
	// Fallback to slow claune play
	bin := "claune"
	if path, err := exec.LookPath("claune"); err == nil {
		bin = path
	}
	return `bash -c '[ "$CLAUNE_ACTIVE" = "1" ] && ` + bin + ` play ` + event + ` &'`
}

func clauneHookEntries() map[string][]HookEntry {
	cacheDir := audio.SoundCacheDir()
	return map[string][]HookEntry{
		"SessionStart": {{
			Matcher: "",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["cli:start"]), "cli:start"),
				Timeout: 5,
			}},
		}},
		"PreToolUse": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["tool:start"]), "tool:start"),
				Timeout: 5,
			}},
		}},
		"PostToolUse": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["tool:success"]), "tool:success"),
				Timeout: 5,
			}},
		}},
		"PostToolUseFailure": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["tool:error"]), "tool:error"),
				Timeout: 5,
			}},
		}},
		"SessionEnd": {{
			Matcher: "",
			Hooks: []Hook{{
				Type:    "command",
				Command: directHookCmd(filepath.Join(cacheDir, audio.DefaultSoundMap["cli:done"]), "cli:done"),
				Timeout: 5,
			}},
		}},
	}
}

func installHooks() error {
	settings, err := readSettings()
	if err != nil {
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
		filtered := removeClauneHooks(entries)
		if len(filtered) != len(entries) {
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
	settings["hooks"] = hooksMap
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

func playViaShell(event string, c config.ClauneConfig) {
	if c.ShouldMute() {
		return
	}
	soundFile, ok := audio.DefaultSoundMap[event]
	if !ok {
		return
	}
	cached := filepath.Join(audio.SoundCacheDir(), soundFile)
	if _, err := os.Stat(cached); err != nil {
		return
	}
	cmd := audio.ShellPlayCmd(cached, c.GetVolume())
	if cmd == "" {
		fmt.Fprintln(os.Stderr, "🔇 Audio unavailable: no supported audio player found (paplay, pw-play, aplay, afplay)")
		return
	}
	exec.Command("bash", "-c", cmd+" &").Run()
}

func runPassthrough(args []string) {
	audio.EnsureSoundCache()
	c := config.Load()
	playViaShell("cli:start", c)

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

	os.Setenv("CLAUNE_ACTIVE", "1")
	syscall.Exec(claudeBin, append([]string{claudeBin}, args...), os.Environ())

	fmt.Fprintf(os.Stderr, "claune: failed to exec %s\n", claudeBin)
	os.Exit(1)
}
