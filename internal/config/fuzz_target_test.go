package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func FuzzParseConfig(f *testing.F) {
	f.Add([]byte(`{"mute": true}`))
	f.Add([]byte(`{"volume": 0.5}`))
	f.Add([]byte(`{"sounds": {"cli:start": {"paths": ["a"]}}}`))
	
	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := os.TempDir()
		t.Setenv("HOME", tmpDir)
		
		configPath := filepath.Join(tmpDir, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
		os.WriteFile(configPath, data, 0644)
		
		c, err := Load()
		if err == nil {
			c.ShouldMute()
			c.GetVolume()
			Save(c)
		} else {
            err.Error()
        }
        
        var c2 ClauneConfig
        json.Unmarshal(data, &c2)
	})
}
