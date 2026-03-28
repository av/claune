package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func spawnPTY(argv []string) int {
	// Play start sound non-blocking
	playSound("cli:start", false)

	// Build command
	cmd := exec.Command(argv[0], argv[1:]...)

	// Start command in PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting PTY: %v\n", err)
		return 1
	}
	defer ptmx.Close()

	// Handle terminal raw mode and SIGWINCH
	isATTY := term.IsTerminal(int(os.Stdin.Fd()))
	if isATTY {
		// Save and restore terminal state
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting raw mode: %v\n", err)
		}
		defer func() {
			if oldState != nil {
				term.Restore(int(os.Stdin.Fd()), oldState)
			}
		}()

		// Handle SIGWINCH
		sigwinch := make(chan os.Signal, 1)
		signal.Notify(sigwinch, syscall.SIGWINCH)

		// Set initial size
		pty.InheritSize(os.Stdin, ptmx)

		go func() {
			for range sigwinch {
				pty.InheritSize(os.Stdin, ptmx)
			}
		}()
		// Trigger initial resize
		sigwinch <- syscall.SIGWINCH
	}

	// Copy stdin -> PTY (goroutine)
	go func() {
		io.Copy(ptmx, os.Stdin)
		// When stdin closes, send EOF to PTY
		ptmx.Write([]byte{4}) // Ctrl-D
	}()

	// Copy PTY -> stdout (main path)
	io.Copy(os.Stdout, ptmx)

	// Wait for process to finish
	err = cmd.Wait()

	// Play done sound (blocking - wait for it to finish)
	playSound("cli:done", true)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}
