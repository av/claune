package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadMissingConfigReturnsDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Sounds == nil {
		t.Fatal("Load() returned nil Sounds map")
	}
	if len(config.Sounds) != 0 {
		t.Fatalf("Load() Sounds length = %d, want 0", len(config.Sounds))
	}
	if got := config.GetVolume(); got != 1.0 {
		t.Fatalf("GetVolume() = %v, want 1.0", got)
	}
}

func TestLoadValidConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	contents := `{
		"mute": true,
		"volume": 0.35,
		"strategy": "round_robin",
		"sounds": {
			"success": {
				"paths": ["test1.mp3", "test2.mp3"],
				"strategy": "random"
			}
		},
		"ai": {
			"enabled": true,
			"model": "gpt-test",
			"api_key": "secret",
			"api_url": "https://example.invalid"
		}
	}`
	if err := os.WriteFile(configPath, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Mute == nil || !*config.Mute {
		t.Fatalf("Load() Mute = %v, want true", config.Mute)
	}
	if got := config.GetVolume(); got != 0.35 {
		t.Fatalf("GetVolume() = %v, want 0.35", got)
	}
	if !config.ShouldMute() {
		t.Fatal("ShouldMute() = false, want true")
	}
	if config.Strategy != "round_robin" {
		t.Fatalf("Strategy = %q, want round_robin", config.Strategy)
	}
	success, ok := config.Sounds["success"]
	if !ok {
		t.Fatal("Load() missing success sound config")
	}
	if len(success.Paths) != 2 || success.Paths[0] != "test1.mp3" || success.Paths[1] != "test2.mp3" {
		t.Fatalf("success.Paths = %#v, want [test1.mp3 test2.mp3]", success.Paths)
	}
	if success.Strategy != "random" {
		t.Fatalf("success.Strategy = %q, want random", success.Strategy)
	}
	if !config.AI.Enabled || config.AI.Model != "gpt-test" || config.AI.APIKey != "secret" || config.AI.APIURL != "https://example.invalid" {
		t.Fatalf("AI config parsed incorrectly: %+v", config.AI)
	}
}

func TestLoadMalformedConfigFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := os.WriteFile(configPath, []byte(`{"sounds":`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want malformed config failure")
	}
	if !strings.Contains(err.Error(), "invalid configuration format in " + configPath + "") {
		t.Fatalf("Load() error = %q, want invalid configuration format context", err)
	}
}

func TestLoadLegacySchemaRejected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := os.WriteFile(configPath, []byte(`{
		"sounds": {
			"success": "test1.mp3"
		}
	}`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want legacy schema rejection")
	}
	if !strings.Contains(err.Error(), "Sounds must now be configured as objects with 'paths' array, not strings") {
		t.Fatalf("Load() error = %q, want legacy schema guidance", err)
	}
}

func TestLoadUnreadableConfigPathFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := os.Mkdir(configPath, 0755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", configPath, err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want unreadable config path failure")
	}
	if !strings.Contains(err.Error(), "failed to read " + configPath + "") {
		t.Fatalf("Load() error = %q, want read failure context", err)
	}
}

func TestShouldMuteUsesMuteUntilBeforeMuteFlag(t *testing.T) {
	falseValue := false
	future := time.Now().Add(24 * time.Hour)

	config := ClauneConfig{
		Mute:      &falseValue,
		MuteUntil: &future,
	}

	if !config.ShouldMute() {
		t.Fatal("ShouldMute() = false, want true while mute_until is in the future")
	}
}

func TestShouldMuteUsesExplicitMuteWhenMuteUntilExpired(t *testing.T) {
	trueValue := true
	past := time.Now().Add(-24 * time.Hour)

	config := ClauneConfig{
		Mute:      &trueValue,
		MuteUntil: &past,
	}

	if !config.ShouldMute() {
		t.Fatal("ShouldMute() = false, want true from explicit mute flag after mute_until expires")
	}
}

func TestGetVolumeDefaultAndExplicit(t *testing.T) {
	config := ClauneConfig{}
	if got := config.GetVolume(); got != 1.0 {
		t.Fatalf("GetVolume() = %v, want 1.0", got)
	}

	volume := 0.75
	config.Volume = &volume
	if got := config.GetVolume(); got != 0.75 {
		t.Fatalf("GetVolume() = %v, want 0.75", got)
	}
}

func TestGetVolumeClampsOutOfRangeValues(t *testing.T) {
	t.Run("below zero", func(t *testing.T) {
		volume := -0.25
		config := ClauneConfig{Volume: &volume}

		if got := config.GetVolume(); got != 0.0 {
			t.Fatalf("GetVolume() = %v, want 0.0", got)
		}
	})

	t.Run("above one", func(t *testing.T) {
		volume := 1.25
		config := ClauneConfig{Volume: &volume}

		if got := config.GetVolume(); got != 1.0 {
			t.Fatalf("GetVolume() = %v, want 1.0", got)
		}
	})
}

func TestSaveConcurrentModifications(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	c := ClauneConfig{Strategy: "concurrent"}
	
	errCh := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			errCh <- Save(c)
		}()
	}
	
	for i := 0; i < 10; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("Concurrent Save failed: %v", err)
		}
	}
}

func TestSaveAutoRecoverStaleLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	lockPath := configPath + ".lock"

	// Create a "stale" lock file (older than 2 seconds)
	if err := os.WriteFile(lockPath, []byte(""), 0666); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-3 * time.Second)
	if err := os.Chtimes(lockPath, past, past); err != nil {
		t.Fatal(err)
	}

	c := ClauneConfig{Strategy: "recovered"}
	if err := Save(c); err != nil {
		t.Fatalf("Save failed with stale lock: %v", err)
	}

	// Lock file should be removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("Lock file was not removed: %v", err)
	}
}
