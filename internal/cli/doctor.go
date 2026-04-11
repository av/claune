package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/everlier/claune/internal/config"
)

func runDoctor(version string) error {
	fmt.Printf("Claune Doctor (v%s)\n", version)
	fmt.Println("--------------------------------")

	fmt.Printf("OS: %s\n", runtime.GOOS)
	fmt.Printf("Architecture: %s\n", runtime.GOARCH)

	// Check config
	cfgPath := config.ConfigFilePath()
	fmt.Printf("Config File: %s\n", cfgPath)
	c, err := loadCommandConfig("doctor")
	if err != nil {
		fmt.Printf("  ❌ Error loading config: %v\n", err)
	} else {
		fmt.Printf("  ✅ Config loaded successfully\n")
		if c.AI.Enabled {
			fmt.Printf("  ✅ AI Features: Enabled\n")
			if c.AI.APIKey != "" {
				fmt.Printf("  ✅ Anthropic API Key: Set\n")
			} else {
				fmt.Printf("  ❌ Anthropic API Key: Missing (run 'claune auth <key>')\n")
			}
		} else {
			fmt.Printf("  ⚠️ AI Features: Disabled (run 'claune auth <key>')\n")
		}
	}

	fmt.Println("\nAudio Dependencies:")
	if runtime.GOOS == "linux" {
		backends := []string{"paplay", "pw-play", "aplay"}
		foundAny := false
		for _, bin := range backends {
			if path, err := exec.LookPath(bin); err == nil {
				fmt.Printf("  ✅ %s: Found at %s\n", bin, path)
				foundAny = true
			} else {
				fmt.Printf("  ❌ %s: Not found\n", bin)
			}
		}
		if !foundAny {
			fmt.Println("  ❌ NO AUDIO BACKEND FOUND! You must install pulseaudio-utils, pipewire, or alsa-utils.")
		} else {
			fmt.Println("  ✅ At least one valid audio backend is available.")
		}
	} else {
		fmt.Printf("  ✅ Using native audio drivers (oto via beep) for %s\n", runtime.GOOS)
	}

	fmt.Println("\nRun 'claune test-sounds' to verify audio playback.")
	return nil
}
