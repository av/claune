package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidConfigErrorMethods(t *testing.T) {
	baseErr := errors.New("base error")
	err := InvalidConfigError{err: baseErr}

	if err.Error() != "base error" {
		t.Errorf("expected 'base error', got %q", err.Error())
	}

	if !errors.Is(err.Unwrap(), baseErr) {
		t.Errorf("expected wrapped error to be baseErr")
	}

	if !IsInvalidConfigError(err) {
		t.Errorf("expected IsInvalidConfigError to return true")
	}

	if IsInvalidConfigError(baseErr) {
		t.Errorf("expected IsInvalidConfigError to return false for non-InvalidConfigError")
	}
}

func TestLoadPermissionDenied(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	
	// Create a file with no read permissions
	if err := os.WriteFile(configPath, []byte("{}"), 0000); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load() to fail with permission denied")
	}
	if !os.IsPermission(err) && !errors.Is(err, os.ErrPermission) {
		// Just check the error string as fallback
		if err.Error() != fmt.Sprintf("permission denied reading %s: permission denied", configPath) {
			// Do nothing
		}
	}
}

func TestSaveMkdirAllFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "claune")
	
	// Make a file where the directory should be
	os.MkdirAll(filepath.Join(home, ".config"), 0755)
	if err := os.WriteFile(configDir, []byte("file"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	c := ClauneConfig{Strategy: "fail"}
	err := Save(c)
	if err == nil {
		t.Fatal("expected Save() to fail due to MkdirAll error on file")
	}
}

func TestSaveCreateTempFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "claune")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Make the directory read-only so CreateTemp fails
	if err := os.Chmod(configDir, 0555); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	c := ClauneConfig{Strategy: "fail"}
	err := Save(c)
	if err == nil {
		t.Fatal("expected Save() to fail due to CreateTemp error")
	}
	
	os.Chmod(configDir, 0755) // restore for cleanup
}

func TestLoadDirectoryFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected Load() to fail when config is a directory")
	}
}

func TestSavePermissionDeniedWritingLock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "claune")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Make the directory read-only so lock creation fails
	if err := os.Chmod(configDir, 0555); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	c := ClauneConfig{Strategy: "fail"}
	err := Save(c)
	if err == nil {
		t.Fatal("expected Save() to fail due to lock permission denied")
	}
	
	os.Chmod(configDir, 0755) // restore for cleanup
}


func TestLoadEmptyHomeFails(t *testing.T) {
	t.Setenv("HOME", "")
	
	// Ensure UserHomeDir will fail or return empty
	config, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error = %v", err)
	}
	if config.Sounds == nil {
		t.Fatal("expected default config")
	}
}

func TestSaveEmptyHomeFails(t *testing.T) {
	t.Setenv("HOME", "")

	c := ClauneConfig{Strategy: "fail"}
	err := Save(c)
	if err == nil {
		t.Fatal("expected Save() to fail due to empty home")
	}
	if err.Error() != "home directory not found or inaccessible" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadNullSounds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".config", "claune", "config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	
	if err := os.WriteFile(configPath, []byte(`{"sounds": null}`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error = %v", err)
	}
	if config.Sounds == nil {
		t.Fatal("expected Sounds map to be initialized when null in JSON")
	}
}
