package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigNilSounds(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configPath := filepath.Join(tmpDir, ".config", "claune", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	os.WriteFile(configPath, []byte(`{"ai":{"enabled":true,"api_key":"test"}, "volume": "invalid"}`), 0644)

	Run([]string{"config", "add a test sound to cli:start event"}, "test-version")
}
