package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallHooks(t *testing.T) {
	dir := t.TempDir()
	settingsPathFunc = func() string {
		return filepath.Join(dir, "settings.json")
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	err := installHooks()
	if err != nil {
		t.Fatalf("Failed to install hooks: %v", err)
	}

	if !hooksInstalled() {
		t.Errorf("Expected hooks to be installed")
	}
}

func TestWriteSettings_ErrorMkdir(t *testing.T) {
	dir := t.TempDir()

	// Create a file where directory should be
	badDir := filepath.Join(dir, "bad")
	os.WriteFile(badDir, []byte("file"), 0644)

	settingsPathFunc = func() string {
		return filepath.Join(badDir, "settings.json")
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	err := writeSettings(map[string]interface{}{"test": "value"})
	if err == nil {
		t.Errorf("Expected error when MkdirAll fails")
	}
}

func TestDirectHookCmd(t *testing.T) {
	cmd := directHookCmd("/path/to/wav", "test:event")
	if cmd == "" {
		t.Errorf("Expected non-empty command")
	}
}

func TestReadSettings_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "settings.json")
	os.WriteFile(file, []byte("{invalid json"), 0644)

	settingsPathFunc = func() string {
		return file
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	_, err := readSettings()
	if err == nil {
		t.Errorf("Expected error for invalid JSON")
	}
}

func TestHooksInstalled_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "settings.json")
	os.WriteFile(file, []byte("{invalid json"), 0644)

	settingsPathFunc = func() string {
		return file
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	if hooksInstalled() {
		t.Errorf("Expected false for hooksInstalled when settings are invalid")
	}
}

func TestUninstallHooks_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "settings.json")
	os.WriteFile(file, []byte("{invalid json"), 0644)

	settingsPathFunc = func() string {
		return file
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	err := uninstallHooks()
	if err == nil {
		t.Errorf("Expected error for uninstallHooks when settings are invalid")
	}
}

func TestParseHookEntries_NilOrInvalid(t *testing.T) {
	hooksMap := map[string]interface{}{
		"invalid": make(chan int), // unmarshalable
		"nilval":  nil,
	}
	if entries := parseHookEntries(hooksMap, "missing"); entries != nil {
		t.Errorf("Expected nil for missing key")
	}
	if entries := parseHookEntries(hooksMap, "nilval"); entries != nil {
		t.Errorf("Expected nil for nilval")
	}
	if entries := parseHookEntries(hooksMap, "invalid"); entries != nil {
		t.Errorf("Expected nil for invalid")
	}
}

func TestInstallHooks_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "settings.json")
	os.WriteFile(file, []byte("{invalid json"), 0644)

	settingsPathFunc = func() string {
		return file
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	err := installHooks()
	if err == nil {
		t.Errorf("Expected error for installHooks when settings are invalid")
	}
}

func TestUninstallHooks_NoHooks(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "settings.json")
	os.WriteFile(file, []byte("{}"), 0644)

	settingsPathFunc = func() string {
		return file
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	err := uninstallHooks()
	if err != nil {
		t.Errorf("Expected no error for uninstallHooks when no hooks exist, got %v", err)
	}
}

func TestInstallHooks_WriteError(t *testing.T) {
	dir := t.TempDir()
	badDir := filepath.Join(dir, "bad")
	os.WriteFile(badDir, []byte("file"), 0644)
	settingsPathFunc = func() string {
		return filepath.Join(badDir, "settings.json")
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	err := installHooks()
	if err == nil {
		t.Errorf("Expected error for installHooks when settings write fails")
	}
}

func TestUninstallHooks_WriteError(t *testing.T) {
	dir := t.TempDir()
	badDir := filepath.Join(dir, "bad")

	// Create valid settings first so read passes
	os.MkdirAll(badDir, 0755)
	file := filepath.Join(badDir, "settings.json")
	os.WriteFile(file, []byte(`{"hooks": {"SessionStart": [{"matcher":"","hooks":[{"type":"command","command":"claune play foo"}]}]}}`), 0644)

	// Now make directory unwritable so write fails
	os.Chmod(badDir, 0500)
	defer os.Chmod(badDir, 0755)

	settingsPathFunc = func() string {
		return file
	}
	defer func() {
		settingsPathFunc = func() string {
			return filepath.Join(os.TempDir(), "settings.json")
		}
	}()

	err := uninstallHooks()
	if err == nil {
		t.Errorf("Expected error for uninstallHooks when settings write fails")
	}
}
