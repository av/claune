package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "play":
		handlePlaySubcommand(args[1:])
	case "install":
		if err := installHooks(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := uninstallHooks(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "status":
		showStatus()
	case "test-sounds":
		testSounds()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: claune <command>

Commands:
  install       Install sound hooks into Claude Code settings
  uninstall     Remove sound hooks from Claude Code settings
  status        Show whether hooks are installed
  play <event>  Play a sound (used by hooks, not usually called directly)
  test-sounds   Play all sounds to verify audio works

Config: ~/.claune.json
  {"mute": false, "volume": 0.7, "sounds": {"tool:success": "/path/to/custom.wav"}}`)
}
