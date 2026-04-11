package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIsClauneHook(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"claune play test", true},
		{"CLAUNE_ACTIVE=1 claune play", true},
		{"bash -c 'something else'", false},
		{"echo hello", false},
	}

	for _, tt := range tests {
		if got := isClauneHook(tt.cmd); got != tt.expected {
			t.Errorf("isClauneHook(%q) = %v, want %v", tt.cmd, got, tt.expected)
		}
	}
}

func TestRemoveClauneHooks(t *testing.T) {
	entries := []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{
				{Type: "command", Command: "echo hello"},
				{Type: "command", Command: "claune play sound"},
			},
		},
		{
			Matcher: "other",
			Hooks: []Hook{
				{Type: "command", Command: "claune play something"},
			},
		},
	}

	kept, changed := removeClauneHooks(entries)
	if !changed {
		t.Errorf("Expected changed = true")
	}
	if len(kept) != 1 {
		t.Fatalf("Expected 1 entry kept, got %d", len(kept))
	}
	if len(kept[0].Hooks) != 1 {
		t.Fatalf("Expected 1 hook kept in entry, got %d", len(kept[0].Hooks))
	}
	if kept[0].Hooks[0].Command != "echo hello" {
		t.Errorf("Expected kept hook to be 'echo hello', got %q", kept[0].Hooks[0].Command)
	}
}

func TestUninstallHooks(t *testing.T) {
	dir := t.TempDir()
	settingsPathFunc = func() string {
		return filepath.Join(dir, "settings.json")
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	// Create initial settings with hooks
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "claune play test",
						},
						map[string]interface{}{
							"type":    "command",
							"command": "echo hello",
						},
					},
				},
			},
		},
	}
	err := writeSettings(settings)
	if err != nil {
		t.Fatalf("Failed to write settings: %v", err)
	}

	if !hooksInstalled() {
		t.Errorf("Expected hooks to be installed")
	}

	err = uninstallHooks()
	if err != nil {
		t.Fatalf("Failed to uninstall hooks: %v", err)
	}

	if hooksInstalled() {
		t.Errorf("Expected hooks to be uninstalled")
	}

	// Read settings back
	data, err := os.ReadFile(settingsPathFunc())
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	var newSettings map[string]interface{}
	json.Unmarshal(data, &newSettings)

	hooksMap, ok := newSettings["hooks"].(map[string]interface{})
	if ok {
		if sessionStart, exists := hooksMap["SessionStart"]; exists {
			entries := sessionStart.([]interface{})
			if len(entries) > 0 {
				entry := entries[0].(map[string]interface{})
				hooks := entry["hooks"].([]interface{})
				if len(hooks) != 1 {
					t.Errorf("Expected 1 hook left, got %d", len(hooks))
				}
			}
		}
	}

	// Uninstall again should not fail
	err = uninstallHooks()
	if err != nil {
		t.Fatalf("Failed second uninstall: %v", err)
	}
}
