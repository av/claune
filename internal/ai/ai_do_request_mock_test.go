package ai

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoAIRequest_Success(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, capture := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, req ClaudeRequest, _ *anthropicRequestCapture) any {
		if req.MaxTokens != 100 {
			t.Fatalf("request max tokens = %d, want 100", req.MaxTokens)
		}
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "success"}}}
	})
	defer server.Close()

	resp, err := doAIRequest(mockConfig(server.URL), ClaudeRequest{MaxTokens: 100})
	if err != nil {
		t.Fatalf("doAIRequest error = %v", err)
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "success" {
		t.Fatalf("response = %+v", resp)
	}
	if capture.Calls != 1 {
		t.Fatalf("calls = %d, want 1", capture.Calls)
	}
}

func TestDoAIRequest_401(t *testing.T) {
	_ = setupHermeticAITest(t)
	server, _ := createMockAnthropicServer(t, 401, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return map[string]string{"error": "unauthorized"}
	})
	defer server.Close()

	_, err := doAIRequest(mockConfig(server.URL), ClaudeRequest{MaxTokens: 100})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "AI API unauthorized (401). Please check your Anthropic API key with 'claune auth'.: {\"error\":\"unauthorized\"}\n" {
		t.Fatalf("error = %q", got)
	}
}

func TestDoAIRequest_429_RetryHonorsRetryAfterWithoutSleeping(t *testing.T) {
	env := setupHermeticAITest(t)
	server, capture := createMockAnthropicServer(t, 429, map[string]string{"Retry-After": "1"}, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return map[string]string{"error": "rate limit"}
	})
	defer server.Close()

	_, err := doAIRequest(mockConfig(server.URL), ClaudeRequest{MaxTokens: 100})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "AI API rate limit exceeded (429 Too Many Requests) after retries. Consider increasing your API rate limits or reducing concurrency." {
		t.Fatalf("error = %q", got)
	}
	if capture.Calls != 3 {
		t.Fatalf("calls = %d, want 3", capture.Calls)
	}
	if len(env.sleeps) != 2 || env.sleeps[0] != time.Second || env.sleeps[1] != time.Second {
		t.Fatalf("sleep durations = %+v, want [1s 1s]", env.sleeps)
	}
}

func TestDoAIRequest_EmptyResponse_RetriesThenFails(t *testing.T) {
	env := setupHermeticAITest(t)
	server, capture := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return map[string]any{"content": []any{}}
	})
	defer server.Close()

	_, err := doAIRequest(mockConfig(server.URL), ClaudeRequest{MaxTokens: 100})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "AI request failed after retries: empty AI response" {
		t.Fatalf("error = %q", got)
	}
	if capture.Calls != 3 {
		t.Fatalf("calls = %d, want 3", capture.Calls)
	}
	if want := []time.Duration{2 * time.Second, 4 * time.Second}; fmt.Sprint(env.sleeps) != fmt.Sprint(want) {
		t.Fatalf("sleep durations = %+v, want %+v", env.sleeps, want)
	}
}

func TestDoAIRequest_InvalidJSON(t *testing.T) {
	_ = setupHermeticAITest(t)
	capture := &anthropicRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.Calls++
		capture.Path = r.URL.Path
		capture.Method = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": [`))
	}))
	defer server.Close()

	_, err := doAIRequest(mockConfig(server.URL), ClaudeRequest{MaxTokens: 100})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "AI response parse failed: unexpected end of JSON input" {
		t.Fatalf("error = %q", got)
	}
	if capture.Calls != 1 {
		t.Fatalf("calls = %d, want 1", capture.Calls)
	}
}

func TestDoAIRequest_MaxTokensRetriesWithExpandedBudget(t *testing.T) {
	env := setupHermeticAITest(t)
	call := 0
	server, capture := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, req ClaudeRequest, _ *anthropicRequestCapture) any {
		call++
		if call == 1 {
			if req.MaxTokens != 10 {
				t.Fatalf("first attempt max tokens = %d, want 10", req.MaxTokens)
			}
			return map[string]any{"content": []map[string]string{{"text": "partial"}}, "stop_reason": "max_tokens"}
		}
		if req.MaxTokens != 20 {
			t.Fatalf("second attempt max tokens = %d, want 20", req.MaxTokens)
		}
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "done"}}}
	})
	defer server.Close()

	resp, err := doAIRequest(mockConfig(server.URL), ClaudeRequest{MaxTokens: 10})
	if err != nil {
		t.Fatalf("doAIRequest error = %v", err)
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "done" {
		t.Fatalf("response = %+v", resp)
	}
	if capture.Calls != 2 {
		t.Fatalf("calls = %d, want 2", capture.Calls)
	}
	if len(env.sleeps) != 1 || env.sleeps[0] != 2*time.Second {
		t.Fatalf("sleep durations = %+v, want [2s]", env.sleeps)
	}
}

func TestDoAIRequest_UsesEnvironmentAPIKeyWhenConfigKeyMissing(t *testing.T) {
	_ = setupHermeticAITest(t)
	t.Setenv("ANTHROPIC_API_KEY", "env-test-key")
	server, capture := createMockAnthropicServer(t, 200, nil, func(_ *http.Request, _ ClaudeRequest, _ *anthropicRequestCapture) any {
		return ClaudeResponse{Content: []struct {
			Text string `json:"text"`
		}{{Text: "success"}}}
	})
	defer server.Close()

	c := mockConfig(server.URL)
	c.AI.APIKey = ""
	if _, err := doAIRequest(c, ClaudeRequest{MaxTokens: 1}); err != nil {
		t.Fatalf("doAIRequest error = %v", err)
	}
	if capture.APIKey != "env-test-key" {
		t.Fatalf("x-api-key = %q, want env-test-key", capture.APIKey)
	}
}

func TestDoAIRequest_RedactsSecretsInNonRetryableErrors(t *testing.T) {
	_ = setupHermeticAITest(t)
	secret := `{"api_key":"sk-ant-api03-secretvalue1234567890"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(403)
		_, _ = w.Write([]byte(secret))
	}))
	defer server.Close()

	_, err := doAIRequest(mockConfig(server.URL), ClaudeRequest{MaxTokens: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got == "" || strings.Contains(got, "secretvalue1234567890") || !containsAll(got, "403", "[REDACTED_ANTHROPIC_KEY]") {
		t.Fatalf("error = %q", got)
	}
}
