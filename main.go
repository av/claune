package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	args := os.Args[1:]

	// Subcommand: claune play <type>
	if len(args) > 0 && args[0] == "play" {
		handlePlaySubcommand(args[1:])
		return
	}

	// Main mode: launch claude with hooks
	tempConfigPath, err := buildHookConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building hook config: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tempConfigPath)

	var cmd []string
	if os.Getenv("CLAUNE_TEST_MODE") == "1" {
		exe, _ := os.Executable()
		scriptDir := filepath.Dir(exe)
		mock := filepath.Join(scriptDir, "mock_claude.py")
		if _, err := os.Stat(mock); err == nil {
			cmd = append([]string{"python3", mock}, args...)
		} else {
			fmt.Fprintln(os.Stderr, "Error: mock_claude.py is missing")
			os.Exit(1)
		}
	} else {
		cmd = append([]string{"claude"}, args...)
		cmd = append(cmd, "--settings", tempConfigPath)
	}

	exitCode := spawnPTY(cmd)
	os.Exit(exitCode)
}
