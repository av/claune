package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/everlier/claune/internal/config"
)

func runDoctor(version string) error {
	fmt.Printf("%sClaune Doctor (v%s)%s\n", ColorCyan+ColorBold, version, ColorReset)
	fmt.Println("--------------------------------")

	fmt.Printf("%sOS:%s %s\n", ColorWhite, ColorReset, runtime.GOOS)
	fmt.Printf("%sArchitecture:%s %s\n", ColorWhite, ColorReset, runtime.GOARCH)

	// Check config
	cfgPath := config.ConfigFilePath()
	fmt.Printf("%sConfig File:%s %s\n", ColorWhite, ColorReset, cfgPath)
	c, err := loadCommandConfig("doctor")
	if err != nil {
		fmt.Printf("  ❌ %sError loading config: %v%s\n", ColorRed, err, ColorReset)
	} else {
		fmt.Printf("  ✅ %sConfig loaded successfully%s\n", ColorGreen, ColorReset)
		if c.AI.Enabled {
			fmt.Printf("  ✅ %sAI Features:%s Enabled\n", ColorGreen, ColorReset)
			if c.AI.APIKey != "" {
				fmt.Printf("  ✅ %sAnthropic API Key:%s Set\n", ColorGreen, ColorReset)
			} else {
				fmt.Printf("  ❌ %sAnthropic API Key:%s Missing (run 'claune auth <key>')\n", ColorRed, ColorReset)
			}
		} else {
			fmt.Printf("  ⚠️  %sAI Features:%s Disabled (run 'claune auth <key>')\n", ColorYellow, ColorReset)
		}
	}

	fmt.Printf("\n%sAudio Dependencies:%s\n", ColorWhite+ColorBold, ColorReset)
	if runtime.GOOS == "linux" {
		backends := []string{"paplay", "pw-play", "aplay"}
		foundAny := false
		for _, bin := range backends {
			if path, err := exec.LookPath(bin); err == nil {
				fmt.Printf("  ✅ %s%s:%s Found at %s\n", ColorGreen, bin, ColorReset, path)
				foundAny = true
			} else {
				fmt.Printf("  ❌ %s%s:%s Not found\n", ColorRed, bin, ColorReset)
			}
		}
		if !foundAny {
			fmt.Printf("  ❌ %sNO AUDIO BACKEND FOUND! You must install pulseaudio-utils, pipewire, or alsa-utils.%s\n", ColorRed, ColorReset)
		} else {
			fmt.Printf("  ✅ %sAt least one valid audio backend is available.%s\n", ColorGreen, ColorReset)
		}
	} else {
		fmt.Printf("  ✅ %sUsing native audio drivers (oto via beep) for %s%s\n", ColorGreen, runtime.GOOS, ColorReset)
	}

	fmt.Printf("\n%sRun 'claune test-sounds' to verify audio playback.%s\n", ColorDim, ColorReset)
	return nil
}
