package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	c, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claune: error loading config: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "play":
		if len(args) > 1 {
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
		} else {
			fmt.Fprintln(os.Stderr, "Usage: claune play <event> [args...]")
			os.Exit(1)
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
	case "import-circus":
		if len(args) > 2 {
			if err := circus.ImportMemeSound(args[1], args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			} else if len(args) > 3 {
				event := args[3]
				if c.Sounds == nil {
					c.Sounds = make(map[string]config.EventSoundConfig)
				}
				// Use the cache dir path for the imported sound
				cachedPath := filepath.Join(audio.SoundCacheDir(), args[2])
				c.Sounds[event] = config.EventSoundConfig{Paths: []string{cachedPath}}
				if err := config.Save(c); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to update config: %v\n", err)
				} else {
					fmt.Printf("Mapped %s to event %s\n", args[2], event)
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
			if err := ai.AutoMapSounds(dir, &c); err != nil {
				fmt.Fprintf(os.Stderr, "Automap failed: %v\n", err)
			} else {
				fmt.Println("Sounds mapped successfully")
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
