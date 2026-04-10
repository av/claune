package xdg

import (
	"os"
	"path/filepath"
)

func ConfigHome() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "claune")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".config", "claune")
	}
	return filepath.Join(os.TempDir(), "claune_config")
}

func DataHome() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "claune")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".local", "share", "claune")
	}
	return filepath.Join(os.TempDir(), "claune_data")
}

func StateHome() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "claune")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".local", "state", "claune")
	}
	return filepath.Join(os.TempDir(), "claune_state")
}

func CacheHome() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "claune")
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".cache", "claune")
	}
	return filepath.Join(os.TempDir(), "claune_cache")
}
