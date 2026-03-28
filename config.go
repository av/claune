package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

// resolveClauneBin finds the absolute path to the claune binary
func resolveClauneBin() string {
	// Check PATH first
	if path, err := exec.LookPath("claune"); err == nil {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs
		}
		return path
	}

	// Fallback to current executable
	if exe, err := os.Executable(); err == nil {
		return exe
	}

	return "claune"
}

// mergeHooks merges new hook entries into existing ones without duplicating
func mergeHooks(existing []HookEntry, newHooks []HookEntry) []HookEntry {
	if len(existing) == 0 {
		return newHooks
	}

	// Collect all existing claune play commands
	existingCmds := make(map[string]bool)
	for _, entry := range existing {
		for _, hook := range entry.Hooks {
			cmd := hook.Command
			if containsClaunePlay(cmd) {
				existingCmds[cmd] = true
			}
		}
	}

	// Only add hooks whose commands aren't already present
	for _, entry := range newHooks {
		dominated := true
		for _, hook := range entry.Hooks {
			if !existingCmds[hook.Command] {
				dominated = false
				break
			}
		}
		if !dominated {
			existing = append(existing, entry)
		}
	}

	return existing
}

func containsClaunePlay(cmd string) bool {
	return len(cmd) >= 11 && // len("claune play") == 11
		contains(cmd, "claune play")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// buildHookConfig creates a temp config file with claune hooks injected
func buildHookConfig() (string, error) {
	home, _ := os.UserHomeDir()

	// Read base Claude settings
	configData := make(map[string]interface{})
	claudeConfigPath := filepath.Join(home, ".claude.json")
	if data, err := os.ReadFile(claudeConfigPath); err == nil {
		json.Unmarshal(data, &configData)
	}

	// Merge non-sound claune settings
	clauneConfigPath := filepath.Join(home, ".claune.json")
	if data, err := os.ReadFile(clauneConfigPath); err == nil {
		var clauneData map[string]interface{}
		if json.Unmarshal(data, &clauneData) == nil {
			for k, v := range clauneData {
				if k != "mute" && k != "volume" && k != "sounds" {
					configData[k] = v
				}
			}
		}
	}

	clauneBin := resolveClauneBin()

	// Get or create hooks map
	hooksRaw, _ := configData["hooks"]
	var hooksMap map[string]interface{}
	if hooksRaw != nil {
		if hm, ok := hooksRaw.(map[string]interface{}); ok {
			hooksMap = hm
		}
	}
	if hooksMap == nil {
		hooksMap = make(map[string]interface{})
	}

	// Parse existing hook entries
	parseHookEntries := func(key string) []HookEntry {
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

	// PreToolUse
	preToolUse := parseHookEntries("PreToolUse")
	preToolUse = mergeHooks(preToolUse, []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: clauneBin + " play tool:start",
				Timeout: 5,
			}},
		},
	})

	// PostToolUse
	postToolUse := parseHookEntries("PostToolUse")
	postToolUse = mergeHooks(postToolUse, []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: clauneBin + " play tool:success",
				Timeout: 5,
			}},
		},
	})

	// PostToolUseFailure
	postToolUseFail := parseHookEntries("PostToolUseFailure")
	postToolUseFail = mergeHooks(postToolUseFail, []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: clauneBin + " play tool:error",
				Timeout: 5,
			}},
		},
	})

	hooksMap["PreToolUse"] = preToolUse
	hooksMap["PostToolUse"] = postToolUse
	hooksMap["PostToolUseFailure"] = postToolUseFail
	configData["hooks"] = hooksMap

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "claune_config_*.json")
	if err != nil {
		return "", err
	}

	encoder := json.NewEncoder(tmpFile)
	if err := encoder.Encode(configData); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}
	tmpFile.Close()

	return tmpFile.Name(), nil
}
