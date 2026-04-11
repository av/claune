package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/everlier/claune/internal/xdg"
)

type EventSoundConfig struct {
	Paths    []string `json:"paths"`
	Strategy string   `json:"strategy,omitempty"`
}

type ClauneConfig struct {
	Mute      *bool                       `json:"mute,omitempty"`
	MuteUntil *time.Time                  `json:"mute_until,omitempty"`
	Volume    *float64                    `json:"volume,omitempty"`
	Strategy  string                      `json:"strategy,omitempty"`
	Sounds    map[string]EventSoundConfig `json:"sounds,omitempty"`
	AI        AIConfig                    `json:"ai,omitempty"`
}

type AIConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Model   string `json:"model,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	APIURL  string `json:"api_url,omitempty"`
}

type InvalidConfigError struct {
	err error
}

func (e InvalidConfigError) Error() string {
	return e.err.Error()
}

func (e InvalidConfigError) Unwrap() error {
	return e.err
}

func IsInvalidConfigError(err error) bool {
	var invalidConfigError InvalidConfigError
	return errors.As(err, &invalidConfigError)
}

func configFilePath() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		legacyPath := filepath.Join(home, ".claune.json")
		if _, err := os.Stat(legacyPath); err == nil {
			return legacyPath
		}
	}
	return filepath.Join(xdg.ConfigHome(), "config.json")
}

func Load() (ClauneConfig, error) {
	config := ClauneConfig{
		Sounds: make(map[string]EventSoundConfig),
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		fmt.Fprintf(os.Stderr, "claune: warning: home directory not found or inaccessible, using default config\n")
		return config, nil
	}
	configPath := configFilePath()

	lockPath := configPath + ".lock"
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(lockPath); os.IsNotExist(err) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fine, file doesn't exist, use default config
		} else if os.IsPermission(err) {
			return config, fmt.Errorf("permission denied reading %s: %w", configPath, err)
		} else {
			return config, fmt.Errorf("failed to read %s: %w", configPath, err)
		}
	} else if err := json.Unmarshal(data, &config); err != nil {
		if strings.Contains(err.Error(), "cannot unmarshal string into Go struct field") && strings.Contains(err.Error(), "EventSoundConfig") {
			return config, InvalidConfigError{err: fmt.Errorf("invalid config format detected in %s. Sounds must now be configured as objects with 'paths' array, not strings. Please update your configuration schema: %w", configPath, err)}
		}
		return config, InvalidConfigError{err: fmt.Errorf("invalid configuration format in %s: %w", configPath, err)}
	}
	if config.Sounds == nil {
		config.Sounds = make(map[string]EventSoundConfig)
	}
	return config, nil
}

func Save(c ClauneConfig) error {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return fmt.Errorf("home directory not found or inaccessible")
	}
	configPath := configFilePath()

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	lockPath := configPath + ".lock"
	locked := false
	for i := 0; i < 50; i++ {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
		if err == nil {
			f.Close()
			locked = true
			break
		} else if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing config lock: %w", err)
		}
		if info, err := os.Stat(lockPath); err == nil {
			if time.Since(info.ModTime()) > 2*time.Second {
				os.Remove(lockPath)
				continue
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if locked {
		defer os.Remove(lockPath)
	} else {
		return fmt.Errorf("failed to acquire config lock %s after 500ms", lockPath)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	tmpFile, err := os.CreateTemp(dir, "config.json.tmp.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpFile.Name(), 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), configPath)
}

func (c ClauneConfig) ShouldMute() bool {
	if c.MuteUntil != nil && time.Now().Before(*c.MuteUntil) {
		return true
	}
	if c.Mute != nil {
		return *c.Mute
	}
	now := time.Now()
	hour := now.Hour()
	return hour >= 23 || hour < 7
}

func (c ClauneConfig) GetVolume() float64 {
	if c.Volume != nil {
		if *c.Volume < 0.0 {
			return 0.0
		}
		if *c.Volume > 1.0 {
			return 1.0
		}
		return *c.Volume
	}
	return 1.0
}
