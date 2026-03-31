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

	stdout, stderr, exitCode, err := runInSubprocess(t, home, []string{"help"})
	if err != nil {
		t.Fatalf("Run(help) error = %v (exit %d)\nstdout:\n%s\nstderr:\n%s", err, exitCode, stdout, stderr)
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

	stdout, stderr, exitCode, err := runInSubprocess(t, home, []string{"config", "set", "volume", "to", "50%", "and", "unmute"})
	if err != nil {
		t.Fatalf("Run(config) error = %v (exit %d)\nstdout:\n%s\nstderr:\n%s", err, exitCode, stdout, stderr)
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

func TestRunManagementCommandsRejectBadUsage(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantStderr   []string
		wantNoStdout bool
	}{
		{
			name:         "config requires prompt",
			args:         []string{"config"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: config requires a natural language prompt", "Usage: claune config <natural language prompt>"},
			wantNoStdout: true,
		},
		{
			name:         "automap requires directory",
			args:         []string{"automap"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: automap requires a directory", "Usage: claune automap <directory>"},
			wantNoStdout: true,
		},
		{
			name:         "automap rejects extra args",
			args:         []string{"automap", "dir", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: automap does not accept additional arguments", "Usage: claune automap <directory>"},
			wantNoStdout: true,
		},
		{
			name:         "import-circus requires url and filename",
			args:         []string{"import-circus"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: import-circus requires a URL and filename", "Usage: claune import-circus <url> <filename> [event]"},
			wantNoStdout: true,
		},
		{
			name:         "import-circus rejects extra args beyond optional event",
			args:         []string{"import-circus", "https://example.com", "sound.mp3", "cli:start", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: import-circus does not accept additional arguments", "Usage: claune import-circus <url> <filename> [event]"},
			wantNoStdout: true,
		},
		{
			name:         "help rejects extra args",
			args:         []string{"help", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: help does not accept additional arguments", "Usage: claune help"},
			wantNoStdout: true,
		},
		{
			name:         "status rejects extra args",
			args:         []string{"status", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: status does not accept additional arguments", "Usage: claune status"},
			wantNoStdout: true,
		},
		{
			name:         "install rejects extra args",
			args:         []string{"install", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: install does not accept additional arguments", "Usage: claune install"},
			wantNoStdout: true,
		},
		{
			name:         "uninstall rejects extra args",
			args:         []string{"uninstall", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: uninstall does not accept additional arguments", "Usage: claune uninstall"},
			wantNoStdout: true,
		},
		{
			name:         "test-sounds rejects extra args",
			args:         []string{"test-sounds", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: test-sounds does not accept additional arguments", "Usage: claune test-sounds"},
			wantNoStdout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			stdout, stderr, exitCode, err := runInSubprocess(t, home, tt.args)
			if err == nil {
				t.Fatalf("Run(%v) error = nil, want exit code %d\nstdout:\n%s\nstderr:\n%s", tt.args, tt.wantExitCode, stdout, stderr)
			}
			if exitCode != tt.wantExitCode {
				t.Fatalf("Run(%v) exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", tt.args, exitCode, tt.wantExitCode, stdout, stderr)
			}
			for _, want := range tt.wantStderr {
				assertContains(t, stderr, want)
			}
			if tt.wantNoStdout && stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
		})
	}
}

func TestRunUnknownCommandStillPassthroughs(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	claudePath := filepath.Join(binDir, "claude")
	if err := os.WriteFile(claudePath, []byte("#!/bin/sh\nprintf 'passthrough:%s\\n' \"$*\"\n"), 0755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", claudePath, err)
	}

	stdout, stderr, exitCode, err := runInSubprocessWithEnv(t, home, []string{"not-a-claune-command", "alpha", "beta"}, []string{fmt.Sprintf("PATH=%s:%s", binDir, os.Getenv("PATH"))})
	if err != nil {
		t.Fatalf("Run(unknown) error = %v (exit %d)\nstdout:\n%s\nstderr:\n%s", err, exitCode, stdout, stderr)
	}
	if exitCode != 0 {
		t.Fatalf("Run(unknown) exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	assertContains(t, stdout, "passthrough:not-a-claune-command alpha beta")
	if strings.Contains(stderr, "does not accept additional arguments") {
		t.Fatalf("stderr = %q, should not contain claune usage validation for unknown commands", stderr)
	}
}

func TestRunManagementCommandsBadUsageWinsOverMalformedConfig(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantStderr   []string
		avoidStderr  []string
	}{
		{
			name:         "analyze-log rejects extra args",
			args:         []string{"analyze-log", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: analyze-log does not accept additional arguments", "Usage: claune analyze-log"},
			avoidStderr:  []string{"error loading config"},
		},
		{
			name:         "analyze-resp rejects extra args",
			args:         []string{"analyze-resp", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: analyze-resp does not accept additional arguments", "Usage: claune analyze-resp"},
			avoidStderr:  []string{"error loading config"},
		},
		{
			name:         "config missing prompt",
			args:         []string{"config"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: config requires a natural language prompt", "Usage: claune config <natural language prompt>"},
			avoidStderr:  []string{"error loading config", "warning: invalid config"},
		},
		{
			name:         "status rejects extra args",
			args:         []string{"status", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: status does not accept additional arguments", "Usage: claune status"},
			avoidStderr:  []string{"error loading config"},
		},
		{
			name:         "test-sounds rejects extra args",
			args:         []string{"test-sounds", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: test-sounds does not accept additional arguments", "Usage: claune test-sounds"},
			avoidStderr:  []string{"error loading config"},
		},
		{
			name:         "automap requires directory",
			args:         []string{"automap"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: automap requires a directory", "Usage: claune automap <directory>"},
			avoidStderr:  []string{"error loading config"},
		},
		{
			name:         "automap rejects extra args",
			args:         []string{"automap", "dir", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: automap does not accept additional arguments", "Usage: claune automap <directory>"},
			avoidStderr:  []string{"error loading config"},
		},
		{
			name:         "import-circus requires url and filename",
			args:         []string{"import-circus"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: import-circus requires a URL and filename", "Usage: claune import-circus <url> <filename> [event]"},
			avoidStderr:  []string{"error loading config"},
		},
		{
			name:         "import-circus rejects extra args beyond optional event",
			args:         []string{"import-circus", "https://example.com", "sound.mp3", "cli:start", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: import-circus does not accept additional arguments", "Usage: claune import-circus <url> <filename> [event]"},
			avoidStderr:  []string{"error loading config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			configPath := filepath.Join(home, ".claune.json")
			if err := os.WriteFile(configPath, []byte(`{"sounds":`), 0644); err != nil {
				t.Fatalf("WriteFile(%q) error = %v", configPath, err)
			}

			stdout, stderr, exitCode, err := runInSubprocess(t, home, tt.args)
			if err == nil {
				t.Fatalf("Run(%v) error = nil, want exit code %d\nstdout:\n%s\nstderr:\n%s", tt.args, tt.wantExitCode, stdout, stderr)
			}
			if exitCode != tt.wantExitCode {
				t.Fatalf("Run(%v) exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", tt.args, exitCode, tt.wantExitCode, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			for _, want := range tt.wantStderr {
				assertContains(t, stderr, want)
			}
			for _, avoid := range tt.avoidStderr {
				if strings.Contains(stderr, avoid) {
					t.Fatalf("stderr = %q, should not contain %q", stderr, avoid)
				}
			}
		})
	}
}

func TestRunAnalyzeCommandsRejectUnexpectedArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantStderr   []string
	}{
		{
			name:         "analyze-log rejects extra args",
			args:         []string{"analyze-log", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: analyze-log does not accept additional arguments", "Usage: claune analyze-log"},
		},
		{
			name:         "analyze-resp rejects extra args",
			args:         []string{"analyze-resp", "extra"},
			wantExitCode: 1,
			wantStderr:   []string{"claune: analyze-resp does not accept additional arguments", "Usage: claune analyze-resp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			stdout, stderr, exitCode, err := runInSubprocess(t, home, tt.args)
			if err == nil {
				t.Fatalf("Run(%v) error = nil, want exit code %d\nstdout:\n%s\nstderr:\n%s", tt.args, tt.wantExitCode, stdout, stderr)
			}
			if exitCode != tt.wantExitCode {
				t.Fatalf("Run(%v) exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", tt.args, exitCode, tt.wantExitCode, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			for _, want := range tt.wantStderr {
				assertContains(t, stderr, want)
			}
		})
	}
}

func TestRunAnalyzeCommandsFailLoudlyOnStdinReadError(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantExitCode int
		wantStderr   string
	}{
		{
			name:         "analyze-log stdin read failure",
			args:         []string{"analyze-log"},
			wantExitCode: 1,
			wantStderr:   "claune: failed to read stdin for analyze-log:",
		},
		{
			name:         "analyze-resp stdin read failure",
			args:         []string{"analyze-resp"},
			wantExitCode: 1,
			wantStderr:   "claune: failed to read stdin for analyze-resp:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			stdout, stderr, exitCode, err := runInSubprocessWithEnv(t, home, tt.args, []string{"CLAUNE_SUBPROCESS_STDIN_MODE=write-only"})
			if err == nil {
				t.Fatalf("Run(%v) error = nil, want exit code %d\nstdout:\n%s\nstderr:\n%s", tt.args, tt.wantExitCode, stdout, stderr)
			}
			if exitCode != tt.wantExitCode {
				t.Fatalf("Run(%v) exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", tt.args, exitCode, tt.wantExitCode, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			assertContains(t, stderr, tt.wantStderr)
		})
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

func runInSubprocess(t *testing.T, home string, args []string) (string, string, int, error) {
	t.Helper()
	return runInSubprocessWithEnv(t, home, args, nil)
}

func runInSubprocessWithEnv(t *testing.T, home string, args []string, extraEnv []string) (string, string, int, error) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestRunSubprocessHelper", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"ANTHROPIC_API_KEY=",
		fmt.Sprintf("CLAUNE_SUBPROCESS_HOME=%s", home),
	)
	cmd.Env = append(cmd.Env, extraEnv...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0, nil
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return stdout.String(), stderr.String(), -1, err
	}

	return stdout.String(), stderr.String(), exitErr.ExitCode(), err
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

	if stdinMode := os.Getenv("CLAUNE_SUBPROCESS_STDIN_MODE"); stdinMode != "" {
		switch stdinMode {
		case "write-only":
			_, stdinWriter, err := os.Pipe()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create stdin pipe: %v\n", err)
				os.Exit(2)
			}
			os.Stdin = stdinWriter
		default:
			fmt.Fprintf(os.Stderr, "unknown stdin mode: %s\n", stdinMode)
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
