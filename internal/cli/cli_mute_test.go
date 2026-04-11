package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMuteUnmuteVolume(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ".config", "claune", "config.json")

	// Create config dir
	os.MkdirAll(filepath.Dir(configPath), 0755)

	// Test mute
	output := captureOutput(t, func() {
		if err := Run([]string{"mute"}, "test-version"); err != nil {
			t.Fatalf("Run(mute) error = %v", err)
		}
	})
	assertContains(t, output.stdout, "Claune is now muted.")

	configBytes, _ := os.ReadFile(configPath)
	var config map[string]interface{}
	json.Unmarshal(configBytes, &config)
	if config["mute"] != true {
		t.Errorf("Expected mute to be true, got %v", config["mute"])
	}

	// Test unmute
	output = captureOutput(t, func() {
		if err := Run([]string{"unmute"}, "test-version"); err != nil {
			t.Fatalf("Run(unmute) error = %v", err)
		}
	})
	assertContains(t, output.stdout, "Claune is now unmuted.")

	configBytes, _ = os.ReadFile(configPath)
	json.Unmarshal(configBytes, &config)
	if config["mute"] != false {
		t.Errorf("Expected mute to be false, got %v", config["mute"])
	}

	// Test volume
	output = captureOutput(t, func() {
		if err := Run([]string{"volume", "75"}, "test-version"); err != nil {
			t.Fatalf("Run(volume) error = %v", err)
		}
	})
	assertContains(t, output.stdout, "Volume set to 75%.")

	configBytes, _ = os.ReadFile(configPath)
	json.Unmarshal(configBytes, &config)
	if config["volume"] != 0.75 {
		t.Errorf("Expected volume to be 0.75, got %v", config["volume"])
	}
}
