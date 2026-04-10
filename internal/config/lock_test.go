package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveWithFreshLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	lockPath := configPath + ".lock"

	os.WriteFile(lockPath, []byte(""), 0666)

	c := ClauneConfig{Strategy: "test"}
	err := Save(c)
	if err == nil {
		t.Fatalf("Save succeeded despite fresh lock! Expected an error.")
	}
}
