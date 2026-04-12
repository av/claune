package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

)

func TestDoAIRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ClaudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: "success"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	req := ClaudeRequest{MaxTokens: 100}
	resp, err := doAIRequest(c, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(resp.Content) == 0 || resp.Content[0].Text != "success" {
		t.Fatalf("Unexpected response: %+v", resp)
	}
}

func TestDoAIRequest_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	req := ClaudeRequest{MaxTokens: 100}
	_, err := doAIRequest(c, req)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("Expected 401 error message, got: %v", err)
	}
}

func TestDoAIRequest_429_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limit"}`))
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	req := ClaudeRequest{MaxTokens: 100}

	start := time.Now()
	_, err := doAIRequest(c, req)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected error, got none")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("Expected 429 error message, got: %v", err)
	}
	// should retry twice (total 3 attempts)
	if attempts != 3 {
		t.Fatalf("Expected 3 attempts, got %d", attempts)
	}
	// sleep duration logic is 1<<1 = 2s for attempt 1, 1<<2 = 4s for attempt 2 if Retry-After is missing, but with Retry-After=1, it sleeps 1s each time.
	// total sleep >= 2s.
	if duration < 2*time.Second {
		t.Fatalf("Expected duration >= 2s, got %v", duration)
	}
}

func TestDoAIRequest_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// valid json but empty content array
		w.Write([]byte(`{"content": []}`))
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	req := ClaudeRequest{MaxTokens: 100}
	_, err := doAIRequest(c, req)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	if !strings.Contains(err.Error(), "empty AI response") && !strings.Contains(err.Error(), "failed after retries") {
		t.Fatalf("Expected empty AI response error, got: %v", err)
	}
}

func TestDoAIRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": [`)) // broken JSON
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	req := ClaudeRequest{MaxTokens: 100}
	_, err := doAIRequest(c, req)
	if err == nil {
		t.Fatal("Expected error, got none")
	}
	if !strings.Contains(err.Error(), "AI response parse failed") {
		t.Fatalf("Expected parse error, got: %v", err)
	}
}
