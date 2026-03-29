package cli

import (
	"fmt"
	"os"

	"github.com/everlier/claune/internal/ai"
	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/config"
)

var clauneSubcommands = map[string]bool{
	"play":        true,
	"install":     true,
	"uninstall":   true,
	"status":      true,
	"test-sounds": true,
	"help":        true,
	"config":      true,
}

func Run(args []string) error {
	if len(args) == 0 || !clauneSubcommands[args[0]] {
		runPassthrough(args)
		return nil
	}

	c := config.Load()

	switch args[0] {
	case "play":
		if len(args) > 1 {
			if len(args) > 3 {
				event := ai.AnalyzeToolIntent(args[2], args[3], c)
				audio.PlaySound(event, false, c)
			} else {
				audio.PlaySound(args[1], false, c)
			}
		}
	case "install":
		if err := installHooks(); err != nil {
			return err
		}
	case "uninstall":
		if err := uninstallHooks(); err != nil {
			return err
		}
	case "status":
		showStatus(c)
	case "test-sounds":
		testSounds(c)
	case "config":
		if len(args) > 1 {
			prompt := args[1]
			if err := ai.HandleNaturalLanguageConfig(prompt, &c); err != nil {
				return fmt.Errorf("AI config failed: %w", err)
			}
			fmt.Println("Config updated successfully via AI")
		} else {
			fmt.Println("Usage: claune config <natural language prompt>")
		}
	case "help":
		printUsage()
	}
	return nil
}

func showStatus(c config.ClauneConfig) {
	if hooksInstalled() {
		fmt.Println("Installed — claune hooks are active in Claude Code.")
	} else {
		fmt.Println("Not installed — run 'claune install' to add sound hooks.")
	}

	if c.ShouldMute() {
		fmt.Println("Sound: muted")
	} else {
		fmt.Printf("Volume: %.0f%%\n", c.GetVolume()*100)
	}
}

func testSounds(c config.ClauneConfig) {
	fmt.Println("Testing all sounds...")
	for _, event := range []string{"cli:start", "tool:start", "tool:success", "tool:error", "cli:done"} {
		fmt.Printf("  %s ", event)
		audio.PlaySound(event, true, c)
		fmt.Println("OK")
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: claune [claude-args...]    Run Claude Code with sound effects
       claune <command>            Run a claune management command

Passthrough mode (default):
  claune                     Start Claude Code interactively with sounds

Management commands:
  install       Install sound hooks into Claude Code settings
  uninstall     Remove sound hooks from Claude Code settings
  status        Show whether hooks are installed
  play <event>  Play a sound
  test-sounds   Play all sounds to verify audio works
  config <msg>  Natural language configuration (e.g., "mute sound")
  help          Show this help message`)
}
