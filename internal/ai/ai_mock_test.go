package ai

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/everlier/claune/internal/config"
)

func TestAnalyzeToolIntent_Mock(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, capture := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, req ClaudeRequest, _ *anthropicRequestCapture) any {
		if req.Model != "claude-3-opus-test" {
			t.Fatalf("request model = %q", req.Model)
		}
		if req.MaxTokens != 10 {
			t.Fatalf("request max tokens = %d, want 10", req.MaxTokens)
		}
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "destructive"}}}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	intent, err := AnalyzeToolIntent("tool:success", "bash", "rm -rf /", c)
	if err != nil {
		t.Fatalf("AnalyzeToolIntent failed: %v", err)
	}
	if intent != "tool:destructive" {
		t.Fatalf("intent = %q, want %q", intent, "tool:destructive")
	}
	if capture.APIKey != "test-key" {
		t.Fatalf("x-api-key = %q, want test key", capture.APIKey)
	}
	if capture.Version != "2023-06-01" {
		t.Fatalf("anthropic-version = %q", capture.Version)
	}
	if capture.ContentType != "application/json" {
		t.Fatalf("content-type = %q", capture.ContentType)
	}
}

func TestAnalyzeResponseSentiment_Mock(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "SUCCESS"}}}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	sentiment, strategy, err := AnalyzeResponseSentiment("Great job!", c)
	if err != nil {
		t.Fatalf("AnalyzeResponseSentiment failed: %v", err)
	}
	if sentiment != "tool:success" || strategy != "random" {
		t.Fatalf("got (%q, %q), want (%q, %q)", sentiment, strategy, "tool:success", "random")
	}
}

func TestHandleNaturalLanguageConfig_Mock(t *testing.T) {
	env := setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: `{"sounds":{"startup":{"paths":["/path/to/startup.mp3"]}}}`}}}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	c.Sounds = make(map[string]config.EventSoundConfig)
	if err := HandleNaturalLanguageConfig("Change startup sound to /path/to/startup.mp3", &c); err != nil {
		t.Fatalf("HandleNaturalLanguageConfig failed: %v", err)
	}

	sound, ok := c.Sounds["startup"]
	if !ok || len(sound.Paths) != 1 || sound.Paths[0] != "/path/to/startup.mp3" {
		t.Fatalf("startup sound = %+v", sound)
	}
	saved := env.loadSavedConfig(t)
	if got := saved.Sounds["startup"].Paths; len(got) != 1 || got[0] != "/path/to/startup.mp3" {
		t.Fatalf("saved startup sound = %+v", saved.Sounds["startup"])
	}
	env.assertNoHostSideEffects(t)
}

func TestAutoMapSounds_Mock(t *testing.T) {
	env := setupHermeticAITest(t)
	dir := t.TempDir()
	filePath := filepath.Join(dir, "victory.mp3")
	if err := osWriteEmptyFile(filePath); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	server, capture := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, req ClaudeRequest, _ *anthropicRequestCapture) any {
		if len(req.Messages) != 1 || !containsAll(req.Messages[0].Content, dir, "victory.mp3") {
			t.Fatalf("prompt missing expected file context: %q", req.Messages[0].Content)
		}
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: fmt.Sprintf(`{"tool:success":{"paths":[%q],"strategy":"random"}}`, filePath)}}}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	mapping, err := AutoMapSounds(dir, &c)
	if err != nil {
		t.Fatalf("AutoMapSounds failed: %v", err)
	}
	if capture.Calls != 1 {
		t.Fatalf("AI calls = %d, want 1", capture.Calls)
	}
	event, ok := mapping["tool:success"]
	if !ok || len(event.Paths) != 1 || event.Paths[0] != filePath || event.Strategy != "random" {
		t.Fatalf("mapping = %+v", mapping)
	}
	saved := env.loadSavedConfig(t)
	if got := saved.Sounds["tool:success"]; len(got.Paths) != 1 || got.Paths[0] != filePath {
		t.Fatalf("saved mapping = %+v", got)
	}
	env.assertNoHostSideEffects(t)
}

func TestDiagnoseInstallFailure_Mock(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "Check your permissions"}}}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	diagnosis := DiagnoseInstallFailure(fmt.Errorf("test err"), c)
	if diagnosis != "Check your permissions" {
		t.Fatalf("diagnosis = %q, want %q", diagnosis, "Check your permissions")
	}
}

func TestGuessEventForSound_Mock(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, capture := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, req ClaudeRequest, _ *anthropicRequestCapture) any {
		if len(req.Messages) != 1 || !containsAll(req.Messages[0].Content, "startup.mp3", "http://example.com") {
			t.Fatalf("prompt missing url/filename context: %q", req.Messages[0].Content)
		}
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "tool:start"}}}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	event, err := GuessEventForSound("http://example.com", "startup.mp3", c)
	if err != nil {
		t.Fatalf("GuessEventForSound failed: %v", err)
	}
	if event != "tool:start" {
		t.Fatalf("event = %q, want %q", event, "tool:start")
	}
	if capture.Calls != 1 {
		t.Fatalf("AI calls = %d, want 1", capture.Calls)
	}
}

func osWriteEmptyFile(path string) error {
	return os.WriteFile(path, nil, 0o644)
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
