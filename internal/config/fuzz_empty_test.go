package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, ".config", "claune", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	os.WriteFile(configPath, []byte(""), 0644)

	_, err := Load()
	t.Logf("Error: %v", err)
}
