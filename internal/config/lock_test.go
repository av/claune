package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveWithFreshLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
os.MkdirAll(filepath.Dir(configPath), 0755)
	lockPath := configPath + ".lock"

	os.WriteFile(lockPath, []byte(""), 0666)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				now := time.Now()
				os.Chtimes(lockPath, now, now)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	c := ClauneConfig{Strategy: "test"}
	err := Save(c)
	close(done)
	if err == nil {
		t.Fatalf("Save succeeded despite fresh lock! Expected an error.")
	}
}
