package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func Load() (ClauneConfig, error) {
	config := ClauneConfig{
		Sounds: make(map[string]EventSoundConfig),
	}
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claune.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return config, fmt.Errorf("failed to read ~/.claune.json: %w", err)
		}
	} else if err := json.Unmarshal(data, &config); err != nil {
		if strings.Contains(err.Error(), "cannot unmarshal string into Go struct field") {
			return config, InvalidConfigError{err: fmt.Errorf("invalid config format detected in ~/.claune.json. Sounds must now be configured as objects with 'paths' array, not strings. Please update your configuration schema: %w", err)}
		}
		return config, InvalidConfigError{err: fmt.Errorf("invalid configuration format in ~/.claune.json: %w", err)}
	}
	if config.Sounds == nil {
		config.Sounds = make(map[string]EventSoundConfig)
	}
	return config, nil
}

func Save(c ClauneConfig) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claune.json")
	
	lockPath := configPath + ".lock"
	locked := false
	for i := 0; i < 50; i++ {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
		if err == nil {
			f.Close()
			locked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if locked {
		defer os.Remove(lockPath)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(configPath)
	tmpFile, err := os.CreateTemp(dir, ".claune.json.tmp.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
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
