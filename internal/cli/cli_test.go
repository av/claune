package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	printUsage()
}

func TestRunConfigUsesFullNaturalLanguagePrompt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ANTHROPIC_API_KEY", "")

	output := captureOutput(t, func() {
		if err := Run([]string{"config", "set", "volume", "to", "50%", "and", "unmute"}); err != nil {
			t.Fatalf("Run(config) error = %v", err)
		}
	})

	if output.stderr != "" {
		t.Fatalf("stderr = %q, want empty", output.stderr)
	}
	assertContains(t, output.stdout, "Config updated successfully via AI")

	configPath := filepath.Join(home, ".claune.json")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", configPath, err)
	}

	var persisted map[string]any
	if err := json.Unmarshal(configBytes, &persisted); err != nil {
		t.Fatalf("json.Unmarshal(config) error = %v", err)
	}

	if got := persisted["volume"]; got != 0.5 {
		t.Fatalf("persisted volume = %#v, want 0.5", got)
	}
	if got := persisted["mute"]; got != false {
		t.Fatalf("persisted mute = %#v, want false", got)
	}
}

type capturedOutput struct {
	stdout string
	stderr string
}

func captureOutput(t *testing.T, fn func()) capturedOutput {
	t.Helper()

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() stdout error = %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() stderr error = %v", err)
	}

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	t.Cleanup(func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	})

	fn()

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("stdoutWriter.Close() error = %v", err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("stderrWriter.Close() error = %v", err)
	}

	stdout, err := readPipe(stdoutReader)
	if err != nil {
		t.Fatalf("read stdout error = %v", err)
	}
	stderr, err := readPipe(stderrReader)
	if err != nil {
		t.Fatalf("read stderr error = %v", err)
	}

	return capturedOutput{stdout: stdout, stderr: stderr}
}

func readPipe(file *os.File) (string, error) {
	defer file.Close()

	data, err := os.ReadFile(file.Name())
	if err == nil {
		return string(data), nil
	}

	var buffer bytes.Buffer
	_, copyErr := buffer.ReadFrom(file)
	if copyErr != nil {
		return "", copyErr
	}
	return buffer.String(), nil
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("output %q does not contain %q", got, want)
	}
}
