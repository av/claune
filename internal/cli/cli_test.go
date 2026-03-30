package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintUsageWritesHelpToStderr(t *testing.T) {
	output := captureOutput(t, func() {
		printUsage()
	})

	if output.stdout != "" {
		t.Fatalf("printUsage() stdout = %q, want empty", output.stdout)
	}
	assertContains(t, output.stderr, "Usage: claune [claude-args...]")
	assertContains(t, output.stderr, "play <event>")
	assertContains(t, output.stderr, "help          Show this help message")
}

func TestRunStatusShowsInstalledHooksWhenConfigIsMalformed(t *testing.T) {
	home := t.TempDir()
	writeMalformedConfig(t, home)
	writeSettingsFile(t, home, `{
		"hooks": {
			"SessionStart": [
				{
					"matcher": "*",
					"hooks": [
						{
							"type": "command",
							"command": "claune play cli:start",
							"timeout": 5
						}
					]
				}
			]
		}
	}`)

	result := runInSubprocess(t, home, "TEST_RUN_STATUS_WITH_MALFORMED_CONFIG")

	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", result.exitCode, result.stderr)
	}
	assertContains(t, result.stdout, "Installed — claune hooks are active in Claude Code.")
	assertContains(t, result.stdout, "Config error:")
	assertContains(t, result.stdout, "invalid configuration format in ~/.claune.json")
	assertNotContains(t, result.stdout, "Not installed")
	assertNotContains(t, result.stderr, "error loading config")
}

func TestRunStatusShowsUnknownInstallStateWhenSettingsAreMalformed(t *testing.T) {
	home := t.TempDir()
	writeSettingsFile(t, home, `{"hooks":`)

	result := runInSubprocess(t, home, "TEST_RUN_STATUS_WITH_MALFORMED_SETTINGS")

	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", result.exitCode, result.stderr)
	}
	assertContains(t, result.stdout, "Install state unknown — could not read Claude Code settings:")
	assertNotContains(t, result.stdout, "Not installed")
	if !strings.Contains(result.stdout, "Sound: muted") && !strings.Contains(result.stdout, "Volume: ") {
		t.Fatalf("stdout = %q, want sound or volume status line", result.stdout)
	}
}

func TestRunHelpIgnoresMalformedConfigAndPrintsUsage(t *testing.T) {
	home := t.TempDir()
	writeMalformedConfig(t, home)

	result := runInSubprocess(t, home, "TEST_RUN_HELP_WITH_MALFORMED_CONFIG")

	if result.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", result.exitCode, result.stderr)
	}
	assertContains(t, result.stderr, "Usage: claune [claude-args...]")
	assertContains(t, result.stderr, "help          Show this help message")
	assertNotContains(t, result.stderr, "error loading config")
	if result.stdout != "" {
		t.Fatalf("stdout = %q, want empty", result.stdout)
	}
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

func TestCLIProcessHelper(t *testing.T) {
	if os.Getenv("TEST_RUN_STATUS_WITH_MALFORMED_CONFIG") != "1" {
		if os.Getenv("TEST_RUN_STATUS_WITH_MALFORMED_SETTINGS") == "1" {
			if err := Run([]string{"status"}); err != nil {
				t.Fatalf("Run(status) error = %v", err)
			}
			os.Exit(0)
		}

		if os.Getenv("TEST_RUN_HELP_WITH_MALFORMED_CONFIG") != "1" {
			return
		}

		if err := Run([]string{"help"}); err != nil {
			t.Fatalf("Run(help) error = %v", err)
		}
		os.Exit(0)
	}

	if err := Run([]string{"status"}); err != nil {
		t.Fatalf("Run(status) error = %v", err)
	}
	os.Exit(0)
}

type capturedOutput struct {
	stdout string
	stderr string
}

type subprocessResult struct {
	stdout   string
	stderr   string
	exitCode int
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

func runInSubprocess(t *testing.T, home string, helperEnv string) subprocessResult {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestCLIProcessHelper")
	cmd.Env = append(os.Environ(), helperEnv+"=1", "HOME="+home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("subprocess error = %T %v, want *exec.ExitError", err, err)
		}
		exitCode = exitErr.ExitCode()
	}

	return subprocessResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: exitCode,
	}
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

func writeMalformedConfig(t *testing.T, home string) {
	t.Helper()

	configPath := filepath.Join(home, ".claune.json")
	if err := os.WriteFile(configPath, []byte(`{"sounds":`), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func writeSettingsFile(t *testing.T, home string, contents string) {
	t.Helper()

	path := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("output %q does not contain %q", got, want)
	}
}

func assertNotContains(t *testing.T, got string, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Fatalf("output %q unexpectedly contains %q", got, unwanted)
	}
}
