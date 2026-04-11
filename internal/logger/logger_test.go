package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLoggerFunctions(t *testing.T) {
	// Setup a temporary XDG_STATE_HOME for testing
	tmpDir := t.TempDir()
	os.Setenv("XDG_STATE_HOME", tmpDir)
	defer os.Unsetenv("XDG_STATE_HOME")

	// Verify logging writes to file
	Info("test info message %d", 1)
	Error("test error message %s", "err")

	logPath := logFilePath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "test info message 1") {
		t.Errorf("log file does not contain expected info message: %s", contentStr)
	}
	if !strings.Contains(contentStr, "test error message err") {
		t.Errorf("log file does not contain expected error message: %s", contentStr)
	}
}

func TestShowLogs(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_STATE_HOME", tmpDir)
	defer os.Unsetenv("XDG_STATE_HOME")

	Info("log line 1")
	Info("log line 2")
	Error("log line 3")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ShowLogs(2)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("ShowLogs failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "log line 2") {
		t.Errorf("Expected output to contain log line 2, got: %s", output)
	}
	if !strings.Contains(output, "log line 3") {
		t.Errorf("Expected output to contain log line 3, got: %s", output)
	}
	if strings.Contains(output, "log line 1") {
		t.Errorf("Expected output NOT to contain log line 1, got: %s", output)
	}
}

func TestClearLogs(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_STATE_HOME", tmpDir)
	defer os.Unsetenv("XDG_STATE_HOME")

	Info("to be cleared")
	logPath := logFilePath()
	
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("log file should exist before clear")
	}

	err := ClearLogs()
	if err != nil {
		t.Fatalf("ClearLogs failed: %v", err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("log file should be deleted after clear")
	}

	// Clearing when already cleared shouldn't error
	err = ClearLogs()
	if err != nil {
		t.Errorf("ClearLogs failed when no logs exist: %v", err)
	}
}
