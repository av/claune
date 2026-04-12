package ai

import (
	"net/http"
	"testing"

	"github.com/everlier/claune/internal/config"
)

func TestHandleNaturalLanguageConfig_InitializesNilSoundsWithoutPanic(t *testing.T) {
	env := setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return map[string]any{"content": []map[string]string{{"text": `{"sounds":{"cli:start":{"paths":["/tmp/a.mp3"]}}}`}}}
	})
	defer server.Close()

	c := &config.ClauneConfig{AI: config.AIConfig{Enabled: true, APIKey: "test", APIURL: server.URL}}
	if err := HandleNaturalLanguageConfig("add a sound", c); err != nil {
		t.Fatalf("HandleNaturalLanguageConfig error = %v", err)
	}
	if c.Sounds == nil {
		t.Fatal("sounds map is nil")
	}
	if got := c.Sounds["cli:start"]; len(got.Paths) != 1 || got.Paths[0] != "/tmp/a.mp3" {
		t.Fatalf("cli:start sound = %+v", got)
	}
	saved := env.loadSavedConfig(t)
	if got := saved.Sounds["cli:start"]; len(got.Paths) != 1 || got.Paths[0] != "/tmp/a.mp3" {
		t.Fatalf("saved cli:start sound = %+v", got)
	}
	env.assertNoHostSideEffects(t)
}
