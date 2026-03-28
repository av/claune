package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldMute(t *testing.T) {
	trueVal := true
	falseVal := false

	c1 := ClauneConfig{Mute: &trueVal}
	if !shouldMute(c1) {
		t.Error("Expected mute when mute=true")
	}

	c2 := ClauneConfig{Mute: &falseVal}
	if shouldMute(c2) {
		t.Error("Expected not muted when mute=false")
	}

	// When mute is nil, smart mute kicks in based on time — just verify no panic
	c3 := ClauneConfig{Mute: nil}
	_ = shouldMute(c3)
}

func TestGetConfig(t *testing.T) {
	config := getConfig()
	if config.Sounds == nil {
		t.Error("Sounds map should not be nil")
	}
}

func TestGetVolume(t *testing.T) {
	c1 := ClauneConfig{}
	if v := getVolume(c1); v != 1.0 {
		t.Errorf("Expected default volume 1.0, got %f", v)
	}

	vol := 0.5
	c2 := ClauneConfig{Volume: &vol}
	if v := getVolume(c2); v != 0.5 {
		t.Errorf("Expected volume 0.5, got %f", v)
	}
}

func TestMergeHooksDedup(t *testing.T) {
	existing := []HookEntry{{
		Matcher: ".*",
		Hooks:   []Hook{{Type: "command", Command: "/usr/bin/claune play tool:start", Timeout: 5}},
	}}
	newHooks := []HookEntry{{
		Matcher: ".*",
		Hooks:   []Hook{{Type: "command", Command: "/usr/bin/claune play tool:start", Timeout: 5}},
	}}
	merged := mergeHooks(existing, newHooks)
	if len(merged) != 1 {
		t.Errorf("Expected 1 hook after dedup, got %d", len(merged))
	}
}

func TestMergeHooksAddsNew(t *testing.T) {
	existing := []HookEntry{{
		Matcher: ".*",
		Hooks:   []Hook{{Type: "command", Command: "/usr/bin/claune play tool:start", Timeout: 5}},
	}}
	newHooks := []HookEntry{{
		Matcher: ".*",
		Hooks:   []Hook{{Type: "command", Command: "/usr/bin/claune play tool:success", Timeout: 5}},
	}}
	merged := mergeHooks(existing, newHooks)
	if len(merged) != 2 {
		t.Errorf("Expected 2 hooks, got %d", len(merged))
	}
}

func TestMergeHooksEmpty(t *testing.T) {
	newHooks := []HookEntry{{
		Matcher: ".*",
		Hooks:   []Hook{{Type: "command", Command: "/usr/bin/claune play tool:start", Timeout: 5}},
	}}
	merged := mergeHooks(nil, newHooks)
	if len(merged) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(merged))
	}
}

func TestDefaultSoundMap(t *testing.T) {
	for _, key := range []string{"cli:start", "tool:start", "tool:success", "tool:error", "cli:done"} {
		if _, ok := defaultSoundMap[key]; !ok {
			t.Errorf("Missing default sound mapping for %s", key)
		}
	}
}

func TestEmbeddedSounds(t *testing.T) {
	for _, file := range defaultSoundMap {
		data, err := soundFS.ReadFile("sounds/" + file)
		if err != nil {
			t.Errorf("Embedded sound missing: sounds/%s: %v", file, err)
		}
		if len(data) == 0 {
			t.Errorf("Embedded sound empty: sounds/%s", file)
		}
	}
}

func TestContainsClaunePlay(t *testing.T) {
	if !containsClaunePlay("/usr/bin/claune play tool:start") {
		t.Error("Expected true for command containing 'claune play'")
	}
	if containsClaunePlay("echo hello") {
		t.Error("Expected false for unrelated command")
	}
	if containsClaunePlay("") {
		t.Error("Expected false for empty string")
	}
}

func TestFindAudioPlayer(t *testing.T) {
	// Just verify no panic
	player, _ := findAudioPlayer()
	_ = player
}

func TestInstallUninstallRoundtrip(t *testing.T) {
	// Use a temp dir to avoid touching real settings
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	// Patch settingsPath and resolveClauneBin for this test
	origFunc := settingsPathFunc
	settingsPathFunc = func() string { return settingsFile }
	defer func() { settingsPathFunc = origFunc }()

	origBin := resolveClauneBinFunc
	resolveClauneBinFunc = func() string { return "/usr/local/bin/claune" }
	defer func() { resolveClauneBinFunc = origBin }()

	// Install
	if err := installHooks(); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify hooks are in file
	data, _ := os.ReadFile(settingsFile)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)
	hooks := settings["hooks"].(map[string]interface{})
	for _, key := range []string{"PreToolUse", "PostToolUse", "PostToolUseFailure"} {
		if _, ok := hooks[key]; !ok {
			t.Errorf("Missing hook after install: %s", key)
		}
	}

	// Verify hooksInstalled returns true
	if !hooksInstalled() {
		t.Error("Expected hooksInstalled() == true after install")
	}

	// Uninstall
	if err := uninstallHooks(); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	// Verify hooks removed
	if hooksInstalled() {
		t.Error("Expected hooksInstalled() == false after uninstall")
	}
}

func TestRemoveClauneHooks(t *testing.T) {
	entries := []HookEntry{
		{Matcher: ".*", Hooks: []Hook{{Type: "command", Command: "claune play tool:start", Timeout: 5}}},
		{Matcher: ".*", Hooks: []Hook{{Type: "command", Command: "some-other-hook", Timeout: 5}}},
	}
	kept := removeClauneHooks(entries)
	if len(kept) != 1 {
		t.Errorf("Expected 1 non-claune hook, got %d", len(kept))
	}
	if kept[0].Hooks[0].Command != "some-other-hook" {
		t.Error("Wrong hook kept")
	}
}
