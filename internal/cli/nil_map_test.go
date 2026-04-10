package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigNilSounds(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	
	configPath := filepath.Join(tmpDir, ".claune.json")
	os.WriteFile(configPath, []byte(`{"ai":{"enabled":true,"api_key":"test"}, "volume": "invalid"}`), 0644)
	
	Run([]string{"config", "add a test sound to cli:start event"})
}
