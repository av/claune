package ai

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/everlier/claune/internal/config"
)

func TestHandleNaturalLanguageConfig_InvalidJSON(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "not json"}}}
	})
	defer server.Close()

	err := HandleNaturalLanguageConfig("do something", &config.ClauneConfig{AI: mockConfig(server.URL).AI})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "config schema validation failed") || !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestHandleNaturalLanguageConfig_FallbackPersistsOnlyInTempRoots(t *testing.T) {
	env := setupHermeticAITest(t)
	c := config.ClauneConfig{AI: config.AIConfig{Enabled: false}}
	if err := HandleNaturalLanguageConfig("set volume to 50% and unmute", &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Volume == nil || *c.Volume != 0.5 {
		t.Fatalf("volume = %+v, want 0.5", c.Volume)
	}
	if c.Mute == nil || *c.Mute {
		t.Fatalf("mute = %+v, want false", c.Mute)
	}
	if c.MuteUntil != nil {
		t.Fatalf("muteUntil = %v, want nil", c.MuteUntil)
	}
	saved := env.loadSavedConfig(t)
	if saved.Volume == nil || *saved.Volume != 0.5 {
		t.Fatalf("saved volume = %+v", saved.Volume)
	}
	env.assertNoHostSideEffects(t)
}

func TestAutoMapSounds_AIErrorFallsBackAndPersistsTempConfig(t *testing.T) {
	env := setupHermeticAITest(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "yay.mp3"), nil, 0o644); err != nil {
		t.Fatalf("write test sound: %v", err)
	}

	server, capture := createMockAnthropicServer(t, 500, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return map[string]string{"error": "boom"}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	mapping, err := AutoMapSounds(dir, &c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capture.Calls != 3 {
		t.Fatalf("calls = %d, want 3 retry attempts", capture.Calls)
	}
	if got := mapping["tool:success"]; len(got.Paths) != 1 || filepath.Base(got.Paths[0]) != "yay.mp3" {
		t.Fatalf("mapping = %+v", mapping)
	}
	saved := env.loadSavedConfig(t)
	if got := saved.Sounds["tool:success"]; len(got.Paths) != 1 || filepath.Base(got.Paths[0]) != "yay.mp3" {
		t.Fatalf("saved config = %+v", saved.Sounds)
	}
	env.assertNoHostSideEffects(t)
}

func TestAutoMapSounds_InvalidJSON(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "not json"}}}
	})
	defer server.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.mp3"), nil, 0o644); err != nil {
		t.Fatalf("write test sound: %v", err)
	}

	c := mockConfig(server.URL)
	_, err := AutoMapSounds(dir, &c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse AI response") || !strings.Contains(err.Error(), "invalid character") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestAutoMapSounds_Fallback(t *testing.T) {
	env := setupHermeticAITest(t)
	c := config.ClauneConfig{AI: config.AIConfig{Enabled: false}}
	dir := t.TempDir()
	for _, name := range []string{"sad.mp3", "yay.mp3", "bomb.mp3", "warn.mp3", "build.mp3", "test.mp3"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	mapping, err := AutoMapSounds(dir, &c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, event := range []string{"tool:error", "tool:success", "panic", "warn", "build:success", "test:fail"} {
		if _, ok := mapping[event]; !ok {
			t.Fatalf("missing event %q in mapping %+v", event, mapping)
		}
	}
	env.assertNoHostSideEffects(t)
}

func TestAutoMapSounds_EmptyDir(t *testing.T) {
	_ = setupHermeticAITest(t)
	_, err := AutoMapSounds(t.TempDir(), &config.ClauneConfig{AI: config.AIConfig{Enabled: false}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no audio files found") {
		t.Fatalf("error = %q", err.Error())
	}
}
