package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ClauneConfig struct {
	Mute   *bool             `json:"mute,omitempty"`
	Volume *float64          `json:"volume,omitempty"`
	Sounds map[string]string `json:"sounds,omitempty"`
	AI     AIConfig          `json:"ai,omitempty"`
}

type AIConfig struct {
	Enabled   bool   `json:"enabled,omitempty"`
	Model     string `json:"model,omitempty"`
	APIKey    string `json:"api_key,omitempty"`
	APIURL    string `json:"api_url,omitempty"`
}

func Load() ClauneConfig {
	config := ClauneConfig{
		Sounds: make(map[string]string),
	}
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".claune.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &config)
	}
	if config.Sounds == nil {
		config.Sounds = make(map[string]string)
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
