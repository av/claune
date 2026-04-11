package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestIsClauneHook(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"claune play test", true},
		{"CLAUNE_ACTIVE=1 claune play", true},
		{"bash -c 'something else'", false},
		{"echo hello", false},
	}

	for _, tt := range tests {
		if got := isClauneHook(tt.cmd); got != tt.expected {
			t.Errorf("isClauneHook(%q) = %v, want %v", tt.cmd, got, tt.expected)
		}
	}
}

func TestRemoveClauneHooks(t *testing.T) {
	entries := []HookEntry{
		{
			Matcher: ".*",
			Hooks: []Hook{
				{Type: "command", Command: "echo hello"},
				{Type: "command", Command: "claune play sound"},
			},
		},
		{
			Matcher: "other",
			Hooks: []Hook{
				{Type: "command", Command: "claune play something"},
			},
		},
	}

	kept, changed := removeClauneHooks(entries)
	if !changed {
		t.Errorf("Expected changed = true")
	}
	if len(kept) != 1 {
		t.Fatalf("Expected 1 entry kept, got %d", len(kept))
	}
	if len(kept[0].Hooks) != 1 {
		t.Fatalf("Expected 1 hook kept in entry, got %d", len(kept[0].Hooks))
	}
	if kept[0].Hooks[0].Command != "echo hello" {
		t.Errorf("Expected kept hook to be 'echo hello', got %q", kept[0].Hooks[0].Command)
	}
}

func TestUninstallHooks(t *testing.T) {
	dir := t.TempDir()
	settingsPathFunc = func() string {
		return filepath.Join(dir, "settings.json")
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	// Create initial settings with hooks
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "claune play test",
						},
						map[string]interface{}{
							"type":    "command",
							"command": "echo hello",
						},
					},
				},
			},
		},
	}
	err := writeSettings(settings)
	if err != nil {
		t.Fatalf("Failed to write settings: %v", err)
	}

	if !hooksInstalled() {
		t.Errorf("Expected hooks to be installed")
	}

	err = uninstallHooks()
	if err != nil {
		t.Fatalf("Failed to uninstall hooks: %v", err)
	}

	if hooksInstalled() {
		t.Errorf("Expected hooks to be uninstalled")
	}

	// Read settings back
	data, err := os.ReadFile(settingsPathFunc())
	if err != nil {
		t.Fatalf("Failed to read settings: %v", err)
	}

	var newSettings map[string]interface{}
	json.Unmarshal(data, &newSettings)

	hooksMap, ok := newSettings["hooks"].(map[string]interface{})
	if ok {
		if sessionStart, exists := hooksMap["SessionStart"]; exists {
			entries := sessionStart.([]interface{})
			if len(entries) > 0 {
				entry := entries[0].(map[string]interface{})
				hooks := entry["hooks"].([]interface{})
				if len(hooks) != 1 {
					t.Errorf("Expected 1 hook left, got %d", len(hooks))
				}
			}
		}
	}

	// Uninstall again should not fail
	err = uninstallHooks()
	if err != nil {
		t.Fatalf("Failed second uninstall: %v", err)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	codeStr := os.Getenv("GO_HELPER_EXIT_CODE")
	code, _ := strconv.Atoi(codeStr)
	if os.Getenv("GO_HELPER_SIGNAL") == "1" {
		p, _ := os.FindProcess(os.Getpid())
		p.Signal(os.Kill)
		select {}
	}
	os.Exit(code)
}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=^TestHelperProcess$", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if code := os.Getenv("MOCK_EXIT_CODE"); code != "" {
		cmd.Env = append(cmd.Env, "GO_HELPER_EXIT_CODE="+code)
	}
	if sig := os.Getenv("MOCK_SIGNAL"); sig != "" {
		cmd.Env = append(cmd.Env, "GO_HELPER_SIGNAL="+sig)
	}
	return cmd
}

func TestRunPassthrough(t *testing.T) {
	originalExecCommand := execCommand
	originalExecLookPath := execLookPath
	originalOsExit := osExit
	defer func() {
		execCommand = originalExecCommand
		execLookPath = originalExecLookPath
		osExit = originalOsExit
	}()

	execCommand = fakeExecCommand

	t.Run("success", func(t *testing.T) {
		execLookPath = func(file string) (string, error) {
			return "/fake/claude", nil
		}
		var exitCode int = -1
		defer func() {
			if r := recover(); r != nil {
				exitCode = r.(int)
			}
		}()
		osExit = func(code int) {
			panic(code)
		}
		os.Setenv("MOCK_EXIT_CODE", "0")
		defer os.Unsetenv("MOCK_EXIT_CODE")

		runPassthrough([]string{"arg1"})

		if exitCode != -1 {
			t.Errorf("expected exit code -1, got %d", exitCode)
		}
	})

	t.Run("exit code 126 on start failure", func(t *testing.T) {
		execLookPath = func(file string) (string, error) {
			return "/fake/claude", nil
		}
		var exitCode int = -1
		defer func() {
			if r := recover(); r != nil {
				exitCode = r.(int)
			}
		}()
		osExit = func(code int) {
			panic(code)
		}

		execCommand = func(command string, args ...string) *exec.Cmd {
			return exec.Command("/does/not/exist/binary/foo")
		}
		defer func() { execCommand = fakeExecCommand }()

		runPassthrough([]string{"arg1"})

		if exitCode != 126 {
			t.Errorf("expected exit code 126, got %d", exitCode)
		}
	})

	t.Run("exit code 127 on lookpath failure", func(t *testing.T) {
		execLookPath = func(file string) (string, error) {
			return "", os.ErrNotExist
		}
		var exitCode int = -1
		defer func() {
			if r := recover(); r != nil {
				exitCode = r.(int)
			}
		}()
		osExit = func(code int) {
			panic(code)
		}

		runPassthrough([]string{"arg1"})

		if exitCode != 127 {
			t.Errorf("expected exit code 127, got %d", exitCode)
		}
	})

	t.Run("fork bomb detection", func(t *testing.T) {
		exe, _ := os.Executable()
		execLookPath = func(file string) (string, error) {
			return exe, nil
		}
		var exitCode int = -1
		defer func() {
			if r := recover(); r != nil {
				exitCode = r.(int)
			}
		}()
		osExit = func(code int) {
			panic(code)
		}

		runPassthrough([]string{"arg1"})

		if exitCode != 1 {
			t.Errorf("expected exit code 1, got %d", exitCode)
		}
	})

	t.Run("nested execution detection", func(t *testing.T) {
		os.Setenv("CLAUNE_ACTIVE", "1")
		defer os.Unsetenv("CLAUNE_ACTIVE")
		execLookPath = func(file string) (string, error) {
			return "/fake/claude", nil
		}
		var exitCode int = -1
		defer func() {
			if r := recover(); r != nil {
				exitCode = r.(int)
			}
		}()
		osExit = func(code int) {
			panic(code)
		}

		runPassthrough([]string{"arg1"})

		if exitCode != -1 {
			t.Errorf("expected exit code -1, got %d", exitCode)
		}
	})

	t.Run("child exit 1", func(t *testing.T) {
		execLookPath = func(file string) (string, error) {
			return "/fake/claude", nil
		}
		var exitCode int = -1
		defer func() {
			if r := recover(); r != nil {
				exitCode = r.(int)
			}
		}()
		osExit = func(code int) {
			panic(code)
		}
		os.Setenv("MOCK_EXIT_CODE", "1")
		defer os.Unsetenv("MOCK_EXIT_CODE")

		runPassthrough([]string{"arg1"})

		if exitCode != 1 {
			t.Errorf("expected exit code 1, got %d", exitCode)
		}
	})

	t.Run("child signaled", func(t *testing.T) {
		execLookPath = func(file string) (string, error) {
			return "/fake/claude", nil
		}
		var exitCode int = -1
		defer func() {
			if r := recover(); r != nil {
				exitCode = r.(int)
			}
		}()
		osExit = func(code int) {
			panic(code)
		}
		os.Setenv("MOCK_SIGNAL", "1")
		defer os.Unsetenv("MOCK_SIGNAL")

		runPassthrough([]string{"arg1"})

		// signal kill is 9, 128 + 9 = 137
		if exitCode != 137 {
			t.Errorf("expected exit code 137, got %d", exitCode)
		}
	})
}
