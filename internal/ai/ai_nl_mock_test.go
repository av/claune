package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"os"
	"path/filepath"

	"github.com/everlier/claune/internal/config"
)

func TestHandleNaturalLanguageConfig_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ClaudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: "not json"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	err := HandleNaturalLanguageConfig("do something", &c)
	if err == nil {
		t.Fatalf("Expected error for invalid JSON response, got nil")
	}
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestAutoMapSounds_AIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := mockConfig(server.URL, t)
	_, err := AutoMapSounds(t.TempDir(), &c)
	if err == nil {
		t.Fatalf("Expected error for AI failure, got nil")
	}
}

func TestAutoMapSounds_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ClaudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: "not json"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "test.mp3"))
	if err == nil {
		file.Close()
	}

	c := mockConfig(server.URL, t)
	_, err = AutoMapSounds(dir, &c)
	if err == nil {
		t.Fatalf("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestAutoMapSounds_Fallback(t *testing.T) {
	c := config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: false,
		},
	}
	dir := t.TempDir()
	
	// create sad.mp3, yay.mp3, bomb.mp3, warn.mp3, build.mp3, test.mp3
	files := []string{"sad.mp3", "yay.mp3", "bomb.mp3", "warn.mp3", "build.mp3", "test.mp3"}
	for _, f := range files {
		file, _ := os.Create(filepath.Join(dir, f))
		file.Close()
	}

	mapping, err := AutoMapSounds(dir, &c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(mapping) == 0 {
		t.Fatalf("Expected mapping, got none")
	}
	
	expectedEvents := []string{"tool:error", "tool:success", "panic", "warn", "build:success", "test:fail"}
	for _, e := range expectedEvents {
		if _, ok := mapping[e]; !ok {
			t.Errorf("Expected event %s in mapping", e)
		}
	}
}

func TestAutoMapSounds_EmptyDir(t *testing.T) {
	c := config.ClauneConfig{
		AI: config.AIConfig{
			Enabled: false,
		},
	}
	dir := t.TempDir()
	
	_, err := AutoMapSounds(dir, &c)
	if err == nil {
		t.Fatalf("Expected error for empty dir, got nil")
	}
	if !strings.Contains(err.Error(), "no audio files found") {
		t.Errorf("Unexpected error message: %v", err)
	}
}
