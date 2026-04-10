package config

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzConfigAggressive(f *testing.F) {
	f.Add([]byte(`{"mute": true}`))
	f.Add([]byte(`{"volume": 0.5}`))
	f.Add([]byte(`{"ai": {"enabled": true}}`))
	
	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := os.TempDir()
		t.Setenv("HOME", tmpDir)
		
		configPath := filepath.Join(tmpDir, ".claune.json")
		err := os.WriteFile(configPath, data, 0644)
		if err != nil {
			t.Skip()
		}
		
		config, err := Load()
		if err == nil {
			config.ShouldMute()
			config.GetVolume()
			Save(config)
		} else {
            err.Error()
            IsInvalidConfigError(err)
        }
	})
}
