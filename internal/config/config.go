package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ClauneConfig struct {
	Mute            *bool               `json:"mute,omitempty"`
	MuteUntil       *time.Time          `json:"mute_until,omitempty"`
	Volume          *float64            `json:"volume,omitempty"`
	Sounds          map[string][]string `json:"sounds,omitempty"`
	SoundStrategies map[string]string   `json:"sound_strategies,omitempty"` // "random" or "round_robin" per event
	SoundStrategy   string              `json:"sound_strategy,omitempty"`   // global fallback
	AI              AIConfig            `json:"ai,omitempty"`
}

type AIConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Model   string `json:"model,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	APIURL  string `json:"api_url,omitempty"`
}

func Load() ClauneConfig {
	config := ClauneConfig{
		Sounds:          make(map[string][]string),
		SoundStrategies: make(map[string]string),
	}
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claune.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		// Read raw map first to handle backward compat of string vs []string
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err == nil {
			// Basic parsing
			json.Unmarshal(data, &config) // this gets everything except properly handling old 'sounds'
			
			if soundsRaw, ok := raw["sounds"].(map[string]interface{}); ok {
				config.Sounds = make(map[string][]string)
				for k, v := range soundsRaw {
					if str, ok := v.(string); ok {
						config.Sounds[k] = []string{str}
					} else if arr, ok := v.([]interface{}); ok {
						for _, item := range arr {
							if str, ok := item.(string); ok {
								config.Sounds[k] = append(config.Sounds[k], str)
							}
						}
					}
				}
			}
		}
	}
	if config.Sounds == nil {
		config.Sounds = make(map[string][]string)
	}
	if config.SoundStrategies == nil {
		config.SoundStrategies = make(map[string]string)
	}
	if config.SoundStrategy == "" {
		config.SoundStrategy = "random"
	}
	return config
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
