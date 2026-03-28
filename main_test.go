package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestBuildHookConfig(t *testing.T) {
	tmpPath, err := buildHookConfig()
	if err != nil {
		t.Fatalf("buildHookConfig failed: %v", err)
	}
	defer os.Remove(tmpPath)

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read temp config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config JSON: %v", err)
	}

	hooks, ok := config["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hooks in config")
	}

	for _, key := range []string{"PreToolUse", "PostToolUse", "PostToolUseFailure"} {
		if _, ok := hooks[key]; !ok {
			t.Errorf("Missing hook: %s", key)
		}
	}
}

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

	// When mute is nil, smart mute kicks in based on time
	c3 := ClauneConfig{Mute: nil}
	// We can't assert the exact value since it depends on current time,
	// but we verify it doesn't panic
	_ = shouldMute(c3)
}

func TestGetConfig(t *testing.T) {
	config := getConfig()
	if config.Sounds == nil {
		t.Error("Sounds map should not be nil")
	}
}

func TestGetVolume(t *testing.T) {
	// Default volume is 1.0
	c1 := ClauneConfig{}
	if v := getVolume(c1); v != 1.0 {
		t.Errorf("Expected default volume 1.0, got %f", v)
	}

	// Custom volume
	vol := 0.5
	c2 := ClauneConfig{Volume: &vol}
	if v := getVolume(c2); v != 0.5 {
		t.Errorf("Expected volume 0.5, got %f", v)
	}
}

func TestMergeHooksDedup(t *testing.T) {
	existing := []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: "/usr/bin/claune play tool:start",
				Timeout: 5,
			}},
		},
	}

	newHooks := []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: "/usr/bin/claune play tool:start",
				Timeout: 5,
			}},
		},
	}

	merged := mergeHooks(existing, newHooks)
	if len(merged) != 1 {
		t.Errorf("Expected 1 hook after dedup, got %d", len(merged))
	}
}

func TestMergeHooksAddsNew(t *testing.T) {
	existing := []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: "/usr/bin/claune play tool:start",
				Timeout: 5,
			}},
		},
	}

	newHooks := []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: "/usr/bin/claune play tool:success",
				Timeout: 5,
			}},
		},
	}

	merged := mergeHooks(existing, newHooks)
	if len(merged) != 2 {
		t.Errorf("Expected 2 hooks after merge, got %d", len(merged))
	}
}

func TestMergeHooksEmpty(t *testing.T) {
	newHooks := []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{{
				Type:    "command",
				Command: "/usr/bin/claune play tool:start",
				Timeout: 5,
			}},
		},
	}

	merged := mergeHooks(nil, newHooks)
	if len(merged) != 1 {
		t.Errorf("Expected 1 hook from empty merge, got %d", len(merged))
	}
}

func TestDefaultSoundMap(t *testing.T) {
	expected := []string{"cli:start", "tool:start", "tool:success", "tool:error", "cli:done"}
	for _, key := range expected {
		if _, ok := defaultSoundMap[key]; !ok {
			t.Errorf("Missing default sound mapping for %s", key)
		}
	}
}

func TestEmbeddedSounds(t *testing.T) {
	for _, file := range defaultSoundMap {
		data, err := soundFS.ReadFile("sounds/" + file)
		if err != nil {
			t.Errorf("Embedded sound file missing: sounds/%s: %v", file, err)
		}
		if len(data) == 0 {
			t.Errorf("Embedded sound file empty: sounds/%s", file)
		}
	}
}

func TestFindAudioPlayer(t *testing.T) {
	player, _ := findAudioPlayer()
	// On any Linux/macOS system at least one player should be found
	// but we don't fail the test if none - just verify no panic
	_ = player
}

func TestContainsClaunePlay(t *testing.T) {
	if !containsClaunePlay("/usr/bin/claune play tool:start") {
		t.Error("Expected true for command containing 'claune play'")
	}
	if containsClaunePlay("echo hello") {
		t.Error("Expected false for command not containing 'claune play'")
	}
	if containsClaunePlay("") {
		t.Error("Expected false for empty string")
	}
}
