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

const playUsage = "Usage: claune play <event>\n       claune play <event> <tool-name> <tool-input>"

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
	case "help":
		ensureExactArgs(args, 1, "claune: help does not accept additional arguments", "Usage: claune help")
		printUsage()
		return nil
	case "install":
		ensureExactArgs(args, 1, "claune: install does not accept additional arguments", "Usage: claune install")
		return installHooks()
	case "uninstall":
		ensureExactArgs(args, 1, "claune: uninstall does not accept additional arguments", "Usage: claune uninstall")
		return uninstallHooks()
	}

	validateManagementArgs(args)

	c, err := loadCommandConfig(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "claune: error loading config: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "play":
		if len(args) == 4 {
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
	case "status":
		showStatus(c)
	case "test-sounds":
		testSounds(c)
	case "config":
		prompt := strings.Join(args[1:], " ")
		if err := ai.HandleNaturalLanguageConfig(prompt, &c); err != nil {
			return fmt.Errorf("AI config failed: %w", err)
		}
		fmt.Println("Config updated successfully via AI")
	case "import-circus":
		url := args[1]
		filename := args[2]
		cachedPath := filepath.Join(audio.SoundCacheDir(), filename)
		if err := circus.ImportMemeSound(url, filename); err != nil {
			fmt.Fprintf(os.Stderr, "Import failed: %v\n", err)
			os.Exit(1)
		} else {
			var event string
			if len(args) == 4 {
				event = args[3]
			} else {
				guessed, err := ai.GuessEventForSound(url, filename, c)
				if err != nil {
					fmt.Printf("Imported %s to %s, but could not map it to an event automatically.\n", filename, cachedPath)
					fmt.Fprintf(os.Stderr, "AI mapping failed: %v. Please rerun with an explicit event to update ~/.claune.json.\n", err)
					return nil
				}
				event = guessed
			}

			if c.Sounds == nil {
				c.Sounds = make(map[string]config.EventSoundConfig)
			}

			// Keep existing config if it exists, just append/overwrite
			eventCfg := c.Sounds[event]
			eventCfg.Paths = append(eventCfg.Paths, cachedPath)
			c.Sounds[event] = eventCfg

			if err := config.Save(c); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update config: %v\n", err)
				fmt.Fprintf(os.Stderr, "%s was downloaded to %s, but claune could not update ~/.claune.json.\n", filename, cachedPath)
				os.Exit(1)
			} else {
				fmt.Printf("Imported %s and mapped it to event %s\n", filename, event)
			}
		}
	case "analyze-log":
		var logText string
		if len(args) > 1 {
			logText = strings.Join(args[1:], " ")
		} else {
			logText = mustReadStdin("analyze-log")
		}
		circus.AnalyzeLogSentiment(logText, c, true)
	case "automap":
		dir := args[1]
		mapping, err := ai.AutoMapSounds(dir, &c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Automap failed: %v\n", err)
			os.Exit(1)
		} else {
			fmt.Println("Sounds mapped successfully:")
			for event, cfg := range mapping {
				if len(cfg.Paths) > 0 {
					fmt.Printf("  - %s mapped to: %s\n", event, cfg.Paths[0])
				}
			}
		}
	case "analyze-resp":
		var respText string
		if len(args) > 1 {
			respText = strings.Join(args[1:], " ")
		} else {
			respText = mustReadStdin("analyze-resp")
		}
		event, strategy, err := ai.AnalyzeResponseSentiment(respText, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Analyze response sentiment failed: %v\n", err)
			os.Exit(1)
		} else if event != "" {
			if err := audio.PlaySoundWithStrategy(event, strategy, true, c); err != nil {
				fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
			}
		}
	}
	return nil
}

func ensureExactArgs(args []string, expected int, message string, usage string) {
	if len(args) != expected {
		exitUsageError(message, usage)
	}
}

func validateManagementArgs(args []string) {
	switch args[0] {
	case "status":
		ensureExactArgs(args, 1, "claune: status does not accept additional arguments", "Usage: claune status")
	case "play":
		if err := validatePlayArgs(args); err != nil {
			exitUsageError(err.Error(), playUsage)
		}
	case "test-sounds":
		ensureExactArgs(args, 1, "claune: test-sounds does not accept additional arguments", "Usage: claune test-sounds")
	case "config":
		if len(args) <= 1 {
			exitUsageError("claune: config requires a natural language prompt", "Usage: claune config <natural language prompt>")
		}
	case "automap":
		switch len(args) {
		case 1:
			exitUsageError("claune: automap requires a directory", "Usage: claune automap <directory>")
		case 2:
			return
		default:
			exitUsageError("claune: automap does not accept additional arguments", "Usage: claune automap <directory>")
		}
	case "import-circus":
		switch len(args) {
		case 1, 2:
			exitUsageError("claune: import-circus requires a URL and name", "Usage: claune import-circus <url> <name> [event]")
		case 3, 4:
			return
		default:
			exitUsageError("claune: import-circus does not accept additional arguments", "Usage: claune import-circus <url> <name> [event]")
		}
	case "analyze-log":
		return
	case "analyze-resp":
		return
	}
}

func validatePlayArgs(args []string) error {
	switch len(args) {
	case 1:
		return fmt.Errorf("claune: play requires an event")
	case 2, 4:
		return nil
	case 3:
		return fmt.Errorf("claune: play accepts either <event> or <event> <tool-name> <tool-input>")
	default:
		return fmt.Errorf("claune: play does not accept additional arguments")
	}
}

func exitUsageError(message string, usage string) {
	fmt.Fprintln(os.Stderr, message)
	fmt.Fprintln(os.Stderr, usage)
	os.Exit(1)
}

func mustReadStdin(command string) string {
	info, err := os.Stdin.Stat()
	if err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
		fmt.Fprintf(os.Stderr, "claune: %s requires piped input or direct string arguments\n", command)
		os.Exit(1)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claune: failed to read stdin for %s: %v\n", command, err)
		os.Exit(1)
	}

	return string(data)
}

func loadCommandConfig(command string) (config.ClauneConfig, error) {
	c, err := config.Load()
	if err == nil {
		return c, nil
	}

	if command == "config" && config.IsInvalidConfigError(err) {
		fmt.Fprintf(os.Stderr, "claune: warning: invalid config, starting from defaults: %v\n", err)
		return c, nil
	}

	return c, err
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
  play <event>  Play a sound for an event
  play <event> <tool-name> <tool-input>
                 Play a sound using semantic tool context
  test-sounds   Play all sounds to verify audio works
  config <msg>  Natural language configuration (e.g., "mute sound")
  automap <dir> Automatically map sound files in a directory to events using AI
  import-circus <url> <name> [event]  Import a meme sound (no slashes allowed) and optionally map to event
  analyze-log   Analyze log from stdin and play a sound
  analyze-resp  Analyze AI response from stdin and optionally override sound strategy
  help          Show this help message`)
}
