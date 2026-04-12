package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/everlier/claune/internal/config"
)

func TestGuessEventForSound_Fallback(t *testing.T) {
	c := config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: false,
		},
	}
	
	event, err := GuessEventForSound("http://example.com/sad.mp3", "sad.mp3", c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if event != "tool:error" {
		t.Errorf("Expected tool:error, got %s", event)
	}
	
	event, err = GuessEventForSound("http://example.com/yay.mp3", "yay.mp3", c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if event != "tool:success" {
		t.Errorf("Expected tool:success, got %s", event)
	}

	event, err = GuessEventForSound("http://example.com/other.mp3", "other.mp3", c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if event != "tool:start" {
		t.Errorf("Expected tool:start, got %s", event)
	}
}

func TestGuessEventForSound_AI_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ClaudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: "not_a_valid_event"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	_, err := GuessEventForSound("http://example.com/test.mp3", "test.mp3", c)
	if err == nil {
		t.Fatalf("Expected error for invalid event response, got nil")
	}
	if !strings.Contains(err.Error(), "did not contain a valid event") {
		t.Errorf("Unexpected error message: %v", err)
	}
}
