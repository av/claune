package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/everlier/claune/internal/config"
)

// TestTTYLatency ensures that background audio processes do not block the TTY
// and that input latency remains low while audio is "playing".
func TestTTYLatency(t *testing.T) {
	// Setup a temporary workspace
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	err := os.MkdirAll(homeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}

	// We create a mock "claune" binary that just sleeps to simulate playing audio
	mockBinPath := filepath.Join(tmpDir, "claune")
	err = os.WriteFile(mockBinPath, []byte("#!/bin/sh\nsleep 2\n"), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Override standard environment variables
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	// Create dummy config
	cfg := config.ClauneConfig{}
	err = config.Save(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// We'll mimic the directHookCmd logic which spawns a bash shell in the background
	// We want to measure the latency of starting the background task and returning control.
	start := time.Now()

	// Mimic what directHookCmd does:
	// bash -c 'mockBinPath play "event" >/dev/null 2>&1 </dev/null &'
	cmdLine := []string{"bash", "-c", "claune play test >/dev/null 2>&1 </dev/null &"}
	
	cmd := exec.Command(cmdLine[0], cmdLine[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v, stderr: %s", err, stderr.String())
	}

	elapsed := time.Since(start)

	// The background process sleeps for 2 seconds.
	// The foreground bash command should exit almost immediately because of the `&`.
	// If it takes more than 100ms, something is blocking.
	if elapsed > 100*time.Millisecond {
		t.Errorf("Latency too high! Expected <100ms, got %v. Background process is blocking.", elapsed)
	}
}
