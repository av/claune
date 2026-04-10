package xdg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXDGPaths(t *testing.T) {
	// Save current env
	originalConfig := os.Getenv("XDG_CONFIG_HOME")
	originalData := os.Getenv("XDG_DATA_HOME")
	originalState := os.Getenv("XDG_STATE_HOME")
	originalCache := os.Getenv("XDG_CACHE_HOME")
	originalHome := os.Getenv("HOME")

	defer func() {
		os.Setenv("XDG_CONFIG_HOME", originalConfig)
		os.Setenv("XDG_DATA_HOME", originalData)
		os.Setenv("XDG_STATE_HOME", originalState)
		os.Setenv("XDG_CACHE_HOME", originalCache)
		os.Setenv("HOME", originalHome)
	}()

	// Test with XDG env vars set
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	os.Setenv("XDG_DATA_HOME", "/custom/data")
	os.Setenv("XDG_STATE_HOME", "/custom/state")
	os.Setenv("XDG_CACHE_HOME", "/custom/cache")

	if p := ConfigHome(); p != filepath.Join("/custom/config", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/custom/config", "claune"), p)
	}
	if p := DataHome(); p != filepath.Join("/custom/data", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/custom/data", "claune"), p)
	}
	if p := StateHome(); p != filepath.Join("/custom/state", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/custom/state", "claune"), p)
	}
	if p := CacheHome(); p != filepath.Join("/custom/cache", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/custom/cache", "claune"), p)
	}

	// Test with XDG env vars unset, using HOME
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_STATE_HOME")
	os.Unsetenv("XDG_CACHE_HOME")
	os.Setenv("HOME", "/fake/home")

	if p := ConfigHome(); p != filepath.Join("/fake/home", ".config", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/fake/home", ".config", "claune"), p)
	}
	if p := DataHome(); p != filepath.Join("/fake/home", ".local", "share", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/fake/home", ".local", "share", "claune"), p)
	}
	if p := StateHome(); p != filepath.Join("/fake/home", ".local", "state", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/fake/home", ".local", "state", "claune"), p)
	}
	if p := CacheHome(); p != filepath.Join("/fake/home", ".cache", "claune") {
		t.Errorf("expected %s, got %s", filepath.Join("/fake/home", ".cache", "claune"), p)
	}

	// Test with no XDG and no HOME
	os.Unsetenv("HOME")
	// On Windows, USERPROFILE might be used. For testing simplicity, just clear standard vars.
	// os.UserHomeDir might still resolve depending on the OS if other vars are set, 
	// but let's check if the fallback works when HOME is missing and userhomedir returns empty/error
	// Actually overriding userhomedir isn't trivial in Go without mock, so we just accept it might fall back to temp or use OS defaults.
	// If os.UserHomeDir fails, it falls back to TempDir. We can simulate it by ensuring HOME is unset, but let's just do a basic test.
	configFallback := ConfigHome()
	if !strings.Contains(configFallback, "claune") {
		t.Errorf("ConfigHome fallback missing claune: %s", configFallback)
	}
}
