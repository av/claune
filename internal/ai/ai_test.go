package ai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/everlier/claune/internal/config"
)

func TestAnalyze(t *testing.T) {
	c := config.ClauneConfig{}
	res, _ := AnalyzeToolIntent("bash", "rm -rf /", c)
	if res != "tool:start" {
		t.Error("Expected tool:start fallback when disabled")
	}
}

func TestAnalyzeResponseSentimentUsesConfiguredAPIURLAndReturnsNon200Error(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		http.Error(w, "upstream exploded", http.StatusBadGateway)
	}))
	defer server.Close()

	event, strategy, err := AnalyzeResponseSentiment("please analyze this", config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: true,
			APIKey:  "test-key",
			APIURL:  server.URL,
		},
	})
	if err == nil {
		t.Fatal("AnalyzeResponseSentiment error = nil, want non-200 error")
	}
	if event != "" {
		t.Fatalf("AnalyzeResponseSentiment event = %q, want empty", event)
	}
	if strategy != "" {
		t.Fatalf("AnalyzeResponseSentiment strategy = %q, want empty", strategy)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("request path = %q, want %q", gotPath, "/v1/messages")
	}
	if !strings.Contains(err.Error(), "AI API returned status 502") {
		t.Fatalf("error = %q, want status detail", err.Error())
	}
	if !strings.Contains(err.Error(), "upstream exploded") {
		t.Fatalf("error = %q, want response body", err.Error())
	}
}
