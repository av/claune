package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func TestRunHelpWorksWithMalformedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(home, ".claune.json")
	if err := os.WriteFile(configPath, []byte(`{"sounds":`), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	stdout, stderr, err := runInSubprocess(t, home, []string{"help"})
	if err != nil {
		t.Fatalf("Run(help) error = %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	assertContains(t, stderr, "Usage: claune")
	if strings.Contains(stderr, "error loading config") {
		t.Fatalf("stderr = %q, should not contain config load error", stderr)
	}
}

func TestRunConfigRepairsMalformedConfigUsingDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ANTHROPIC_API_KEY", "")

	configPath := filepath.Join(home, ".claune.json")
	if err := os.WriteFile(configPath, []byte(`{"sounds":`), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", configPath, err)
	}

	stdout, stderr, err := runInSubprocess(t, home, []string{"config", "set", "volume", "to", "50%", "and", "unmute"})
	if err != nil {
		t.Fatalf("Run(config) error = %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	assertContains(t, stdout, "Config updated successfully via AI")
	assertContains(t, stderr, "warning: invalid config")

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

func runInSubprocess(t *testing.T, home string, args []string) (string, string, error) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestRunSubprocessHelper", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"ANTHROPIC_API_KEY=",
		fmt.Sprintf("CLAUNE_SUBPROCESS_HOME=%s", home),
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestRunSubprocessHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	home := os.Getenv("CLAUNE_SUBPROCESS_HOME")
	if home != "" {
		if err := os.Setenv("HOME", home); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set HOME: %v\n", err)
			os.Exit(2)
		}
	}

	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}

	if err := Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Run error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
