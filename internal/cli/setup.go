package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/everlier/claune/internal/config"
)

func runSetup() error {
	fmt.Println(Style("Welcome to the Claune interactive setup!", ColorCyan+ColorBold))
	fmt.Println("Let's configure your sound experience.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	c, _ := config.Load()
	if c.Sounds == nil {
		c.Sounds = make(map[string]config.EventSoundConfig)
	}
	if c.AI.Model == "" {
		c.AI.Model = "claude-3-7-sonnet-latest"
	}

	// 1. Mute status
	fmt.Print("Do you want to start with sounds enabled? [Y/n]: ")
	muteResp, _ := reader.ReadString('\n')
	muteResp = strings.TrimSpace(strings.ToLower(muteResp))

	f := false
	t := true
	if muteResp == "n" || muteResp == "no" {
		c.Mute = &t
		PrintInfo("Sounds will be muted initially.")
	} else {
		c.Mute = &f
		PrintSuccess("Sounds enabled!")
	}
	fmt.Println()

	// 2. AI Key
	fmt.Print("Do you have an Anthropic API key to enable AI-driven sound mappings? [y/N]: ")
	aiResp, _ := reader.ReadString('\n')
	aiResp = strings.TrimSpace(strings.ToLower(aiResp))

	if aiResp == "y" || aiResp == "yes" {
		fmt.Print("Please enter your Anthropic API key (sk-ant-...): ")
		keyResp, _ := reader.ReadString('\n')
		keyResp = strings.TrimSpace(keyResp)
		if keyResp != "" {
			c.AI.Enabled = true
			c.AI.APIKey = keyResp
			PrintSuccess("AI features enabled!")
		} else {
			PrintWarning("No key provided. AI features will remain disabled.")
		}
	} else {
		PrintInfo("Skipping AI setup. You can always run 'claune auth <key>' later.")
	}
	fmt.Println()

	// 3. Set volume
	fmt.Print("What volume level would you like? (0.0 to 1.0, default 0.7): ")
	volResp, _ := reader.ReadString('\n')
	volResp = strings.TrimSpace(volResp)
	if volResp != "" {
		var vol float64
		_, err := fmt.Sscanf(volResp, "%f", &vol)
		if err == nil && vol >= 0.0 && vol <= 1.0 {
			c.Volume = &vol
			PrintSuccess("Volume set to %.1f.", vol)
		} else {
			PrintWarning("Invalid volume, keeping default.")
		}
	}

	fmt.Println()
	if err := config.Save(c); err != nil {
		PrintError("Failed to save configuration: %v", err)
		return err
	}

	PrintSuccess("Configuration saved to %s", config.ConfigFilePath())
	fmt.Println(Style("You're all set! Try running: claune test-sounds", ColorCyan))
	return nil
}
