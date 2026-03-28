package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HookEntry represents a single hook configuration entry
type HookEntry struct {
	Matcher string `json:"matcher"`
	Hooks   []Hook `json:"hooks"`
}

// Hook represents a hook command
type Hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

// settingsPathFunc is a variable so tests can override it
var settingsPathFunc = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

func settingsPath() string {
	return settingsPathFunc()
}

// resolveClauneBinFunc is a variable so tests can override it
var resolveClauneBinFunc = func() string {
	if path, err := exec.LookPath("claune"); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "claune"
}

func resolveClauneBin() string {
	return resolveClauneBinFunc()
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

func containsClaunePlay(cmd string) bool {
	return strings.Contains(cmd, "claune play")
}

// mergeHooks adds new hook entries without duplicating existing claune hooks
func mergeHooks(existing []HookEntry, newHooks []HookEntry) []HookEntry {
	if len(existing) == 0 {
		return newHooks
	}

	existingCmds := make(map[string]bool)
	for _, entry := range existing {
		for _, hook := range entry.Hooks {
			if containsClaunePlay(hook.Command) {
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

// removeClauneHooks filters out any hook entries that contain claune play commands
func removeClauneHooks(entries []HookEntry) []HookEntry {
	var kept []HookEntry
	for _, entry := range entries {
		isClaune := false
		for _, hook := range entry.Hooks {
			if containsClaunePlay(hook.Command) {
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

func clauneHookEntries() map[string][]HookEntry {
	bin := resolveClauneBin()
	return map[string][]HookEntry{
		"PreToolUse": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type: "command", Command: bin + " play tool:start", Timeout: 5,
			}},
		}},
		"PostToolUse": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type: "command", Command: bin + " play tool:success", Timeout: 5,
			}},
		}},
		"PostToolUseFailure": {{
			Matcher: ".*",
			Hooks: []Hook{{
				Type: "command", Command: bin + " play tool:error", Timeout: 5,
			}},
		}},
	}
}

func installHooks() error {
	settings, err := readSettings()
	if err != nil {
		return err
	}

	// Get or create hooks map
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

	fmt.Println("Hooks installed into " + settingsPath())
	fmt.Println("Sound effects are now active in Claude Code.")
	fmt.Println("Config: ~/.claune.json (mute, volume, custom sounds)")
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
	for _, key := range []string{"PreToolUse", "PostToolUse", "PostToolUseFailure"} {
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
	for _, key := range []string{"PreToolUse", "PostToolUse", "PostToolUseFailure"} {
		entries := parseHookEntries(hooksMap, key)
		for _, entry := range entries {
			for _, hook := range entry.Hooks {
				if containsClaunePlay(hook.Command) {
					return true
				}
			}
		}
	}
	return false
}

func showStatus() {
	if hooksInstalled() {
		fmt.Println("Installed — claune hooks are active in Claude Code.")
	} else {
		fmt.Println("Not installed — run 'claune install' to add sound hooks.")
	}

	config := getConfig()
	if shouldMute(config) {
		fmt.Println("Sound: muted")
	} else {
		fmt.Printf("Volume: %.0f%%\n", getVolume(config)*100)
	}

	player, _ := findAudioPlayer()
	if player != "" {
		fmt.Println("Audio player: " + player)
	} else {
		fmt.Println("Audio player: none found (install paplay, aplay, or afplay)")
	}
}

func testSounds() {
	fmt.Println("Testing all sounds...")
	for _, event := range []string{"cli:start", "tool:start", "tool:success", "tool:error", "cli:done"} {
		fmt.Printf("  %s ", event)
		config := getConfig()
		volume := getVolume(config)
		soundFile, ok := defaultSoundMap[event]
		if !ok {
			fmt.Println("— no mapping")
			continue
		}
		if err := playEmbeddedSound(soundFile, volume, true); err != nil {
			fmt.Printf("— error: %v\n", err)
		} else {
			fmt.Println("OK")
		}
	}
}
