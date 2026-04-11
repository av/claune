package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configPath := filepath.Join(home, ".config", "claune", "config.json")

	output := captureOutput(t, func() {
		if err := Run([]string{"init"}, "test-version"); err != nil {
			t.Fatalf("Run(init) error = %v", err)
		}
	})

	assertContains(t, output.stdout, "Default configuration file created at:")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Expected config file to be created at %s", configPath)
	}

	// Test init when config already exists
	output2 := captureOutput(t, func() {
		err := Run([]string{"init"}, "test-version")
		if err != nil {
			t.Fatalf("Run(init) error = %v", err)
		}
	})
	assertContains(t, output2.stdout, "Configuration file already exists at:")
}
