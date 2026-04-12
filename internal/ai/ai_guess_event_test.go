package ai

import (
	"net/http"
	"testing"

	"github.com/everlier/claune/internal/config"
)

func TestGuessEventForSound_Fallback(t *testing.T) {
	_ = setupHermeticAITest(t)
	c := config.ClauneConfig{AI: config.AIConfig{Enabled: false}}

	for _, tc := range []struct {
		filename string
		want     string
	}{
		{filename: "sad.mp3", want: "tool:error"},
		{filename: "yay.mp3", want: "tool:success"},
		{filename: "other.mp3", want: "tool:start"},
	} {
		event, err := GuessEventForSound("http://example.com/"+tc.filename, tc.filename, c)
		if err != nil {
			t.Fatalf("GuessEventForSound(%q) error = %v", tc.filename, err)
		}
		if event != tc.want {
			t.Fatalf("GuessEventForSound(%q) = %q, want %q", tc.filename, event, tc.want)
		}
	}
}

func TestGuessEventForSound_AI_InvalidResponse(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "not_a_valid_event"}}}
	})
	defer server.Close()

	_, err := GuessEventForSound("http://example.com/test.mp3", "test.mp3", mockConfig(server.URL))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "AI response did not contain a valid event: not_a_valid_event" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestGuessEventForSound_AIFailureFallsBackDeterministically(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, capture := createMockAnthropicServer(t, 500, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return map[string]string{"error": "upstream down"}
	})
	defer server.Close()

	event, err := GuessEventForSound("http://example.com/yay.mp3", "yay.mp3", mockConfig(server.URL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != "tool:success" {
		t.Fatalf("event = %q, want %q", event, "tool:success")
	}
	if capture.Calls != 3 {
		t.Fatalf("calls = %d, want 3 retry attempts", capture.Calls)
	}
}
