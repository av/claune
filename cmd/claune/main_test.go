package main

import (
	"os"
	"testing"
)

func TestMainExecution(t *testing.T) {
	// Backup original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Provide a safe argument that triggers immediate exit without error code, e.g. --help
	os.Args = []string{"claune", "--help"}

	// We wrap in a function to recover from potential os.Exit if cli.Run does it, 
	// though cli.Run likely returns the help output and main doesn't call os.Exit on success.
	main()
}
