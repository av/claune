package config

import (
	"encoding/json"
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

func Load() (ClauneConfig, error) {
	config := ClauneConfig{
		Sounds: make(map[string]EventSoundConfig),
	}
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claune.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			if strings.Contains(err.Error(), "cannot unmarshal string into Go struct field") {
				return config, fmt.Errorf("invalid config format detected in ~/.claune.json. Sounds must now be configured as objects with 'paths' array, not strings. Please update your configuration schema: %w", err)
			}
			return config, fmt.Errorf("invalid configuration format in ~/.claune.json: %w", err)
		}
	}
	if config.Sounds == nil {
		config.Sounds = make(map[string]EventSoundConfig)
	}
	return config, nil
}

func Save(c ClauneConfig) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claune.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
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
		return *c.Volume
	}
	return 1.0
}
