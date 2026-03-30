package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/everlier/claune/internal/ai"
	"github.com/everlier/claune/internal/audio"
	"github.com/everlier/claune/internal/circus"
	"github.com/everlier/claune/internal/config"
)

var clauneSubcommands = map[string]bool{
	"play":          true,
	"install":       true,
	"uninstall":     true,
	"status":        true,
	"test-sounds":   true,
	"help":          true,
	"config":        true,
	"import-circus": true,
	"analyze-log":   true,
	"automap":       true,
	"analyze-resp":  true,
}

func Run(args []string) error {
	if len(args) == 0 || !clauneSubcommands[args[0]] {
		runPassthrough(args)
		return nil
	}

	switch args[0] {
	case "status":
		c, err := config.Load()
		showStatus(c, err)
		return nil
	case "help":
		printUsage()
		return nil
	}

	c, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claune: error loading config: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "help":
		printUsage()
	case "play":
		if len(args) <= 1 {
			fmt.Fprintln(os.Stderr, "Usage: claune play <event> [args...]")
			os.Exit(1)
		}

		if len(args) > 3 {
			event, err := ai.AnalyzeToolIntent(args[2], args[3], c)
			if err != nil && c.AI.Enabled {
				fmt.Fprintf(os.Stderr, "⚠️ AI Semantic Audio Error: %v\n", err)
			}
			if err := audio.PlaySound(event, true, c); err != nil {
				fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
			}
		} else {
			if err := audio.PlaySound(args[1], true, c); err != nil {
				fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
			}
		}
	case "config":
		if len(args) <= 1 {
			fmt.Println("Usage: claune config <natural language prompt>")
			return nil
		}

		prompt := strings.Join(args[1:], " ")
		if err := ai.HandleNaturalLanguageConfig(prompt, &c); err != nil {
			return fmt.Errorf("AI config failed: %w", err)
		}
		fmt.Println("Config updated successfully via AI")
	case "install":
		if err := installHooks(); err != nil {
			return err
		}
	case "uninstall":
		if err := uninstallHooks(); err != nil {
			return err
		}
	case "status":
		showStatus(c, nil)
	case "test-sounds":
		testSounds(c)
	case "import-circus":
		if len(args) > 2 {
			url := args[1]
			filename := args[2]
			if err := circus.ImportMemeSound(url, filename); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else {
				var event string
				if len(args) > 3 {
					event = args[3]
				} else {
					guessed, err := ai.GuessEventForSound(url, filename, c)
					if err != nil {
						fmt.Fprintf(os.Stderr, "AI mapping failed: %v. Please specify an event manually.\n", err)
						return nil
					}
					event = guessed
					fmt.Printf("AI intelligently mapped %s to %s\n", filename, event)
				}

				if c.Sounds == nil {
					c.Sounds = make(map[string]config.EventSoundConfig)
				}
				cachedPath := filepath.Join(audio.SoundCacheDir(), filename)

				// Keep existing config if it exists, just append/overwrite
				eventCfg := c.Sounds[event]
				eventCfg.Paths = append(eventCfg.Paths, cachedPath)
				c.Sounds[event] = eventCfg

				if err := config.Save(c); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to update config: %v\n", err)
				} else {
					fmt.Printf("Mapped %s to event %s\n", filename, event)
				}
			}
		} else {
			fmt.Println("Usage: claune import-circus <url> <filename> [event]")
		}
	case "analyze-log":
		logText, err := io.ReadAll(os.Stdin)
		if err == nil {
			circus.AnalyzeLogSentiment(string(logText), c, true)
		}
	case "automap":
		if len(args) > 1 {
			dir := args[1]
			mapping, err := ai.AutoMapSounds(dir, &c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Automap failed: %v\n", err)
			} else {
				fmt.Println("Sounds mapped successfully:")
				for event, cfg := range mapping {
					if len(cfg.Paths) > 0 {
						fmt.Printf("  - %s mapped to: %s\n", event, cfg.Paths[0])
					}
				}
			}
		} else {
			fmt.Println("Usage: claune automap <directory>")
		}
	case "analyze-resp":
		respText, err := io.ReadAll(os.Stdin)
		if err == nil {
			event, strategy, err := ai.AnalyzeResponseSentiment(string(respText), c)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Analyze response sentiment failed: %v\n", err)
			} else if event != "" {
				if err := audio.PlaySoundWithStrategy(event, strategy, true, c); err != nil {
					fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
				}
			}
		}
	}
	return nil
}

func showStatus(c config.ClauneConfig, configErr error) {
	hooksInstalled, hooksErr := hooksInstallState()
	if hooksErr != nil {
		fmt.Printf("Install state unknown — could not read Claude Code settings: %v\n", hooksErr)
	} else if hooksInstalled {
		fmt.Println("Installed — claune hooks are active in Claude Code.")
	} else {
		fmt.Println("Not installed — run 'claune install' to add sound hooks.")
	}

	if configErr != nil {
		fmt.Printf("Config error: %v\n", configErr)
		return
	}

	if c.ShouldMute() {
		fmt.Println("Sound: muted")
	} else {
		fmt.Printf("Volume: %.0f%%\n", c.GetVolume()*100)
	}
}

func testSounds(c config.ClauneConfig) {
	if c.ShouldMute() {
		return
	}
	fmt.Println("Testing all sounds...")
	for _, event := range []string{"cli:start", "tool:start", "tool:success", "tool:error", "cli:done", "build:success", "test:fail", "panic", "warn"} {
		fmt.Printf("  %s ", event)
		if err := audio.PlaySound(event, true, c); err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Println("OK")
		}
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
  automap <dir> Automatically map sound files in a directory to events using AI
  import-circus <url> <file> [event]  Import a meme sound and optionally map to event
  analyze-log   Analyze log from stdin and play a sound
  analyze-resp  Analyze AI response from stdin and optionally override sound strategy
  help          Show this help message`)
}
