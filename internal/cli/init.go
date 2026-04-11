package cli

import (
	"os"

	"github.com/everlier/claune/internal/config"
)

func createDefaultConfig() error {
	configPath := config.ConfigFilePath()

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		PrintInfo("Configuration file already exists at: %s", configPath)
		return nil
	}

	// Create a new default config
	c := config.ClauneConfig{
		Sounds: make(map[string]config.EventSoundConfig),
		AI: config.AIConfig{
			Enabled: false,
			Model:   "claude-3-7-sonnet-latest",
		},
	}
	
	f := false
	c.Mute = &f
	v := 1.0
	c.Volume = &v

	if err := config.Save(c); err != nil {
		PrintError("Failed to create default configuration: %v", err)
		return err
	}

	PrintSuccess("Default configuration file created at: %s", configPath)
	return nil
}
