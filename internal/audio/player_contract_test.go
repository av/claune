package audio

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/everlier/claune/internal/config"
)

func installAudioTestSeams(t *testing.T, stderr *bytes.Buffer, custom func(string, float64, bool) error, embedded func(string, float64, bool) error, home func() (string, error)) {
	t.Helper()

	oldCustom := customSoundPlayer
	oldEmbedded := embeddedSoundPlayer
	oldStderr := stderrWriter
	oldHome := userHomeDir

	if custom != nil {
		customSoundPlayer = custom
	}
	if embedded != nil {
		embeddedSoundPlayer = embedded
	}
	if stderr != nil {
		stderrWriter = stderr
	}
	if home != nil {
		userHomeDir = home
	}

	t.Cleanup(func() {
		customSoundPlayer = oldCustom
		embeddedSoundPlayer = oldEmbedded
		stderrWriter = oldStderr
		userHomeDir = oldHome
	})
}

func unmutedConfig() config.ClauneConfig {
	unmute := false
	return config.ClauneConfig{Mute: &unmute}
}

func TestPlaySoundWithStrategyBuiltInMissingCustomFallsBackWithWarning(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.mp3")
	stderr := &bytes.Buffer{}
	var embeddedCalls []string

	installAudioTestSeams(t, stderr,
		func(string, float64, bool) error {
			t.Fatal("custom player should not run for missing path")
			return nil
		},
		func(soundFile string, _ float64, _ bool) error {
			embeddedCalls = append(embeddedCalls, soundFile)
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"cli:start": {Paths: []string{missingPath}},
	}

	if err := PlaySoundWithStrategy("cli:start", "", false, c); err != nil {
		t.Fatalf("PlaySoundWithStrategy() error = %v, want nil", err)
	}
	if len(embeddedCalls) != 1 || embeddedCalls[0] != "cli-start.mp3" {
		t.Fatalf("embedded calls = %v, want [cli-start.mp3]", embeddedCalls)
	}
	if got := stderr.String(); !strings.Contains(got, "Warning: invalid custom sound path") || !strings.Contains(got, missingPath) {
		t.Fatalf("stderr = %q, want invalid custom path warning for %q", got, missingPath)
	}
}

func TestPlaySoundWithStrategyBuiltInUnplayableCustomFallsBackWithWarning(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "bad.mp3")
	if err := os.WriteFile(customPath, []byte("not playable"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stderr := &bytes.Buffer{}
	var customCalls []string
	var embeddedCalls []string

	installAudioTestSeams(t, stderr,
		func(path string, _ float64, _ bool) error {
			customCalls = append(customCalls, path)
			return errors.New("decode failed")
		},
		func(soundFile string, _ float64, _ bool) error {
			embeddedCalls = append(embeddedCalls, soundFile)
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"cli:start": {Paths: []string{customPath}},
	}

	if err := PlaySoundWithStrategy("cli:start", "", false, c); err != nil {
		t.Fatalf("PlaySoundWithStrategy() error = %v, want nil", err)
	}
	if len(customCalls) != 1 || customCalls[0] != customPath {
		t.Fatalf("custom calls = %v, want [%s]", customCalls, customPath)
	}
	if len(embeddedCalls) != 1 || embeddedCalls[0] != "cli-start.mp3" {
		t.Fatalf("embedded calls = %v, want [cli-start.mp3]", embeddedCalls)
	}
	if got := stderr.String(); !strings.Contains(got, "Warning: failed to play custom sound") || !strings.Contains(got, "decode failed") {
		t.Fatalf("stderr = %q, want playback failure warning", got)
	}
}

func TestPlaySoundWithStrategyBuiltInValidCustomIsAuthoritative(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "good.mp3")
	if err := os.WriteFile(customPath, []byte("playable"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stderr := &bytes.Buffer{}
	var customCalls []string

	installAudioTestSeams(t, stderr,
		func(path string, _ float64, _ bool) error {
			customCalls = append(customCalls, path)
			return nil
		},
		func(string, float64, bool) error {
			t.Fatal("embedded fallback should not run when custom playback succeeds")
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"cli:start": {Paths: []string{customPath}},
	}

	if err := PlaySoundWithStrategy("cli:start", "", false, c); err != nil {
		t.Fatalf("PlaySoundWithStrategy() error = %v, want nil", err)
	}
	if len(customCalls) != 1 || customCalls[0] != customPath {
		t.Fatalf("custom calls = %v, want [%s]", customCalls, customPath)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestPlaySoundWithStrategyNonBuiltInValidCustomSucceeds(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "custom.mp3")
	if err := os.WriteFile(customPath, []byte("playable"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stderr := &bytes.Buffer{}
	var customCalls []string

	installAudioTestSeams(t, stderr,
		func(path string, _ float64, _ bool) error {
			customCalls = append(customCalls, path)
			return nil
		},
		func(string, float64, bool) error {
			t.Fatal("embedded fallback should not run for non-built-in custom success")
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"custom:event": {Paths: []string{customPath}},
	}

	if err := PlaySoundWithStrategy("custom:event", "", false, c); err != nil {
		t.Fatalf("PlaySoundWithStrategy() error = %v, want nil", err)
	}
	if len(customCalls) != 1 || customCalls[0] != customPath {
		t.Fatalf("custom calls = %v, want [%s]", customCalls, customPath)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestPlaySoundWithStrategyNonBuiltInNoPlayableCustomReturnsUnknownEvent(t *testing.T) {
	stderr := &bytes.Buffer{}

	installAudioTestSeams(t, stderr,
		func(string, float64, bool) error {
			t.Fatal("custom player should not run for empty custom path")
			return nil
		},
		func(string, float64, bool) error {
			t.Fatal("embedded fallback should not run for unknown event without default")
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"custom:event": {Paths: []string{""}},
	}

	err := PlaySoundWithStrategy("custom:event", "", false, c)
	if err == nil {
		t.Fatal("PlaySoundWithStrategy() error = nil, want unknown event error")
	}
	if !strings.Contains(err.Error(), "unknown event type: custom:event") {
		t.Fatalf("error = %q, want unknown event type", err)
	}
	if !strings.Contains(err.Error(), "Valid types: "+ValidEventTypes()) {
		t.Fatalf("error = %q, want valid event list", err)
	}
	if got := stderr.String(); !strings.Contains(got, "Warning: invalid custom sound path") || !strings.Contains(got, "path is empty") {
		t.Fatalf("stderr = %q, want empty-path warning", got)
	}
}

func TestPlaySoundWithStrategyNonBuiltInUnplayableCustomWarnsThenReturnsUnknownEvent(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "bad.mp3")
	if err := os.WriteFile(customPath, []byte("not playable"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stderr := &bytes.Buffer{}
	var customCalls []string

	installAudioTestSeams(t, stderr,
		func(path string, _ float64, _ bool) error {
			customCalls = append(customCalls, path)
			return errors.New("decode failed")
		},
		func(string, float64, bool) error {
			t.Fatal("embedded fallback should not run for unknown event without default")
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"custom:event": {Paths: []string{customPath}},
	}

	err := PlaySoundWithStrategy("custom:event", "", false, c)
	if err == nil {
		t.Fatal("PlaySoundWithStrategy() error = nil, want unknown event error")
	}
	if len(customCalls) != 1 || customCalls[0] != customPath {
		t.Fatalf("custom calls = %v, want [%s]", customCalls, customPath)
	}
	if !strings.Contains(err.Error(), "unknown event type: custom:event") {
		t.Fatalf("error = %q, want unknown event type", err)
	}
	if got := stderr.String(); !strings.Contains(got, "Warning: failed to play custom sound") || !strings.Contains(got, "decode failed") {
		t.Fatalf("stderr = %q, want playback failure warning", got)
	}
}

func TestPlaySoundWithStrategyDirectoryCustomWarnsAndFallsBack(t *testing.T) {
	customDir := t.TempDir()
	stderr := &bytes.Buffer{}
	var embeddedCalls []string

	installAudioTestSeams(t, stderr,
		func(string, float64, bool) error {
			t.Fatal("custom player should not run for directory path")
			return nil
		},
		func(soundFile string, _ float64, _ bool) error {
			embeddedCalls = append(embeddedCalls, soundFile)
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"cli:start": {Paths: []string{customDir}},
	}

	if err := PlaySoundWithStrategy("cli:start", "", false, c); err != nil {
		t.Fatalf("PlaySoundWithStrategy() error = %v, want nil", err)
	}
	if len(embeddedCalls) != 1 || embeddedCalls[0] != "cli-start.mp3" {
		t.Fatalf("embedded calls = %v, want [cli-start.mp3]", embeddedCalls)
	}
	if got := stderr.String(); !strings.Contains(got, "path is a directory") {
		t.Fatalf("stderr = %q, want directory warning", got)
	}
}

func TestPlaySoundWithStrategyEmptyFileCustomWarnsAndFallsBack(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "empty.mp3")
	if err := os.WriteFile(customPath, nil, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stderr := &bytes.Buffer{}
	var embeddedCalls []string

	installAudioTestSeams(t, stderr,
		func(string, float64, bool) error {
			t.Fatal("custom player should not run for empty file")
			return nil
		},
		func(soundFile string, _ float64, _ bool) error {
			embeddedCalls = append(embeddedCalls, soundFile)
			return nil
		},
		nil,
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"cli:start": {Paths: []string{customPath}},
	}

	if err := PlaySoundWithStrategy("cli:start", "", false, c); err != nil {
		t.Fatalf("PlaySoundWithStrategy() error = %v, want nil", err)
	}
	if len(embeddedCalls) != 1 || embeddedCalls[0] != "cli-start.mp3" {
		t.Fatalf("embedded calls = %v, want [cli-start.mp3]", embeddedCalls)
	}
	if got := stderr.String(); !strings.Contains(got, "file is empty") {
		t.Fatalf("stderr = %q, want empty-file warning", got)
	}
}

func TestPlaySoundWithStrategyExpandsHomeForCustomPath(t *testing.T) {
	homeDir := t.TempDir()
	customPath := filepath.Join(homeDir, "sounds", "tilde.mp3")
	if err := os.MkdirAll(filepath.Dir(customPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(customPath, []byte("playable"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stderr := &bytes.Buffer{}
	var customCalls []string

	installAudioTestSeams(t, stderr,
		func(path string, _ float64, _ bool) error {
			customCalls = append(customCalls, path)
			return nil
		},
		func(string, float64, bool) error {
			t.Fatal("embedded fallback should not run when expanded custom path succeeds")
			return nil
		},
		func() (string, error) { return homeDir, nil },
	)

	c := unmutedConfig()
	c.Sounds = map[string]config.EventSoundConfig{
		"custom:event": {Paths: []string{"~/sounds/tilde.mp3"}},
	}

	if err := PlaySoundWithStrategy("custom:event", "", false, c); err != nil {
		t.Fatalf("PlaySoundWithStrategy() error = %v, want nil", err)
	}
	if len(customCalls) != 1 || customCalls[0] != customPath {
		t.Fatalf("custom calls = %v, want [%s]", customCalls, customPath)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
